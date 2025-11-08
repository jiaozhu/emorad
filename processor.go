package main

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
)

// FileProcessor 定义文件处理器接口
type FileProcessor interface {
	Process(inputPath string, outputDir string, report *DecompileReport) error
	GetType() string
}

// ClassFileProcessor 处理单个.class文件
type ClassFileProcessor struct {
	cfrManager *CFRManager
}

func NewClassFileProcessor(cfrManager *CFRManager) *ClassFileProcessor {
	return &ClassFileProcessor{cfrManager: cfrManager}
}

func (p *ClassFileProcessor) GetType() string {
	return "class"
}

func (p *ClassFileProcessor) Process(inputPath string, outputDir string, report *DecompileReport) error {
	startTime := time.Now()
	result := DecompileResult{
		ClassName:   filepath.Base(inputPath),
		PackageName: extractPackageName(inputPath),
		Success:     false,
		TimeStamp:   startTime,
	}

	// 使用CFR反编译
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
	report.AddResult(result)
	return err
}

// JarFileProcessor 处理JAR文件
type JarFileProcessor struct {
	cfrManager *CFRManager
	workers    int
}

func NewJarFileProcessor(cfrManager *CFRManager, workers int) *JarFileProcessor {
	return &JarFileProcessor{
		cfrManager: cfrManager,
		workers:    workers,
	}
}

func (p *JarFileProcessor) GetType() string {
	return "jar"
}

func (p *JarFileProcessor) Process(inputPath string, outputDir string, report *DecompileReport) error {
	color.Cyan("正在处理JAR文件: %s", filepath.Base(inputPath))

	// 创建临时目录
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("emorad-%s-%d",
		filepath.Base(inputPath), time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %v", err)
	}

	// 解压JAR文件
	if err := unzipFile(inputPath, tempDir); err != nil {
		return fmt.Errorf("解压JAR文件失败: %v", err)
	}

	// 收集所有class文件和嵌套的JAR文件
	classFiles, nestedJars, err := scanDirectory(tempDir)
	if err != nil {
		return fmt.Errorf("扫描目录失败: %v", err)
	}

	// 设置预期文件数
	report.SetTotalExpectedFiles(int32(len(classFiles)))

	// 处理嵌套的JAR文件(递归)
	for _, nestedJar := range nestedJars {
		color.Yellow("处理嵌套JAR: %s", filepath.Base(nestedJar))
		nestedProcessor := NewJarFileProcessor(p.cfrManager, p.workers)
		if err := nestedProcessor.Process(nestedJar, outputDir, report); err != nil {
			color.Red("处理嵌套JAR失败: %v", err)
		}
	}

	// 并发处理所有class文件
	return p.processClassFiles(classFiles, outputDir, report)
}

func (p *JarFileProcessor) processClassFiles(classFiles []string, outputDir string, report *DecompileReport) error {
	jobs := make(chan string, len(classFiles))
	var wg sync.WaitGroup

	// 启动工作线程
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processor := NewClassFileProcessor(p.cfrManager)
			for classPath := range jobs {
				processor.Process(classPath, outputDir, report)
			}
		}()
	}

	// 分发任务
	for _, file := range classFiles {
		jobs <- file
	}
	close(jobs)

	// 等待完成
	wg.Wait()
	return nil
}

// WarFileProcessor 处理WAR文件
type WarFileProcessor struct {
	*JarFileProcessor
}

func NewWarFileProcessor(cfrManager *CFRManager, workers int) *WarFileProcessor {
	return &WarFileProcessor{
		JarFileProcessor: NewJarFileProcessor(cfrManager, workers),
	}
}

func (p *WarFileProcessor) GetType() string {
	return "war"
}

// DirectoryProcessor 处理目录
type DirectoryProcessor struct {
	cfrManager *CFRManager
	workers    int
}

func NewDirectoryProcessor(cfrManager *CFRManager, workers int) *DirectoryProcessor {
	return &DirectoryProcessor{
		cfrManager: cfrManager,
		workers:    workers,
	}
}

