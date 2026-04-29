package processor

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/jiaozhu/emorad/internal/cfr"
	"github.com/jiaozhu/emorad/internal/report"
)

// 默认排除的框架包前缀
var DefaultExcludes = []string{
	"org/springframework/",
	"org/apache/",
	"com/fasterxml/",
	"org/hibernate/",
	"org/mybatis/",
	"ch/qos/logback/",
	"org/slf4j/",
	"com/google/",
	"javax/",
	"jakarta/",
	"org/aspectj/",
	"org/yaml/",
	"com/zaxxer/",
	"org/jboss/",
	"io/netty/",
	"com/alibaba/",
	"org/thymeleaf/",
	"org/bouncycastle/",
}

// FilterConfig 过滤配置
type FilterConfig struct {
	Includes          []string // 包含的包前缀（优先级最高）
	Excludes          []string // 排除的包前缀
	SkipLibs          bool     // 是否跳过 lib 目录下的 JAR
	JarIncludes       []string // JAR 名称匹配关键字
	JarMatchMode      string   // JAR匹配模式: contains/prefix
	NestedJarStrategy string   // 嵌套JAR策略: skip/filtered/full
	MaxJarDepth       int      // 嵌套JAR最大递归深度
	LogFormat         string   // 日志格式: text/json
	CopyResources     bool     // 是否复制配置文件到输出目录
	CopyLibJars       bool     // 是否复制依赖 JAR 到 libs 目录
	GenerateIDEA      bool     // 是否生成 IDEA 项目配置
	UnicodePostprocess bool    // 是否执行 Unicode 后处理
	FailFast          bool     // 遇到错误是否立即失败
	EventLogger       func(level string, event string, kv map[string]any)
	AppJarHints       []string // 业务JAR识别提示词（来自应用名/目录名）
	CommonJarPrefixes []string // 公共依赖JAR前缀（默认过滤）
}

// NewDefaultFilterConfig 创建默认过滤配置
func NewDefaultFilterConfig() *FilterConfig {
	return &FilterConfig{
		Includes:          nil,
		Excludes:          DefaultExcludes,
		SkipLibs:          true,
		JarMatchMode:      "contains",
		NestedJarStrategy: "filtered",
		MaxJarDepth:       8,
		LogFormat:         "text",
		UnicodePostprocess: true,
		FailFast:          false,
		CommonJarPrefixes: []string{
			"spring-", "springframework-",
			"commons-", "apache-",
			"jackson-", "fastjson", "gson",
			"log4j", "slf4j", "logback",
			"mybatis", "hibernate",
			"tomcat-", "servlet-", "jsp-", "jstl",
			"javax.", "jakarta.",
			"guava", "netty", "cglib", "asm-", "antlr-",
			"byte-buddy", "kotlin-", "scala-",
			"xml-", "stax-", "xerces", "xalan",
			"validation-", "jaxb-", "jaxws-", "saaj-",
		},
	}
}

// ShouldProcessClass 判断是否应该处理该 class 文件
// baseDir 是解压后的临时目录
func (f *FilterConfig) ShouldProcessClass(classPath, baseDir string) bool {
	relativePath := extractRelativePathFromBase(classPath, baseDir)

	if len(f.Includes) > 0 {
		for _, include := range f.Includes {
			if strings.HasPrefix(relativePath, include) {
				return true
			}
		}
		return false
	}

	for _, exclude := range f.Excludes {
		if strings.HasPrefix(relativePath, exclude) {
			return false
		}
	}

	return true
}

func (f *FilterConfig) LogEvent(level string, event string, kv map[string]any) {
	if f == nil || f.EventLogger == nil {
		return
	}
	f.EventLogger(level, event, kv)
}

