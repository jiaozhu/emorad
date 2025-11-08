package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
)

const (
	// CFR下载地址
	CFR_DOWNLOAD_URL = "https://github.com/leibnitz27/cfr/releases/download/0.152/cfr-0.152.jar"
	CFR_VERSION      = "0.152"
)

// CFRManager 管理CFR反编译器
type CFRManager struct {
	cfrPath    string // CFR JAR文件路径或命令路径
	useJar     bool   // 是否使用JAR文件
	javaPath   string // Java命令路径
	workingDir string // 工作目录
}

// NewCFRManager 创建CFR管理器
func NewCFRManager() (*CFRManager, error) {
	manager := &CFRManager{}

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

// ensureCFRJar 确保CFR JAR文件存在,如果不存在则下载
func (m *CFRManager) ensureCFRJar() (string, error) {
	// 确定CFR存储目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户主目录: %v", err)
	}

	cfrDir := filepath.Join(homeDir, ".emorad", "cfr")
	if err := os.MkdirAll(cfrDir, 0755); err != nil {
		return "", fmt.Errorf("创建CFR目录失败: %v", err)
	}

	cfrJarPath := filepath.Join(cfrDir, fmt.Sprintf("cfr-%s.jar", CFR_VERSION))

	// 检查文件是否存在
	if _, err := os.Stat(cfrJarPath); err == nil {
		return cfrJarPath, nil
	}

	// 下载CFR JAR
	color.Cyan("正在下载CFR反编译器 v%s...", CFR_VERSION)
	if err := m.downloadCFR(cfrJarPath); err != nil {
		return "", err
	}

	color.Green("✓ CFR下载完成")
	return cfrJarPath, nil
}

// downloadCFR 下载CFR JAR文件
func (m *CFRManager) downloadCFR(destPath string) error {
	resp, err := http.Get(CFR_DOWNLOAD_URL)
	if err != nil {
		return fmt.Errorf("下载CFR失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载CFR失败,状态码: %d", resp.StatusCode)
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
func (m *CFRManager) Decompile(inputPath string, outputDir string) error {
	var cmd *exec.Cmd

	if m.useJar {
		// 使用Java运行CFR JAR
		args := []string{
			"-jar", m.cfrPath,
			inputPath,
			"--outputdir", outputDir,
			"--caseinsensitivefs", "true", // Windows兼容
		}
		cmd = exec.Command(m.javaPath, args...)
	} else {
		// 使用系统CFR命令
		cmd = exec.Command(m.cfrPath, inputPath, "--outputdir", outputDir)
	}

	// 捕获输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}

	return nil
}

// DecompileWithOptions 使用自定义选项反编译
func (m *CFRManager) DecompileWithOptions(inputPath string, outputDir string, options map[string]string) error {
	var args []string

	if m.useJar {
		args = append(args, "-jar", m.cfrPath)
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
func (m *CFRManager) GetVersion() (string, error) {
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
