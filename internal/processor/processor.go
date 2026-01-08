package processor

import (
	"archive/zip"
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
	Includes      []string // 包含的包前缀（优先级最高）
	Excludes      []string // 排除的包前缀
	SkipLibs      bool     // 是否跳过 lib 目录下的 JAR
	JarIncludes   []string // JAR 名称必须包含的关键字
	CopyResources bool     // 是否复制配置文件到输出目录
}

// NewDefaultFilterConfig 创建默认过滤配置
func NewDefaultFilterConfig() *FilterConfig {
	return &FilterConfig{
		Includes: nil,
		Excludes: DefaultExcludes,
		SkipLibs: true,
	}
}

// ShouldProcessClass 判断是否应该处理该 class 文件
func (f *FilterConfig) ShouldProcessClass(classPath string) bool {
	relativePath := extractRelativePath(classPath)

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

// ShouldProcessJar 判断是否应该处理该 JAR 文件
func (f *FilterConfig) ShouldProcessJar(jarPath string) bool {
	isLibJar := strings.Contains(jarPath, "BOOT-INF/lib") || strings.Contains(jarPath, "WEB-INF/lib")

	if isLibJar {
		if f.SkipLibs {
			if len(f.JarIncludes) > 0 {
				jarName := strings.ToLower(filepath.Base(jarPath))
				for _, keyword := range f.JarIncludes {
					if strings.Contains(jarName, strings.ToLower(keyword)) {
						return true
					}
				}
			}
			return false
		}
		if len(f.JarIncludes) > 0 {
			jarName := strings.ToLower(filepath.Base(jarPath))
			for _, keyword := range f.JarIncludes {
				if strings.Contains(jarName, strings.ToLower(keyword)) {
					return true
				}
			}
			return false
		}
	}
	return true
}

// extractRelativePath 从 class 路径中提取相对包路径
func extractRelativePath(classPath string) string {
	if idx := strings.Index(classPath, "BOOT-INF/classes/"); idx != -1 {
		return classPath[idx+len("BOOT-INF/classes/"):]
	}
	if idx := strings.Index(classPath, "WEB-INF/classes/"); idx != -1 {
		return classPath[idx+len("WEB-INF/classes/"):]
	}
	// 对于 JAR 根目录下的类（如 org/springframework/boot/loader/），
	// 返回完整相对路径以便排除规则能够匹配
	// 将路径转换为正斜杠形式以匹配排除列表
	return filepath.ToSlash(classPath)
}

// Processor 定义文件处理器接口
type Processor interface {
	Process(inputPath string, outputDir string, rpt *report.Report) error
	GetType() string
}

// ClassProcessor 处理单个.class文件
type ClassProcessor struct {
	cfrManager *cfr.Manager
}

func NewClassProcessor(cfrManager *cfr.Manager) *ClassProcessor {
	return &ClassProcessor{cfrManager: cfrManager}
}

func (p *ClassProcessor) GetType() string {
	return "class"
}

func (p *ClassProcessor) Process(inputPath string, outputDir string, rpt *report.Report) error {
	startTime := time.Now()
	result := report.Result{
		ClassName:   filepath.Base(inputPath),
		PackageName: ExtractPackageName(inputPath),
		Success:     false,
		TimeStamp:   startTime,
	}

	err := p.cfrManager.Decompile(inputPath, outputDir)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("反编译失败: %v", err)
		color.Red("✗ %s", result.ClassName)
	} else {
		result.Success = true
		color.Green("✓ %s", result.ClassName)
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

func (p *JarProcessor) Process(inputPath string, outputDir string, rpt *report.Report) error {
	color.Cyan("正在处理JAR文件: %s", filepath.Base(inputPath))

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
				color.Red("复制配置文件失败: %s - %v", filepath.Base(resFile), err)
			} else {
				copiedCount++
			}
		}
		if copiedCount > 0 {
			color.Green("✅ 复制了 %d 个配置文件", copiedCount)
		}
	}

	filteredClasses := make([]string, 0, len(classFiles))
	for _, classPath := range classFiles {
		if p.filterConfig.ShouldProcessClass(classPath) {
			filteredClasses = append(filteredClasses, classPath)
		}
	}

	if len(classFiles) != len(filteredClasses) {
		color.Yellow("📝 过滤后: %d/%d 个 class 文件需要处理", len(filteredClasses), len(classFiles))
	}

	rpt.AddExpectedFiles(int32(len(filteredClasses)))

	for _, nestedJar := range nestedJars {
		if !p.filterConfig.ShouldProcessJar(nestedJar) {
			continue
		}
		color.Yellow("处理嵌套JAR: %s", filepath.Base(nestedJar))
		nestedProcessor := NewJarProcessor(p.cfrManager, p.workers, p.filterConfig)
		if err := nestedProcessor.Process(nestedJar, outputDir, rpt); err != nil {
			color.Red("处理嵌套JAR失败: %v", err)
		}
	}

	return p.processClassFiles(filteredClasses, outputDir, rpt)
}

