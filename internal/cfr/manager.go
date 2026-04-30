package cfr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
)

const (
	// CFR下载地址
	DownloadURL = "https://github.com/leibnitz27/cfr/releases/download/0.152/cfr-0.152.jar"
	Version     = "0.152"
)

const (
	envCFRJarPath     = "EMORAD_CFR_JAR_PATH"
	envCFRDir         = "EMORAD_CFR_DIR"
	envCFRDownloadURL = "EMORAD_CFR_DOWNLOAD_URL"
	envSkipDownload   = "EMORAD_SKIP_CFR_DOWNLOAD"
)

// Manager 管理CFR反编译器
type Manager struct {
	cfrPath  string // CFR JAR文件路径或命令路径
	useJar   bool   // 是否使用JAR文件
	javaPath string // Java命令路径
}

// NewManager 创建CFR管理器
func NewManager() (*Manager, error) {
	manager := &Manager{}

	// 首先尝试使用系统安装的cfr-decompiler命令
	if path, err := exec.LookPath("cfr-decompiler"); err == nil {
		color.Green("✓ 找到系统CFR命令: %s", path)
		manager.cfrPath = path
		manager.useJar = false
		return manager, nil
	}

	// 如果没有系统命令,尝试使用Java运行CFR JAR
	color.Yellow("系统未安装CFR命令,尝试使用CFR JAR文件...")

	// 检查Java是否可用
	javaPath, err := exec.LookPath("java")
	if err != nil {
		return nil, fmt.Errorf("未找到Java环境,请安装Java: %v", err)
	}
	manager.javaPath = javaPath

	// 获取或下载CFR JAR文件
	cfrJarPath, err := manager.ensureCFRJar()
	if err != nil {
		return nil, err
	}

	manager.cfrPath = cfrJarPath
	manager.useJar = true
	color.Green("✓ 使用CFR JAR: %s", cfrJarPath)

	return manager, nil
}

func getDefaultCFRDir() (string, error) {
	if custom := strings.TrimSpace(os.Getenv(envCFRDir)); custom != "" {
		return custom, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户主目录: %v", err)
	}
	return filepath.Join(homeDir, ".emorad", "cfr"), nil
}

func getDefaultCFRJarPath() (string, error) {
	cfrDir, err := getDefaultCFRDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfrDir, fmt.Sprintf("cfr-%s.jar", Version)), nil
}

func getDownloadURL() string {
	if custom := strings.TrimSpace(os.Getenv(envCFRDownloadURL)); custom != "" {
		return custom
	}
	return DownloadURL
}

func shouldSkipDownload() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(envSkipDownload)))
	return v == "1" || v == "true" || v == "yes"
}

func manualInstallHint(targetPath string) string {
	base := []string{
		"可选方案:",
		fmt.Sprintf("1) 手动下载 cfr-%s.jar 并放到: %s", Version, targetPath),
		fmt.Sprintf("2) 设置环境变量 %s 指向本地 cfr jar", envCFRJarPath),
		fmt.Sprintf("3) 设置环境变量 %s 指定缓存目录", envCFRDir),
		fmt.Sprintf("4) 设置环境变量 %s 使用内网镜像地址", envCFRDownloadURL),
	}
	if runtime.GOOS == "windows" {
		base = append(base,
			"",
			"Windows 示例:",
			fmt.Sprintf("  set %s=C:\\tools\\cfr\\cfr-%s.jar", envCFRJarPath, Version),
			fmt.Sprintf("  set %s=1", envSkipDownload),
		)
	} else {
		base = append(base,
			"",
			"Linux 示例:",
			fmt.Sprintf("  export %s=/opt/tools/cfr/cfr-%s.jar", envCFRJarPath, Version),
			fmt.Sprintf("  export %s=1", envSkipDownload),
		)
	}
	return strings.Join(base, "\n")
}

