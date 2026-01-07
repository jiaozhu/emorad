package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeUnicodeEscapes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "中文转义序列",
			input:    `log.error("\u5bb9\u5668\u505c\u6b62\u6267\u884c\u5f02\u5e38", (Throwable)e);`,
			expected: `log.error("容器停止执行异常", (Throwable)e);`,
		},
		{
			name:     "混合内容",
			input:    `String msg = "Hello \u4e16\u754c World";`,
			expected: `String msg = "Hello 世界 World";`,
		},
		{
			name:     "无转义内容",
			input:    `String msg = "Hello World";`,
			expected: `String msg = "Hello World";`,
		},
		{
			name:     "多个转义序列",
			input:    `// \u6ce8\u91ca: \u8fd9\u662f\u4e2d\u6587`,
			expected: `// 注释: 这是中文`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeUnicodeEscapes(tt.input)
			if result != tt.expected {
				t.Errorf("DecodeUnicodeEscapes() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestProcessDirectoryUnicode(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "unicode_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试 Java 文件
	testFile := filepath.Join(tempDir, "Test.java")
	content := `package test;

public class Test {
    public void test() {
        log.error("\u5bb9\u5668\u505c\u6b62\u6267\u884c\u5f02\u5e38");
    }
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	// 处理目录
	processed, modified, err := ProcessDirectoryUnicode(tempDir)
	if err != nil {
		t.Fatalf("ProcessDirectoryUnicode() error = %v", err)
	}

	if processed != 1 {
		t.Errorf("processed = %d, want 1", processed)
	}
	if modified != 1 {
		t.Errorf("modified = %d, want 1", modified)
	}

	// 验证文件内容
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("读取结果文件失败: %v", err)
	}

	expected := `package test;

public class Test {
    public void test() {
        log.error("容器停止执行异常");
    }
}
`
	if string(result) != expected {
		t.Errorf("文件内容不匹配:\n得到: %s\n期望: %s", string(result), expected)
	}
}
