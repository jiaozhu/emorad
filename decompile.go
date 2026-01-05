package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

// decompile 执行反编译操作
func decompile(inputPath, outputDir string, workers int, filterConfig *FilterConfig) error {

	color.Cyan("\n🚀 开始反编译...")
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 显示过滤配置
	if len(filterConfig.Includes) > 0 {
		color.Green("📋 包含过滤: %v", filterConfig.Includes)
	}
	if len(filterConfig.Excludes) > 0 {
		color.Yellow("🚫 排除过滤: %d 个包前缀", len(filterConfig.Excludes))
	}
	if filterConfig.SkipLibs {
		color.Yellow("📦 跳过依赖库: 已启用")
	}

	// 初始化CFR管理器
	color.Cyan("📦 初始化反编译器...")
	cfrManager, err := NewCFRManager()
	if err != nil {
		color.Red("❌ 初始化CFR失败: %v", err)
		color.Yellow("\n💡 提示:")
		color.Yellow("   1. 请确保已安装Java环境")
		color.Yellow("   2. 工具会自动下载CFR反编译器")
		color.Yellow("   3. 或手动安装: brew install cfr-decompiler")
		return err
	}

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		color.Red("❌ 创建输出目录失败: %v", err)
		return err
	}

	// 检查输入路径的类型
	info, err := os.Stat(inputPath)
	if err != nil {
		color.Red("❌ 无法访问输入路径: %v", err)
		return err
	}

	// 创建报告
	report := NewDecompileReport(inputPath, outputDir)

	// 根据文件类型选择处理器
	var processor FileProcessor

	if info.IsDir() {
		// 目录处理
		processor = NewDirectoryProcessor(cfrManager, workers, filterConfig)
		color.Cyan("📁 检测到目录,使用目录处理器")
	} else {
		// 文件处理
		ext := strings.ToLower(filepath.Ext(inputPath))
		switch ext {
		case ".jar":
			processor = NewJarFileProcessor(cfrManager, workers, filterConfig)
			color.Cyan("📦 检测到JAR文件,使用JAR处理器")
		case ".war":
			processor = NewWarFileProcessor(cfrManager, workers, filterConfig)
			color.Cyan("📦 检测到WAR文件,使用WAR处理器")
		case ".class":
			processor = NewClassFileProcessor(cfrManager)
			color.Cyan("📄 检测到CLASS文件,使用CLASS处理器")
			report.SetTotalExpectedFiles(1)
		default:
			return fmt.Errorf("不支持的文件类型: %s", ext)
		}
	}

	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// 执行处理
	if err := processor.Process(inputPath, outputDir, report); err != nil {
		color.Red("\n❌ 处理失败: %v", err)
		// 即使有错误也生成报告
		report.GenerateReport()
		return err
	}

	// 生成报告
	return report.GenerateReport()
}