func (f *FilterConfig) jarNameMatched(jarName string) bool {
	if len(f.JarIncludes) == 0 {
		return true
	}
	name := strings.ToLower(jarName)
	mode := strings.ToLower(f.JarMatchMode)
	for _, keyword := range f.JarIncludes {
		k := strings.ToLower(keyword)
		if mode == "prefix" {
			if strings.HasPrefix(name, k) {
				return true
			}
			continue
		}
		if strings.Contains(name, k) {
			return true
		}
	}
	return false
}

func normalizeArchivePath(path string) string {
	return strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
}

func isLibJarPath(path string) bool {
	p := normalizeArchivePath(path)
	return strings.Contains(p, "BOOT-INF/lib/") ||
		strings.Contains(p, "WEB-INF/lib/") ||
		strings.Contains(p, "APP-INF/lib/")
}

func isClassesPath(path string) bool {
	p := normalizeArchivePath(path)
	return strings.Contains(p, "BOOT-INF/classes/") ||
		strings.Contains(p, "WEB-INF/classes/") ||
		strings.Contains(p, "APP-INF/classes/")
}

func (f *FilterConfig) isCommonJar(jarName string) bool {
	name := strings.ToLower(jarName)
	for _, prefix := range f.CommonJarPrefixes {
		p := strings.ToLower(strings.TrimSpace(prefix))
		if p != "" && strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func (f *FilterConfig) isLikelyBusinessJar(jarName string) bool {
	name := strings.ToLower(strings.TrimSuffix(jarName, ".jar"))
	for _, hint := range f.AppJarHints {
		h := strings.ToLower(strings.TrimSpace(hint))
		if len(h) < 3 {
			continue
		}
		if strings.Contains(name, h) {
			return true
		}
	}
	return false
}

// ShouldProcessJar 判断是否应该处理该 JAR 文件
func (f *FilterConfig) ShouldProcessJar(jarPath string) bool {
	isLibJar := isLibJarPath(jarPath)

	if isLibJar {
		jarName := filepath.Base(jarPath)
		if len(f.JarIncludes) > 0 {
			return f.jarNameMatched(jarName)
		}
		if f.SkipLibs {
			// 智能模式：过滤公共依赖，保留疑似业务JAR
			if f.isCommonJar(jarName) {
				return false
			}
			return f.isLikelyBusinessJar(jarName)
		}
	}
	return true
}

// extractRelativePathFromBase 从 class 路径中提取相对包路径
// baseDir 是解压后的临时目录，classPath 是 class 文件的完整路径
func extractRelativePathFromBase(classPath, baseDir string) string {
	// 先计算相对于临时目录的路径
	relPath, err := filepath.Rel(baseDir, classPath)
	if err != nil {
		relPath = classPath
	}
	// 转换为正斜杠形式
	relPath = filepath.ToSlash(relPath)

	// 如果是 BOOT-INF/classes 或 WEB-INF/classes 下的类，提取真正的包路径
	if idx := strings.Index(relPath, "BOOT-INF/classes/"); idx != -1 {
		return relPath[idx+len("BOOT-INF/classes/"):]
	}
	if idx := strings.Index(relPath, "WEB-INF/classes/"); idx != -1 {
		return relPath[idx+len("WEB-INF/classes/"):]
	}
	// 对于 JAR 根目录下的类（如 org/springframework/boot/loader/），
	// 直接返回相对路径
	return relPath
}

// Processor 定义文件处理器接口
type Processor interface {
	Process(ctx context.Context, inputPath string, outputDir string, rpt *report.Report) error
	GetType() string
}

// ClassProcessor 处理单个.class文件
type ClassProcessor struct {
	cfrManager *cfr.Manager
	textMode   bool
}

func NewClassProcessor(cfrManager *cfr.Manager, textMode bool) *ClassProcessor {
	return &ClassProcessor{cfrManager: cfrManager, textMode: textMode}
}

func (p *ClassProcessor) GetType() string {
	return "class"
}

func (p *ClassProcessor) Process(ctx context.Context, inputPath string, outputDir string, rpt *report.Report) error {
	startTime := time.Now()
	result := report.Result{
		ClassName:   filepath.Base(inputPath),
		PackageName: ExtractPackageName(inputPath),
		FilePath:    inputPath,
		Stage:       "decompile",
		Success:     false,
		TimeStamp:   startTime,
	}

	err := p.cfrManager.Decompile(ctx, inputPath, outputDir)
	if err != nil {
		result.Success = false
		result.ErrorCode = "DECOMPILE_FAILED"
		result.Error = fmt.Sprintf("反编译失败: %v", err)
	} else {
		result.Success = true
	}

	result.TimeTaken = time.Since(startTime).Seconds()
	rpt.AddResult(result)
	return err
}

// JarProcessor 处理JAR文件
type JarProcessor struct {
	cfrManager   *cfr.Manager
	workers      int
	filterConfig *FilterConfig
}

func (p *JarProcessor) textMode() bool {
	return p.filterConfig == nil || p.filterConfig.LogFormat == "text"
}

func NewJarProcessor(cfrManager *cfr.Manager, workers int, filterConfig *FilterConfig) *JarProcessor {
	return &JarProcessor{
		cfrManager:   cfrManager,
		workers:      workers,
		filterConfig: filterConfig,
	}
}

func (p *JarProcessor) GetType() string {
	return "jar"
}

func (p *JarProcessor) Process(ctx context.Context, inputPath string, outputDir string, rpt *report.Report) error {
	return p.processJar(ctx, inputPath, outputDir, rpt, 0)
}

func (p *JarProcessor) shouldProcessNestedJar(jarPath string) bool {
	switch strings.ToLower(p.filterConfig.NestedJarStrategy) {
	case "skip":
		return false
	case "full":
		return true
	default: // filtered
		return p.filterConfig.ShouldProcessJar(jarPath)
	}
}

func (p *JarProcessor) processJar(ctx context.Context, inputPath string, outputDir string, rpt *report.Report, depth int) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("emorad-%s-%d",
		filepath.Base(inputPath), time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %v", err)
	}

	if err := UnzipFile(inputPath, tempDir); err != nil {
		return fmt.Errorf("解压JAR文件失败: %v", err)
	}

	classFiles, nestedJars, resourceFiles, err := ScanDirectory(tempDir)
	if err != nil {
		return fmt.Errorf("扫描目录失败: %v", err)
	}

	if p.filterConfig.CopyResources && len(resourceFiles) > 0 {
		copiedCount := 0
		for _, resFile := range resourceFiles {
			if err := CopyResourceFile(resFile, tempDir, outputDir); err != nil {
				if p.textMode() {
					color.Red("复制配置文件失败: %s - %v", filepath.Base(resFile), err)
				}
			} else {
				copiedCount++
			}
		}
		if copiedCount > 0 {
			if p.textMode() {
				color.Green("[OK] 复制了 %d 个配置文件", copiedCount)
			}
			p.filterConfig.LogEvent("info", "resources_copied", map[string]any{
				"count": copiedCount,
			})
		}
	}

	filteredClasses := make([]string, 0, len(classFiles))
	for _, classPath := range classFiles {
		if p.filterConfig.ShouldProcessClass(classPath, tempDir) {
			filteredClasses = append(filteredClasses, classPath)
		}
	}

	if len(classFiles) != len(filteredClasses) {
		if p.textMode() {
			color.Yellow("[FILTER] 过滤后: %d/%d 个 class 文件需要处理", len(filteredClasses), len(classFiles))
		}
		p.filterConfig.LogEvent("info", "class_filter_applied", map[string]any{
			"before": len(classFiles),
			"after":  len(filteredClasses),
		})
	}

	rpt.AddExpectedFiles(int32(len(filteredClasses)))

	// 复制依赖 JAR 到 libs 目录
	if p.filterConfig.CopyLibJars && len(nestedJars) > 0 {
		copiedJars, err := CopyLibJars(nestedJars, outputDir)
		if err != nil {
			if p.textMode() {
				color.Yellow("[WARN] 复制依赖 JAR 失败: %v", err)
			}
		} else if copiedJars > 0 {
			if p.textMode() {
				color.Green("[OK] 复制了 %d 个依赖 JAR 到 libs 目录", copiedJars)
			}
			p.filterConfig.LogEvent("info", "lib_jars_copied", map[string]any{
				"count": copiedJars,
			})
		}
	}

	rpt.AddNestedStats(int32(len(nestedJars)), 0, 0)

	if depth >= p.filterConfig.MaxJarDepth {
		if len(nestedJars) > 0 {
			if p.textMode() {
				color.Yellow("[FILTER] 已达最大嵌套深度 %d，跳过 %d 个嵌套JAR", p.filterConfig.MaxJarDepth, len(nestedJars))
			}
			p.filterConfig.LogEvent("warn", "nested_jar_depth_limit_reached", map[string]any{
				"max_depth": p.filterConfig.MaxJarDepth,
				"skipped":   len(nestedJars),
			})
			rpt.AddNestedStats(0, 0, int32(len(nestedJars)))
		}
		return p.processClassFiles(ctx, filteredClasses, outputDir, rpt)
	}

	var nestedErrs []error
	for _, nestedJar := range nestedJars {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !p.shouldProcessNestedJar(nestedJar) {
			rpt.AddNestedStats(0, 0, 1)
			p.filterConfig.LogEvent("info", "nested_jar_skipped", map[string]any{
				"jar":   filepath.Base(nestedJar),
				"depth": depth + 1,
			})
			continue
		}
		rpt.AddNestedStats(0, 1, 0)
		nestedProcessor := NewJarProcessor(p.cfrManager, p.workers, p.filterConfig)
		if err := nestedProcessor.processJar(ctx, nestedJar, outputDir, rpt, depth+1); err != nil {
			if p.textMode() {
				color.Red("处理嵌套JAR失败: %v", err)
			}
			if p.filterConfig.FailFast {
				return err
			}
			p.filterConfig.LogEvent("error", "nested_jar_failed", map[string]any{
				"jar":   filepath.Base(nestedJar),
				"depth": depth + 1,
				"error": err.Error(),
			})
			nestedErrs = append(nestedErrs, fmt.Errorf("%s: %w", filepath.Base(nestedJar), err))
		}
	}

	classErr := p.processClassFiles(ctx, filteredClasses, outputDir, rpt)
	if classErr != nil && p.filterConfig.FailFast {
		return classErr
	}
	if classErr != nil {
		nestedErrs = append(nestedErrs, classErr)
	}
	if len(nestedErrs) > 0 {
		return errors.Join(nestedErrs...)
	}
	return nil
}

func (p *JarProcessor) processClassFiles(ctx context.Context, classFiles []string, outputDir string, rpt *report.Report) error {
	jobs := make(chan string, len(classFiles))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processor := NewClassProcessor(p.cfrManager, p.textMode())
			for classPath := range jobs {
				if ctx.Err() != nil {
					return
				}
				if err := processor.Process(ctx, classPath, outputDir, rpt); err != nil {
					p.filterConfig.LogEvent("error", "class_decompile_failed", map[string]any{
						"class_file": filepath.Base(classPath),
						"error":      err.Error(),
					})
					mu.Lock()
					errs = append(errs, fmt.Errorf("%s: %w", filepath.Base(classPath), err))
					mu.Unlock()
				}
			}
		}()
	}

	for _, file := range classFiles {
		if ctx.Err() != nil {
			break
		}
		jobs <- file
	}
	close(jobs)

	wg.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// WarProcessor 处理WAR文件
