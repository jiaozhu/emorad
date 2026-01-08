package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/jiaozhu/emorad/internal/decompile"
	"github.com/jiaozhu/emorad/internal/processor"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

var rootCmd *cobra.Command

func isTomcatDeployDir(path string) bool {
	classesPath := filepath.Join(path, "WEB-INF", "classes")
	if stat, err := os.Stat(classesPath); err == nil && stat.IsDir() {
		return true
	}

	libPath := filepath.Join(path, "WEB-INF", "lib")
	if stat, err := os.Stat(libPath); err == nil && stat.IsDir() {
		return true
	}

	return false
}

func parsePackagePrefixes(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			p = strings.ReplaceAll(p, ".", "/")
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
		Use:   "emorad [file or directory]",
		Short: "Java decompiler for Spring Boot applications",
		Long: `Decompile JAR, WAR, CLASS files and Tomcat deployments.

Automatically filters framework code and generates HTML/JSON reports.
Without arguments, decompiles the current directory.`,
		Version: Version,
		Run: func(cmd *cobra.Command, args []string) {
			var inputPath string
			var err error

			if len(args) == 0 {
				inputPath, err = os.Getwd()
				if err != nil {
					color.Red("Error: cannot get current directory: %v", err)
					return
				}

				if !isTomcatDeployDir(inputPath) {
					color.Red("Error: current directory is not a valid Tomcat deployment")
					color.Yellow("Hint: directory should contain WEB-INF/classes or WEB-INF/lib")
					color.Yellow("Hint: or specify a JAR/WAR file or directory as argument")
					return
				}
			} else {
				inputPath = args[0]
			}

			absInputPath, err := filepath.Abs(inputPath)
			if err != nil {
				color.Red("Error: cannot get absolute path: %v", err)
				return
			}

			outputDir, _ := cmd.Flags().GetString("output")
			if outputDir == "" {
				if stat, err := os.Stat(absInputPath); err == nil && !stat.IsDir() {
					outputDir = filepath.Join(filepath.Dir(absInputPath), "src")
				} else {
					outputDir = filepath.Join(absInputPath, "src")
				}
			}

			workers, _ := cmd.Flags().GetInt("workers")

			includeStr, _ := cmd.Flags().GetString("include")
			excludeStr, _ := cmd.Flags().GetString("exclude")
			jarIncludeStr, _ := cmd.Flags().GetString("jar-include")
			skipLibs, _ := cmd.Flags().GetBool("skip-libs")
			noDefaultExclude, _ := cmd.Flags().GetBool("no-default-exclude")

			filterConfig := processor.NewDefaultFilterConfig()
			filterConfig.SkipLibs = skipLibs
			filterConfig.CopyResources, _ = cmd.Flags().GetBool("copy-resources")
			filterConfig.CopyLibJars, _ = cmd.Flags().GetBool("copy-libs")
			filterConfig.GenerateIDEA, _ = cmd.Flags().GetBool("idea-project")

			if includes := parsePackagePrefixes(includeStr); len(includes) > 0 {
				filterConfig.Includes = includes
			}

			if excludes := parsePackagePrefixes(excludeStr); len(excludes) > 0 {
				filterConfig.Excludes = append(filterConfig.Excludes, excludes...)
			}

			if noDefaultExclude {
				filterConfig.Excludes = parsePackagePrefixes(excludeStr)
			}

			if jarIncludeStr != "" {
				parts := strings.Split(jarIncludeStr, ",")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						filterConfig.JarIncludes = append(filterConfig.JarIncludes, p)
					}
				}
			}

			if err := decompile.Run(absInputPath, outputDir, workers, filterConfig); err != nil {
				color.Red("Decompile failed: %v", err)
				return
			}
		},
	}

	rootCmd.Flags().StringP("output", "o", "", "Output directory (default: ./src)")
	rootCmd.Flags().IntP("workers", "w", runtime.NumCPU(), "Number of concurrent workers")
	rootCmd.Flags().StringP("include", "i", "", "Only process matching package prefixes, comma-separated")
	rootCmd.Flags().StringP("exclude", "e", "", "Exclude matching package prefixes, comma-separated")
	rootCmd.Flags().Bool("skip-libs", true, "Skip JAR files in lib directory")
	rootCmd.Flags().Bool("no-default-exclude", false, "Disable default framework exclusion list")
	rootCmd.Flags().StringP("jar-include", "j", "", "Only process lib JARs containing specified keywords")
	rootCmd.Flags().BoolP("copy-resources", "r", false, "Copy resource files to output/resources")
	rootCmd.Flags().Bool("copy-libs", false, "Copy dependency JARs to output/libs")
	rootCmd.Flags().Bool("idea-project", false, "Generate IDEA project structure with .iml file")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