func (p *DirectoryProcessor) GetType() string {
	return "directory"
}

func (p *DirectoryProcessor) Process(inputPath string, outputDir string, report *DecompileReport) error {
	color.Cyan("正在处理目录: %s", inputPath)

	// 扫描目录 - 包括顶层JAR文件和class文件
	classFiles, jarFiles, warFiles, err := scanDirectoryComplete(inputPath, outputDir)
	if err != nil {
		return fmt.Errorf("扫描目录失败: %v", err)
	}

	color.Cyan("📊 扫描结果: %d个JAR, %d个WAR, %d个CLASS文件",
		len(jarFiles), len(warFiles), len(classFiles))

	// 如果没有找到任何文件
	if len(jarFiles) == 0 && len(warFiles) == 0 && len(classFiles) == 0 {
		color.Yellow("⚠️  未找到任何需要反编译的文件")
		return nil
	}

	// 设置预期文件数
	report.SetTotalExpectedFiles(int32(len(classFiles)))

	// 处理JAR文件
	for _, jarPath := range jarFiles {
		color.Yellow("处理JAR文件: %s", filepath.Base(jarPath))
		jarProcessor := NewJarFileProcessor(p.cfrManager, p.workers)
		if err := jarProcessor.Process(jarPath, outputDir, report); err != nil {
			color.Red("处理JAR失败: %v", err)
		}
	}

	// 处理WAR文件
	for _, warPath := range warFiles {
		color.Yellow("处理WAR文件: %s", filepath.Base(warPath))
		warProcessor := NewWarFileProcessor(p.cfrManager, p.workers)
		if err := warProcessor.Process(warPath, outputDir, report); err != nil {
			color.Red("处理WAR失败: %v", err)
		}
	}

	// 处理class文件
	if len(classFiles) > 0 {
		jobs := make(chan string, len(classFiles))
		var wg sync.WaitGroup

		for i := 0; i < p.workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				processor := NewClassFileProcessor(p.cfrManager)
				for classPath := range jobs {
					processor.Process(classPath, outputDir, report)
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

// 工具函数

// extractPackageName 从文件路径中提取包名
func extractPackageName(classPath string) string {
	// Spring Boot JAR结构
	if strings.Contains(classPath, "BOOT-INF/classes/") {
		parts := strings.Split(classPath, "BOOT-INF/classes/")
		if len(parts) > 1 {
			return filepath.ToSlash(filepath.Dir(parts[1]))
		}
	}

	// Tomcat WAR结构
	if strings.Contains(classPath, "WEB-INF/classes/") {
		parts := strings.Split(classPath, "WEB-INF/classes/")
		if len(parts) > 1 {
			return filepath.ToSlash(filepath.Dir(parts[1]))
		}
	}

	// 普通目录结构
	return filepath.ToSlash(filepath.Dir(classPath))
}

// scanDirectoryComplete 扫描目录,返回所有class、JAR和WAR文件(包括顶层)
func scanDirectoryComplete(dir string, outputDir string) (classFiles []string, jarFiles []string, warFiles []string, err error) {
	// 获取输出目录的绝对路径用于排除
	absOutputDir, _ := filepath.Abs(outputDir)

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 排除输出目录
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

// scanDirectory 扫描目录,返回class文件和嵌套JAR文件列表(用于已解压的JAR内部)
func scanDirectory(dir string) (classFiles []string, jarFiles []string, err error) {
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
				// 只处理BOOT-INF/lib和WEB-INF/lib下的JAR
				if strings.Contains(path, "BOOT-INF/lib") || strings.Contains(path, "WEB-INF/lib") {
					jarFiles = append(jarFiles, path)
				}
			}
		}
		return nil
	})
	return
}

// unzipFile 解压ZIP/JAR/WAR文件
func unzipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// 安全检查:防止路径遍历攻击
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("非法文件路径: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
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
