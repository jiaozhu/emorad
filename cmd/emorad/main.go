package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"

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

func resolveDefaultOutput(absInputPath string, ideaProject bool) string {
	if stat, err := os.Stat(absInputPath); err == nil && !stat.IsDir() {
		baseDir := filepath.Dir(absInputPath)
		if ideaProject {
			return filepath.Join(baseDir, "decompiled")
		}
		return filepath.Join(baseDir, "src")
	}
	if ideaProject {
		return filepath.Join(absInputPath, "decompiled")
	}
	return filepath.Join(absInputPath, "src")
}

func deriveAppJarHints(absInputPath string) []string {
	candidates := []string{filepath.Base(absInputPath)}
	if stat, err := os.Stat(absInputPath); err == nil && !stat.IsDir() {
		candidates = append(candidates, filepath.Base(filepath.Dir(absInputPath)))
	}

	partsSet := make(map[string]struct{})
	for _, c := range candidates {
		c = strings.ToLower(strings.TrimSpace(c))
		c = strings.TrimSuffix(c, filepath.Ext(c))
		for _, part := range strings.FieldsFunc(c, func(r rune) bool {
			return r == '-' || r == '_' || r == '.' || r == ' '
		}) {
			if len(part) >= 3 {
				partsSet[part] = struct{}{}
			}
		}
	}

	out := make([]string, 0, len(partsSet))
	for k := range partsSet {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func init() {
	rootCmd = &cobra.Command{
		Use:   "emorad [file or directory]",
		Short: "Java decompiler for Spring Boot applications",
		Long: `Decompile JAR, WAR, CLASS files and Tomcat deployments.

Automatically filters framework code and generates HTML/JSON reports.
Without arguments, automatically scans and decompiles supported artifacts in the current directory.`,
		Version: Version,
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			var inputPath string
			var err error

			if len(args) == 0 {
				inputPath, err = os.Getwd()
				if err != nil {
					color.Red("Error: cannot get current directory: %v", err)
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

			workers, _ := cmd.Flags().GetInt("workers")
			if workers <= 0 {
				color.Red("Error: --workers must be > 0")
				return
			}

			includeStr, _ := cmd.Flags().GetString("include")
			excludeStr, _ := cmd.Flags().GetString("exclude")
			excludeMode, _ := cmd.Flags().GetString("exclude-mode")
			jarIncludeStr, _ := cmd.Flags().GetString("jar-include")
			jarMatchMode, _ := cmd.Flags().GetString("jar-match-mode")
			skipLibs, _ := cmd.Flags().GetBool("skip-libs")
			noDefaultExclude, _ := cmd.Flags().GetBool("no-default-exclude")
			nestedJarStrategy, _ := cmd.Flags().GetString("nested-jar-strategy")
			maxJarDepth, _ := cmd.Flags().GetInt("max-jar-depth")
			logFormat, _ := cmd.Flags().GetString("log-format")
			unicodePostprocess, _ := cmd.Flags().GetBool("unicode-postprocess")
			failFast, _ := cmd.Flags().GetBool("fail-fast")
			configPath, _ := cmd.Flags().GetString("config")

			filterConfig := processor.NewDefaultFilterConfig()
			filterConfig.SkipLibs = skipLibs
			filterConfig.JarMatchMode = strings.ToLower(jarMatchMode)
			filterConfig.NestedJarStrategy = nestedJarStrategy
			filterConfig.MaxJarDepth = maxJarDepth
			filterConfig.LogFormat = strings.ToLower(logFormat)
			filterConfig.CopyResources, _ = cmd.Flags().GetBool("copy-resources")
			filterConfig.CopyLibJars, _ = cmd.Flags().GetBool("copy-libs")
			filterConfig.GenerateIDEA, _ = cmd.Flags().GetBool("idea-project")
			filterConfig.UnicodePostprocess = unicodePostprocess
			filterConfig.FailFast = failFast
			filterConfig.AppJarHints = deriveAppJarHints(absInputPath)

			cfg, loadedPath, err := loadConfig(configPath)
			if err != nil {
				color.Red("Error: load config failed: %v", err)
				return
			}
			if cfg != nil {
				if filterConfig.LogFormat == "text" {
					color.Cyan("[CONFIG] 已加载配置文件: %s", loadedPath)
				}
				if len(cfg.Filter.CommonJarPrefixes) > 0 {
					filterConfig.CommonJarPrefixes = append([]string{}, cfg.Filter.CommonJarPrefixes...)
				}
				if len(cfg.Filter.AppJarHints) > 0 {
					filterConfig.AppJarHints = append(filterConfig.AppJarHints, cfg.Filter.AppJarHints...)
				}
				if len(cfg.Filter.JarInclude) > 0 {
					filterConfig.JarIncludes = append(filterConfig.JarIncludes, cfg.Filter.JarInclude...)
				}
			}
			if outputDir == "" {
				outputDir = resolveDefaultOutput(absInputPath, filterConfig.GenerateIDEA)
			}

			if includes := parsePackagePrefixes(includeStr); len(includes) > 0 {
				filterConfig.Includes = includes
			}

			if noDefaultExclude {
				excludeMode = "replace"
			}
			excludeMode = strings.ToLower(strings.TrimSpace(excludeMode))
			excludes := parsePackagePrefixes(excludeStr)
			switch excludeMode {
			case "append":
				if len(excludes) > 0 {
					filterConfig.Excludes = append(filterConfig.Excludes, excludes...)
				}
			case "replace":
				filterConfig.Excludes = excludes
			case "none":
				filterConfig.Excludes = nil
			default:
				color.Red("Error: --exclude-mode must be append, replace or none")
				return
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

			s := strings.ToLower(filterConfig.NestedJarStrategy)
			if s != "skip" && s != "filtered" && s != "full" {
				color.Red("Error: invalid --nested-jar-strategy, expected skip|filtered|full")
				return
			}
			filterConfig.NestedJarStrategy = s
			if filterConfig.MaxJarDepth < 0 {
				color.Red("Error: --max-jar-depth must be >= 0")
				return
			}
			if filterConfig.LogFormat != "text" && filterConfig.LogFormat != "json" {
				color.Red("Error: --log-format must be text or json")
				return
			}
			if filterConfig.JarMatchMode != "contains" && filterConfig.JarMatchMode != "prefix" {
				color.Red("Error: --jar-match-mode must be contains or prefix")
				return
			}

			if err := decompile.Run(ctx, absInputPath, outputDir, workers, filterConfig); err != nil {
				color.Red("Decompile failed: %v", err)
				return
			}
		},
	}

	rootCmd.Flags().StringP("output", "o", "", "Output directory (default auto: src; decompiled when --idea-project)")
	rootCmd.Flags().IntP("workers", "w", runtime.NumCPU(), "Number of concurrent workers")
	rootCmd.Flags().StringP("include", "i", "", "Only process matching package prefixes, comma-separated")
	rootCmd.Flags().StringP("exclude", "e", "", "Exclude matching package prefixes, comma-separated")
	rootCmd.Flags().String("exclude-mode", "append", "Exclude strategy: append | replace | none")
	rootCmd.Flags().Bool("skip-libs", true, "Skip JAR files in lib directory")
	rootCmd.Flags().Bool("no-default-exclude", false, "Disable default framework exclusion list")
	_ = rootCmd.Flags().MarkDeprecated("no-default-exclude", "use --exclude-mode=replace")
	rootCmd.Flags().StringP("jar-include", "j", "", "Only process lib JARs matching specified keywords")
	rootCmd.Flags().String("jar-match-mode", "contains", "JAR match mode: contains or prefix")
	rootCmd.Flags().String("nested-jar-strategy", "filtered", "Nested JAR strategy: skip | filtered | full")
	rootCmd.Flags().Int("max-jar-depth", 8, "Maximum nested JAR recursion depth")
	rootCmd.Flags().String("log-format", "text", "Log format: text or json")
	rootCmd.Flags().String("config", "", "Config file path (default: ./.emorad.yaml or ~/.emorad/config.yaml)")
	rootCmd.Flags().Bool("unicode-postprocess", true, "Decode Unicode escape sequences in generated .java files")
	rootCmd.Flags().Bool("fail-fast", false, "Stop processing immediately on first error")
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
