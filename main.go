package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// 版本信息，将通过 -ldflags 在编译时注入
var (
	Version   = "dev"
	BuildTime = "unknown"
)

var rootCmd *cobra.Command

// 检查目录是否是 Tomcat 部署目录
func isTomcatDeployDir(path string) bool {
	// 检查是否存在 WEB-INF/classes 目录
	classesPath := filepath.Join(path, "WEB-INF", "classes")
	if stat, err := os.Stat(classesPath); err == nil && stat.IsDir() {
		return true
	}

	// 检查是否存在 WEB-INF/lib 目录
	libPath := filepath.Join(path, "WEB-INF", "lib")
	if stat, err := os.Stat(libPath); err == nil && stat.IsDir() {
		return true
	}

	return false
}

func init() {
	rootCmd = &cobra.Command{
		Use:   "emorad [文件或目录]",
		Short: "🎯 Emorad - Explore More Of Reverse And Decompile",
		Long: `Emorad is a powerful Java decompiler tool for Spring Boot JAR, WAR files, and Tomcat deployments.

✨ Features:
- 📦 Spring Boot JAR with nested dependencies
- 📦 WAR files and Tomcat deployments
- 📄 Individual CLASS files
- 🚀 Multi-core concurrent processing
- 📊 Beautiful HTML reports
- 🔧 Auto-managed CFR decompiler

如果不指定参数，将尝试反编译当前目录（假定为 Tomcat 部署目录）。`,
		Version: Version,
		Run: func(cmd *cobra.Command, args []string) {
			var inputPath string
			var err error

			if len(args) == 0 {
				// 如果没有参数，使用当前目录
				inputPath, err = os.Getwd()
				if err != nil {
					color.Red("无法获取当前目录: %v", err)
					return
				}

				// 检查当前目录是否是 Tomcat 部署目录
				if !isTomcatDeployDir(inputPath) {
					color.Red("当前目录不是有效的 Tomcat 部署目录")
					color.Yellow("需要包含 WEB-INF/classes 或 WEB-INF/lib 目录")
					color.Yellow("或者指定具体的 JAR/WAR 文件或目录作为参数")
					return
				}
			} else {
				inputPath = args[0]
			}

			// 获取输入文件的绝对路径
			absInputPath, err := filepath.Abs(inputPath)
			if err != nil {
				color.Red("无法获取输入路径的绝对路径: %v", err)
				return
			}

			// 获取输出目录
			outputDir, _ := cmd.Flags().GetString("output")
			if outputDir == "" {
				// 如果没有指定输出目录，使用输入文件所在目录下的 src 目录
				if stat, err := os.Stat(absInputPath); err == nil && !stat.IsDir() {
					// 如果输入是文件，使用其所在目录
					outputDir = filepath.Join(filepath.Dir(absInputPath), "src")
				} else {
					// 如果输入是目录，直接在其下创建 src 目录
					outputDir = filepath.Join(absInputPath, "src")
				}
			}

			// 执行反编译
			if err := decompile(absInputPath, outputDir); err != nil {
				color.Red("反编译失败: %v", err)
				return
			}
		},
	}

	rootCmd.Flags().StringP("output", "o", "", "输出目录（默认为当前目录下的 src 目录）")
	rootCmd.Flags().IntP("workers", "w", runtime.NumCPU(), "并发工作器数量")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