type WarProcessor struct {
	*JarProcessor
}

func NewWarProcessor(cfrManager *cfr.Manager, workers int, filterConfig *FilterConfig) *WarProcessor {
	return &WarProcessor{
		JarProcessor: NewJarProcessor(cfrManager, workers, filterConfig),
	}
}

func (p *WarProcessor) GetType() string {
	return "war"
}

// DirectoryProcessor 处理目录
type DirectoryProcessor struct {
	cfrManager   *cfr.Manager
	workers      int
	filterConfig *FilterConfig
}

func (p *DirectoryProcessor) textMode() bool {
	return p.filterConfig == nil || p.filterConfig.LogFormat == "text"
}

func NewDirectoryProcessor(cfrManager *cfr.Manager, workers int, filterConfig *FilterConfig) *DirectoryProcessor {
	return &DirectoryProcessor{
		cfrManager:   cfrManager,
		workers:      workers,
		filterConfig: filterConfig,
	}
}

func (p *DirectoryProcessor) GetType() string {
	return "directory"
}

func (p *DirectoryProcessor) Process(ctx context.Context, inputPath string, outputDir string, rpt *report.Report) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if p.textMode() {
		color.Cyan("正在处理目录: %s", inputPath)
	}

	classFiles, jarFiles, warFiles, err := ScanDirectoryComplete(inputPath, outputDir)
	if err != nil {
		return fmt.Errorf("扫描目录失败: %v", err)
	}

	filteredJars := make([]string, 0, len(jarFiles))
	for _, jarPath := range jarFiles {
		if p.filterConfig.ShouldProcessJar(jarPath) {
			filteredJars = append(filteredJars, jarPath)
		}
	}

	filteredClasses := make([]string, 0, len(classFiles))
	for _, classPath := range classFiles {
		if p.filterConfig.ShouldProcessClass(classPath, inputPath) {
			filteredClasses = append(filteredClasses, classPath)
		}
	}

	p.filterConfig.LogEvent("info", "directory_scan_completed", map[string]any{
		"jar_count":   len(filteredJars),
		"war_count":   len(warFiles),
		"class_count": len(filteredClasses),
	})

	if p.textMode() {
		color.Cyan("[SCAN] 扫描结果: %d个JAR, %d个WAR, %d个CLASS文件",
			len(filteredJars), len(warFiles), len(filteredClasses))
	}

	if len(filteredJars) == 0 && len(warFiles) == 0 && len(filteredClasses) == 0 {
		if p.textMode() {
			color.Yellow("[WARN] 未找到任何需要反编译的文件")
		}
		return nil
	}

	rpt.AddExpectedFiles(int32(len(filteredClasses)))

	var procErrs []error
	for _, jarPath := range filteredJars {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		jarProcessor := NewJarProcessor(p.cfrManager, p.workers, p.filterConfig)
		if err := jarProcessor.Process(ctx, jarPath, outputDir, rpt); err != nil {
			if p.textMode() {
				color.Red("处理JAR失败: %v", err)
			}
			if p.filterConfig.FailFast {
				return err
			}
			p.filterConfig.LogEvent("error", "jar_process_failed", map[string]any{
				"jar":   filepath.Base(jarPath),
				"error": err.Error(),
			})
			procErrs = append(procErrs, fmt.Errorf("jar %s: %w", filepath.Base(jarPath), err))
		}
	}

	for _, warPath := range warFiles {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		warProcessor := NewWarProcessor(p.cfrManager, p.workers, p.filterConfig)
		if err := warProcessor.Process(ctx, warPath, outputDir, rpt); err != nil {
			if p.textMode() {
				color.Red("处理WAR失败: %v", err)
			}
			if p.filterConfig.FailFast {
				return err
			}
			p.filterConfig.LogEvent("error", "war_process_failed", map[string]any{
				"war":   filepath.Base(warPath),
				"error": err.Error(),
			})
			procErrs = append(procErrs, fmt.Errorf("war %s: %w", filepath.Base(warPath), err))
		}
	}

	if len(filteredClasses) > 0 {
		jobs := make(chan string, len(filteredClasses))
		var wg sync.WaitGroup
		var mu sync.Mutex

		for i := 0; i < p.workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
			proc := NewClassProcessor(p.cfrManager, p.textMode())
				for classPath := range jobs {
					if ctx.Err() != nil {
						return
					}
					if err := proc.Process(ctx, classPath, outputDir, rpt); err != nil {
						mu.Lock()
						procErrs = append(procErrs, fmt.Errorf("class %s: %w", filepath.Base(classPath), err))
						mu.Unlock()
					}
				}
			}()
		}

		for _, file := range filteredClasses {
			if ctx.Err() != nil {
				break
			}
			jobs <- file
		}
		close(jobs)
		wg.Wait()
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}
	if len(procErrs) > 0 {
		return errors.Join(procErrs...)
	}

	return nil
}