// ensureCFRJar 确保CFR JAR文件存在,如果不存在则下载
func (m *Manager) ensureCFRJar() (string, error) {
	// 1) 显式路径优先
	if customPath := strings.TrimSpace(os.Getenv(envCFRJarPath)); customPath != "" {
		if _, err := os.Stat(customPath); err == nil {
			color.Green("✓ 使用环境变量指定的 CFR JAR: %s", customPath)
			return customPath, nil
		}
		return "", fmt.Errorf("环境变量 %s 指定的文件不存在: %s", envCFRJarPath, customPath)
	}

	// 2) 默认缓存路径
	cfrDir, err := getDefaultCFRDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cfrDir, 0755); err != nil {
		return "", fmt.Errorf("创建CFR目录失败: %v", err)
	}

	cfrJarPath := filepath.Join(cfrDir, fmt.Sprintf("cfr-%s.jar", Version))

	// 检查文件是否存在
	if _, err := os.Stat(cfrJarPath); err == nil {
		return cfrJarPath, nil
	}

	if shouldSkipDownload() {
		return "", fmt.Errorf("未找到本地 CFR JAR，且已禁用自动下载 (%s=1)\n%s", envSkipDownload, manualInstallHint(cfrJarPath))
	}

	// 下载CFR JAR
	color.Cyan("正在下载CFR反编译器 v%s...", Version)
	if err := m.downloadCFR(cfrJarPath); err != nil {
		return "", fmt.Errorf("%v\n%s", err, manualInstallHint(cfrJarPath))
	}

	color.Green("✓ CFR下载完成")
	return cfrJarPath, nil
}

// downloadCFR 下载CFR JAR文件
func (m *Manager) downloadCFR(destPath string) error {
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	downloadURL := getDownloadURL()
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载CFR失败 (url=%s): %v", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载CFR失败 (url=%s), 状态码: %d", downloadURL, resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer out.Close()

	// 显示下载进度
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}

	return nil
}

// Decompile 反编译class文件或JAR文件
func (m *Manager) Decompile(ctx context.Context, inputPath string, outputDir string) error {
	var cmd *exec.Cmd

	if m.useJar {
		// 使用Java运行CFR JAR
		// 添加 -Dfile.encoding=UTF-8 确保输出使用UTF-8编码，解决中文乱码问题
		args := []string{
			"-Dfile.encoding=UTF-8",
			"-jar", m.cfrPath,
			inputPath,
			"--outputdir", outputDir,
			"--caseinsensitivefs", "true", // Windows兼容
		}
		cmd = exec.CommandContext(ctx, m.javaPath, args...)
	} else {
		// 使用系统CFR命令
		cmd = exec.CommandContext(ctx, m.cfrPath, inputPath, "--outputdir", outputDir)
	}

	// 捕获输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}

	return nil
}

// DecompileWithOptions 使用自定义选项反编译
func (m *Manager) DecompileWithOptions(inputPath string, outputDir string, options map[string]string) error {
	var args []string

	if m.useJar {
		// 添加 -Dfile.encoding=UTF-8 确保输出使用UTF-8编码，解决中文乱码问题
		args = append(args, "-Dfile.encoding=UTF-8", "-jar", m.cfrPath)
	}

	args = append(args, inputPath, "--outputdir", outputDir)

	// 添加自定义选项
	for key, value := range options {
		if value == "" {
			args = append(args, "--"+key)
		} else {
			args = append(args, "--"+key, value)
		}
	}

	// Windows兼容性
	if runtime.GOOS == "windows" {
		args = append(args, "--caseinsensitivefs", "true")
	}

	var cmd *exec.Cmd
	if m.useJar {
		cmd = exec.Command(m.javaPath, args...)
	} else {
		cmd = exec.Command(m.cfrPath, args[1:]...) // 跳过jar参数
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}

	return nil
}

// GetVersion 获取CFR版本信息
func (m *Manager) GetVersion() (string, error) {
	var cmd *exec.Cmd

	if m.useJar {
		cmd = exec.Command(m.javaPath, "-jar", m.cfrPath, "--version")
	} else {
		cmd = exec.Command(m.cfrPath, "--version")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// CheckJavaInstallation 检查Java是否已安装
func CheckJavaInstallation() error {
	cmd := exec.Command("java", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Java未安装或不在PATH中\n请安装Java: https://www.java.com/\n错误: %v", err)
	}

	color.Green("✓ Java环境检测成功")
	// 输出Java版本信息
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		color.Cyan("  %s", strings.TrimSpace(lines[0]))
	}

	return nil
}
