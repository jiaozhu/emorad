package main

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
)

const consoleWidth = 80

// DecompileResult 表示单个文件的反编译结果
type DecompileResult struct {
	ClassName   string    `json:"className"`
	PackageName string    `json:"packageName"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	TimeTaken   float64   `json:"timeTaken"`
	TimeStamp   time.Time `json:"timestamp"`
}

// DecompileReport 表示整体反编译报告
type DecompileReport struct {
	InputPath     string            `json:"inputPath"`
	OutputPath    string            `json:"outputPath"`
	StartTime     time.Time         `json:"startTime"`
	EndTime       time.Time         `json:"endTime"`
	TotalFiles    int32             `json:"totalFiles"`    // 已处理的文件数
	ExpectedFiles int32             `json:"expectedFiles"` // 预期要处理的总文件数
	SuccessCount  int32             `json:"successCount"`
	FailureCount  int32             `json:"failureCount"`
	Results       []DecompileResult `json:"results"`
	mu            sync.Mutex        // 保护Results切片
}

// NewDecompileReport 创建新的反编译报告
func NewDecompileReport(inputPath, outputPath string) *DecompileReport {
	return &DecompileReport{
		InputPath:  inputPath,
		OutputPath: outputPath,
		StartTime:  time.Now(),
		Results:    make([]DecompileResult, 0),
	}
}

// AddResult 添加单个反编译结果并更新进度
func (r *DecompileReport) AddResult(result DecompileResult) {
	if result.Success {
		atomic.AddInt32(&r.SuccessCount, 1)
	} else {
		atomic.AddInt32(&r.FailureCount, 1)
	}

	// 线程安全地添加结果
	r.mu.Lock()
	r.Results = append(r.Results, result)
	r.mu.Unlock()

	completed := atomic.AddInt32(&r.TotalFiles, 1)
	expected := atomic.LoadInt32(&r.ExpectedFiles)

	// 计算并显示进度
	if expected > 0 {
		progress := float64(completed) / float64(expected) * 100
		fmt.Printf("\r反编译进度: %.1f%% (%d/%d)", progress, completed, expected)
	}
}

// GetTotalExpectedFiles 获取预期总文件数
func (r *DecompileReport) GetTotalExpectedFiles() int32 {
	return atomic.LoadInt32(&r.ExpectedFiles)
}

// SetTotalExpectedFiles 设置预期总文件数
func (r *DecompileReport) SetTotalExpectedFiles(total int32) {
	atomic.StoreInt32(&r.ExpectedFiles, total)
}

// AddExpectedFiles 增加预期文件数(用于嵌套JAR)
func (r *DecompileReport) AddExpectedFiles(count int32) {
	atomic.AddInt32(&r.ExpectedFiles, count)
}

// GenerateReport 生成最终报告
func (r *DecompileReport) GenerateReport() error {
	r.EndTime = time.Now()
	duration := r.EndTime.Sub(r.StartTime)
	successCount := atomic.LoadInt32(&r.SuccessCount)
	failureCount := atomic.LoadInt32(&r.FailureCount)
	totalFiles := atomic.LoadInt32(&r.TotalFiles)

	// 清除进度显示的行
	fmt.Print("\r" + strings.Repeat(" ", consoleWidth) + "\r")

	// 打印摘要报告
	color.Green("\n✓ 反编译完成！")
	fmt.Printf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
              反编译报告摘要
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📂 输入路径: %s
📁 输出路径: %s
⏱️  总耗时: %.2f 秒

📊 文件统计:
   • 总文件数: %d
   • 成功数量: %d
   • 失败数量: %d
   • 成功率: %.2f%%

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,
		r.InputPath,
		r.OutputPath,
		duration.Seconds(),
		totalFiles,
		successCount,
		failureCount,
		getSuccessRate(successCount, totalFiles))

	// 生成详细报告文件
	if err := r.saveDetailedReports(); err != nil {
		color.Yellow("⚠️  保存详细报告失败: %v", err)
	} else {
		color.Cyan("📄 详细报告已保存到: %s/reports/", r.OutputPath)
	}

	return nil
}

// saveDetailedReports 保存详细的JSON和HTML报告
func (r *DecompileReport) saveDetailedReports() error {
	reportsDir := filepath.Join(r.OutputPath, "reports")
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return err
	}

	timestamp := r.StartTime.Format("20060102-150405")

	// 保存JSON报告
	jsonPath := filepath.Join(reportsDir, fmt.Sprintf("report-%s.json", timestamp))
	if err := r.saveJSONReport(jsonPath); err != nil {
		return err
	}

	// 保存HTML报告
	htmlPath := filepath.Join(reportsDir, fmt.Sprintf("report-%s.html", timestamp))
	if err := r.saveHTMLReport(htmlPath); err != nil {
		return err
	}

	return nil
}

// saveJSONReport 保存JSON格式报告
func (r *DecompileReport) saveJSONReport(path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// saveHTMLReport 保存HTML格式报告
func (r *DecompileReport) saveHTMLReport(path string) error {
	successCount := atomic.LoadInt32(&r.SuccessCount)
	failureCount := atomic.LoadInt32(&r.FailureCount)
	totalFiles := atomic.LoadInt32(&r.TotalFiles)
	duration := r.EndTime.Sub(r.StartTime)

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>反编译报告 - %s</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; border-radius: 8px 8px 0 0; }
        .header h1 { font-size: 28px; margin-bottom: 10px; }
        .header .subtitle { opacity: 0.9; font-size: 14px; }
        .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; padding: 30px; }
        .stat-card { background: #f8f9fa; padding: 20px; border-radius: 8px; border-left: 4px solid #667eea; }
        .stat-card .label { color: #6c757d; font-size: 14px; margin-bottom: 8px; }
        .stat-card .value { font-size: 32px; font-weight: bold; color: #212529; }
        .stat-card.success { border-left-color: #28a745; }
        .stat-card.success .value { color: #28a745; }
        .stat-card.failure { border-left-color: #dc3545; }
        .stat-card.failure .value { color: #dc3545; }
        .details { padding: 0 30px 30px; }
        .details h2 { margin-bottom: 20px; color: #212529; }
        .table-wrapper { overflow-x: auto; }
        table { width: 100%%; border-collapse: collapse; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #dee2e6; }
        th { background: #f8f9fa; font-weight: 600; color: #495057; }
        tr:hover { background: #f8f9fa; }
        .status { display: inline-block; padding: 4px 12px; border-radius: 12px; font-size: 12px; font-weight: 600; }
        .status.success { background: #d4edda; color: #155724; }
        .status.failure { background: #f8d7da; color: #721c24; }
        .error-msg { color: #dc3545; font-size: 12px; max-width: 300px; overflow: hidden; text-overflow: ellipsis; }
        .footer { padding: 20px 30px; border-top: 1px solid #dee2e6; color: #6c757d; font-size: 14px; text-align: center; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎯 反编译报告</h1>
            <div class="subtitle">生成时间: %s</div>
        </div>

        <div class="summary">
            <div class="stat-card">
                <div class="label">总文件数</div>
                <div class="value">%d</div>
            </div>
            <div class="stat-card success">
                <div class="label">成功</div>
                <div class="value">%d</div>
            </div>
            <div class="stat-card failure">
                <div class="label">失败</div>
                <div class="value">%d</div>
            </div>
            <div class="stat-card">
                <div class="label">成功率</div>
                <div class="value">%.1f%%</div>
            </div>
            <div class="stat-card">
                <div class="label">耗时</div>
                <div class="value">%.1fs</div>
            </div>
        </div>

        <div class="details">
            <h2>📋 处理详情</h2>
            <div class="table-wrapper">
                <table>
                    <thead>
                        <tr>
                            <th>文件名</th>
                            <th>包名</th>
                            <th>状态</th>
                            <th>耗时(秒)</th>
                            <th>错误信息</th>
                        </tr>
                    </thead>
                    <tbody>`,
		r.StartTime.Format("2006-01-02 15:04:05"),
		r.StartTime.Format("2006-01-02 15:04:05"),
		totalFiles,
		successCount,
		failureCount,
		getSuccessRate(successCount, totalFiles),
		duration.Seconds())

	// 添加每个结果的行
	for _, result := range r.Results {
		status := "success"
		statusText := "成功"
		errorMsg := "-"
		if !result.Success {
			status = "failure"
			statusText = "失败"
			errorMsg = html.EscapeString(result.Error)
		}

		htmlContent += fmt.Sprintf(`
                        <tr>
                            <td>%s</td>
                            <td>%s</td>
                            <td><span class="status %s">%s</span></td>
                            <td>%.3f</td>
                            <td><div class="error-msg">%s</div></td>
                        </tr>`,
			html.EscapeString(result.ClassName),
			html.EscapeString(result.PackageName),
			status,
			statusText,
			result.TimeTaken,
			errorMsg)
	}

	htmlContent += fmt.Sprintf(`
                    </tbody>
                </table>
            </div>
        </div>

        <div class="footer">
            <p>📂 输入: %s</p>
            <p>📁 输出: %s</p>
            <p>Powered by Emorad - Explore More Of Reverse And Decompile</p>
        </div>
    </div>
</body>
</html>`, r.InputPath, r.OutputPath)

	return os.WriteFile(path, []byte(htmlContent), 0644)
}

// 辅助函数
func getSuccessRate(success, total int32) float64 {
	if total == 0 {
		return 0
	}
	return float64(success) / float64(total) * 100
}