func (p *JarProcessor) processClassFiles(classFiles []string, outputDir string, rpt *report.Report) error {
	jobs := make(chan string, len(classFiles))
	var wg sync.WaitGroup

	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processor := NewClassProcessor(p.cfrManager)
			for classPath := range jobs {
				processor.Process(classPath, outputDir, rpt)
			}
		}()
	}

	for _, file := range classFiles {
		jobs <- file
	}
	close(jobs)

	wg.Wait()
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

func (p *DirectoryProcessor) Process(inputPath string, outputDir string, rpt *report.Report) error {
	color.Cyan("正在处理目录: %s", inputPath)

	classFiles, jarFiles, warFiles, err := ScanDirectoryComplete(inputPath, outputDir)
	if err != nil {
		return fmt.Errorf("扫描目录失败: %v", err)
	}

	color.Cyan("📊 扫描结果: %d个JAR, %d个WAR, %d个CLASS文件",
		len(jarFiles), len(warFiles), len(classFiles))

	if len(jarFiles) == 0 && len(warFiles) == 0 && len(classFiles) == 0 {
		color.Yellow("⚠️  未找到任何需要反编译的文件")
		return nil
	}

	rpt.AddExpectedFiles(int32(len(classFiles)))

	for _, jarPath := range jarFiles {
		color.Yellow("处理JAR文件: %s", filepath.Base(jarPath))
		jarProcessor := NewJarProcessor(p.cfrManager, p.workers, p.filterConfig)
		if err := jarProcessor.Process(jarPath, outputDir, rpt); err != nil {
			color.Red("处理JAR失败: %v", err)
		}
	}

	for _, warPath := range warFiles {
		color.Yellow("处理WAR文件: %s", filepath.Base(warPath))
		warProcessor := NewWarProcessor(p.cfrManager, p.workers, p.filterConfig)
		if err := warProcessor.Process(warPath, outputDir, rpt); err != nil {
			color.Red("处理WAR失败: %v", err)
		}
	}

	if len(classFiles) > 0 {
		jobs := make(chan string, len(classFiles))
		var wg sync.WaitGroup

		for i := 0; i < p.workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				proc := NewClassProcessor(p.cfrManager)
				for classPath := range jobs {
					proc.Process(classPath, outputDir, rpt)
				}
			}()
		}

		for _, file := range classFiles {
			jobs <- file
		}
		close(jobs)
		wg.Wait()
	}

	return nil
}

// ExtractPackageName 从文件路径中提取包名
func ExtractPackageName(classPath string) string {
	if strings.Contains(classPath, "BOOT-INF/classes/") {
		parts := strings.Split(classPath, "BOOT-INF/classes/")
		if len(parts) > 1 {
			return filepath.ToSlash(filepath.Dir(parts[1]))
		}
	}

	if strings.Contains(classPath, "WEB-INF/classes/") {
		parts := strings.Split(classPath, "WEB-INF/classes/")
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
				if strings.Contains(path, "BOOT-INF/lib") || strings.Contains(path, "WEB-INF/lib") {
					jarFiles = append(jarFiles, path)
				}
			default:
				if resourceExts[ext] {
					if strings.Contains(path, "BOOT-INF/classes") || strings.Contains(path, "WEB-INF/classes") {
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