// ExtractPackageName 从文件路径中提取包名
func ExtractPackageName(classPath string) string {
	normalized := normalizeArchivePath(classPath)
	if strings.Contains(normalized, "BOOT-INF/classes/") {
		parts := strings.Split(normalized, "BOOT-INF/classes/")
		if len(parts) > 1 {
			return filepath.ToSlash(filepath.Dir(parts[1]))
		}
	}

	if strings.Contains(normalized, "WEB-INF/classes/") {
		parts := strings.Split(normalized, "WEB-INF/classes/")
		if len(parts) > 1 {
			return filepath.ToSlash(filepath.Dir(parts[1]))
		}
	}

	return filepath.ToSlash(filepath.Dir(classPath))
}

// ScanDirectoryComplete 扫描目录,返回所有class、JAR和WAR文件(包括顶层)
func ScanDirectoryComplete(dir string, outputDir string) (classFiles []string, jarFiles []string, warFiles []string, err error) {
	absOutputDir, _ := filepath.Abs(outputDir)

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		absPath, _ := filepath.Abs(path)
		if strings.HasPrefix(absPath, absOutputDir) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			switch ext {
			case ".class":
				classFiles = append(classFiles, path)
			case ".jar":
				jarFiles = append(jarFiles, path)
			case ".war":
				warFiles = append(warFiles, path)
			}
		}
		return nil
	})
	return
}

