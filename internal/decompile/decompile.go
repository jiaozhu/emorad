package decompile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jiaozhu/emorad/internal/cfr"
	"github.com/jiaozhu/emorad/internal/processor"
	"github.com/jiaozhu/emorad/internal/report"
)

func logEvent(format string, level string, event string, kv map[string]any) {
	if format == "json" {
		payload := map[string]any{
			"ts":    time.Now().Format(time.RFC3339),
			"level": level,
			"event": event,
		}
		for k, v := range kv {
			payload[k] = v
		}
		b, _ := json.Marshal(payload)
		fmt.Println(string(b))
		return
	}
	if len(kv) == 0 {
		fmt.Printf("[%s] %s\n", strings.ToUpper(level), event)
		return
	}
	fmt.Printf("[%s] %s: %v\n", strings.ToUpper(level), event, kv)
}

// Run 执行反编译操作
func Run(ctx context.Context, inputPath, outputDir string, workers int, filterConfig *processor.FilterConfig) error {

	logEvent(filterConfig.LogFormat, "info", "start_decompile", map[string]any{"input": inputPath, "workers": workers})
	color.Cyan("\n[START] 开始反编译...")
	color.Cyan("============================================")

	// 显示过滤配置
	if len(filterConfig.Includes) > 0 {
		color.Green("[FILTER] 包含过滤: %v", filterConfig.Includes)
	}
	if len(filterConfig.Excludes) > 0 {
		color.Yellow("[FILTER] 排除过滤: %d 个包前缀", len(filterConfig.Excludes))
	}
	if filterConfig.SkipLibs {
		color.Yellow("[CONFIG] 跳过依赖库: 已启用")
	}
	if len(filterConfig.JarIncludes) > 0 {
		color.Green("[FILTER] JAR 名称过滤(%s): %v", filterConfig.JarMatchMode, filterConfig.JarIncludes)
	}
	color.Green("[CONFIG] 嵌套JAR策略: %s (max-depth=%d)", filterConfig.NestedJarStrategy, filterConfig.MaxJarDepth)
	if filterConfig.CopyResources {
		color.Green("[CONFIG] 复制配置文件: 已启用")
	}
	if filterConfig.CopyLibJars {
		color.Green("[CONFIG] 复制依赖 JAR: 已启用")
	}
	if filterConfig.GenerateIDEA {
		color.Green("[CONFIG] 生成 IDEA 项目: 已启用")
	}

	// 初始化CFR管理器
	color.Cyan("[INIT] 初始化反编译器...")
	initStart := time.Now()
	cfrManager, err := cfr.NewManager()
	if err != nil {
		color.Red("[ERROR] 初始化CFR失败: %v", err)
		color.Yellow("\n[TIP] 提示:")
		color.Yellow("   1. 请确保已安装Java环境")
		color.Yellow("   2. 工具会自动下载CFR反编译器")
		color.Yellow("   3. 或手动安装: brew install cfr-decompiler")
		return err
	}

	// 创建输出目录（源代码放在 src 子目录）
	srcDir := outputDir
	if filterConfig.GenerateIDEA {
		srcDir = filepath.Join(outputDir, "src")
	}
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		color.Red("[ERROR] 创建输出目录失败: %v", err)
		return err
	}

	// 检查输入路径的类型
	info, err := os.Stat(inputPath)
	if err != nil {
		color.Red("[ERROR] 无法访问输入路径: %v", err)
		return err
	}

	// 创建报告
	rpt := report.New(inputPath, srcDir)
	rpt.AddStageDuration("init", time.Since(initStart).Seconds())

	// 根据文件类型选择处理器
	var proc processor.Processor

	if info.IsDir() {
		// 目录处理
		proc = processor.NewDirectoryProcessor(cfrManager, workers, filterConfig)
		color.Cyan("[DETECT] 检测到目录,使用目录处理器")
	} else {
		// 文件处理
		ext := strings.ToLower(filepath.Ext(inputPath))
		switch ext {
		case ".jar":
			proc = processor.NewJarProcessor(cfrManager, workers, filterConfig)
			color.Cyan("[DETECT] 检测到JAR文件,使用JAR处理器")
		case ".war":
			proc = processor.NewWarProcessor(cfrManager, workers, filterConfig)
			color.Cyan("[DETECT] 检测到WAR文件,使用WAR处理器")
		case ".class":
			proc = processor.NewClassProcessor(cfrManager)
			color.Cyan("[DETECT] 检测到CLASS文件,使用CLASS处理器")
			rpt.SetTotalExpectedFiles(1)
		default:
			return fmt.Errorf("不支持的文件类型: %s", ext)
		}
	}

	color.Cyan("============================================\n")

	// 执行处理
	processStart := time.Now()
	if err := proc.Process(ctx, inputPath, srcDir, rpt); err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			color.Yellow("\n[CANCEL] 任务已中断: %v", err)
			rpt.MarkCancelled()
			rpt.AddStageDuration("process", time.Since(processStart).Seconds())
			logEvent(filterConfig.LogFormat, "warn", "decompile_cancelled", map[string]any{"error": err.Error()})
			_ = rpt.Generate()
			return err
		}
		color.Red("\n[ERROR] 处理失败: %v", err)
		rpt.AddResult(report.Result{
			ClassName: filepath.Base(inputPath),
			FilePath:  inputPath,
			Stage:     "process",
			ErrorCode: "PROCESS_FAILED",
			Success:   false,
			Error:     err.Error(),
			TimeStamp: time.Now(),
		})
		rpt.AddStageDuration("process", time.Since(processStart).Seconds())
		logEvent(filterConfig.LogFormat, "error", "decompile_failed", map[string]any{"error": err.Error()})
		// 即使有错误也生成报告
		rpt.Generate()
		return err
	}
	rpt.AddStageDuration("process", time.Since(processStart).Seconds())

	// Unicode 后处理：将 \uXXXX 转换为实际的中文字符
	color.Cyan("\n[PROCESS] 处理 Unicode 转义序列...")
	unicodeStart := time.Now()
	processed, modified, err := processor.ProcessDirectoryUnicode(srcDir)
	if err != nil {
		color.Yellow("[WARN] Unicode 后处理警告: %v", err)
	} else if modified > 0 {
		color.Green("[OK] Unicode 后处理完成: 处理 %d 文件, 修复 %d 文件", processed, modified)
	}
	rpt.AddStageDuration("unicode_postprocess", time.Since(unicodeStart).Seconds())

	// 生成 IDEA 项目配置
	if filterConfig.GenerateIDEA {
		color.Cyan("\n[PROCESS] 生成 IDEA 项目配置...")
		ideaStart := time.Now()
		projectName := filepath.Base(outputDir)
		if projectName == "." || projectName == "" {
			projectName = "decompiled"
		}

		projectConfig := &processor.ProjectConfig{
			ProjectName: projectName,
			OutputDir:   outputDir,
			SrcDir:      srcDir,
			LibsDir:     filepath.Join(outputDir, "libs"),
		}

		if err := processor.GenerateIDEAProject(projectConfig); err != nil {
			color.Yellow("[WARN] 生成 IDEA 项目配置失败: %v", err)
		} else {
			color.Green("[OK] IDEA 项目配置已生成，可直接用 IDEA 打开: %s", outputDir)
		}
		rpt.AddStageDuration("idea_project", time.Since(ideaStart).Seconds())
	}

	logEvent(filterConfig.LogFormat, "info", "decompile_finished", map[string]any{"status": rpt.Status})
	// 生成报告
	return rpt.Generate()
}
