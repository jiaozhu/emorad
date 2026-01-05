package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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

// parsePackagePrefixes 解析包前缀参数（支持逗号分隔）
func parsePackagePrefixes(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			// 将 . 分隔符转换为 / 以匹配 class 路径
			p = strings.ReplaceAll(p, ".", "/")
			// 确保以 / 结尾以匹配完整包名
			if !strings.HasSuffix(p, "/") {
				p += "/"
			}
			result = append(result, p)
		}
	}
	return result
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
- 🎯 Business code filtering (skip framework dependencies)

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

			workers, _ := cmd.Flags().GetInt("workers")

			// 构建过滤配置
			includeStr, _ := cmd.Flags().GetString("include")
			excludeStr, _ := cmd.Flags().GetString("exclude")
			skipLibs, _ := cmd.Flags().GetBool("skip-libs")
			noDefaultExclude, _ := cmd.Flags().GetBool("no-default-exclude")

			filterConfig := NewDefaultFilterConfig()
			filterConfig.SkipLibs = skipLibs

			// 处理包含过滤器
			if includes := parsePackagePrefixes(includeStr); len(includes) > 0 {
				filterConfig.Includes = includes
			}

			// 处理排除过滤器
			if excludes := parsePackagePrefixes(excludeStr); len(excludes) > 0 {
				filterConfig.Excludes = append(filterConfig.Excludes, excludes...)
			}

			// 如果设置了不使用默认排除
			if noDefaultExclude {
				filterConfig.Excludes = parsePackagePrefixes(excludeStr)
			}

			// 执行反编译
			if err := decompile(absInputPath, outputDir, workers, filterConfig); err != nil {
				color.Red("反编译失败: %v", err)
				return
			}
		},
	}

	rootCmd.Flags().StringP("output", "o", "", "输出目录（默认为当前目录下的 src 目录）")
	rootCmd.Flags().IntP("workers", "w", runtime.NumCPU(), "并发工作器数量")
	rootCmd.Flags().StringP("include", "i", "", "只处理匹配的包前缀，逗号分隔（如: com.mycompany,com.partner）")
	rootCmd.Flags().StringP("exclude", "e", "", "排除匹配的包前缀，逗号分隔（追加到默认排除列表）")
	rootCmd.Flags().Bool("skip-libs", true, "跳过 lib 目录下的依赖 JAR（默认启用）")
	rootCmd.Flags().Bool("no-default-exclude", false, "不使用默认的框架包排除列表")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