// ScanDirectory 扫描目录,返回class文件、嵌套JAR文件和配置文件列表
func ScanDirectory(dir string) (classFiles []string, jarFiles []string, resourceFiles []string, err error) {
	resourceExts := map[string]bool{
		".properties": true,
		".yml":        true,
		".yaml":       true,
		".xml":        true,
		".json":       true,
		".conf":       true,
		".config":     true,
		".txt":        true,
		".sql":        true,
		".sh":         true,
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			switch ext {
			case ".class":
				classFiles = append(classFiles, path)
			case ".jar":
				if isLibJarPath(path) {
					jarFiles = append(jarFiles, path)
				}
			default:
				if resourceExts[ext] {
					if isClassesPath(path) {
						resourceFiles = append(resourceFiles, path)
					}
				}
			}
		}
		return nil
	})
	return
}

// CopyResourceFile 复制配置文件到输出目录
func CopyResourceFile(srcPath, tempDir, outputDir string) error {
	relPath, err := filepath.Rel(tempDir, srcPath)
	if err != nil {
		return err
	}
	relPath = filepath.ToSlash(relPath)

	if idx := strings.Index(relPath, "BOOT-INF/classes/"); idx != -1 {
		relPath = relPath[idx+len("BOOT-INF/classes/"):]
	} else if idx := strings.Index(relPath, "WEB-INF/classes/"); idx != -1 {
		relPath = relPath[idx+len("WEB-INF/classes/"):]
	}

	destPath := filepath.Join(outputDir, "resources", relPath)

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// UnzipFile 解压ZIP/JAR/WAR文件
func UnzipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("非法文件路径: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return fmt.Errorf("创建目录失败 %s: %w", fpath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
