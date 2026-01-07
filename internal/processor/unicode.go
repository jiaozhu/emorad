package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// unicodeEscapePattern 匹配 Unicode 转义序列 \uXXXX
var unicodeEscapePattern = regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)

// DecodeUnicodeEscapes 将字符串中的 \uXXXX 转换为实际的 Unicode 字符
func DecodeUnicodeEscapes(input string) string {
	return unicodeEscapePattern.ReplaceAllStringFunc(input, func(match string) string {
		// 提取十六进制码点
		hex := match[2:] // 去掉 \u 前缀
		codePoint, err := strconv.ParseInt(hex, 16, 32)
		if err != nil {
			return match // 解析失败则保留原样
		}
		return string(rune(codePoint))
	})
}

// ProcessJavaFileUnicode 处理单个 Java 文件中的 Unicode 转义序列
func ProcessJavaFileUnicode(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	originalContent := string(content)
	decodedContent := DecodeUnicodeEscapes(originalContent)

	// 只有内容发生变化时才写入
	if decodedContent != originalContent {
		if err := os.WriteFile(filePath, []byte(decodedContent), 0644); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}
	}

	return nil
}

// ProcessDirectoryUnicode 递归处理目录中所有 Java 文件的 Unicode 转义序列
func ProcessDirectoryUnicode(dir string) (int, int, error) {
	processed := 0
	modified := 0

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 只处理 .java 文件
		if strings.ToLower(filepath.Ext(path)) != ".java" {
			return nil
		}

		processed++

		// 读取文件内容
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // 跳过读取失败的文件
		}

		originalContent := string(content)
		decodedContent := DecodeUnicodeEscapes(originalContent)

		// 只有内容发生变化时才写入
		if decodedContent != originalContent {
			if err := os.WriteFile(path, []byte(decodedContent), 0644); err != nil {
				return nil // 跳过写入失败的文件
			}
			modified++
		}

		return nil
	})

	return processed, modified, err
}
