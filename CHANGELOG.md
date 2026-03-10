# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [1.0.4] - 2026-03-10

### Added
- 引入全链路 context 取消能力：支持 Ctrl+C / SIGTERM 优雅中断。
- 报告新增任务状态（completed/cancelled）。
- 新增嵌套 JAR 策略与深度控制：
  - `--nested-jar-strategy` (`skip|filtered|full`)
  - `--max-jar-depth`
- 新增错误模型字段：`stage` / `errorCode` / `filePath`。
- 报告新增失败按阶段汇总（`stageFailures`）。
- 新增结构化日志输出：`--log-format text|json`。
- 报告新增阶段耗时统计（`stageDurations`）。

### Changed
- Release workflow 对齐 emorash 风格（verify/build/release 三阶段）。
- 发布产物扩展为多平台：darwin/linux/windows + amd64/arm64。
- Release 自动生成更新说明（changelog + related issues + generated notes）。

### Fixed
- 构建/发布链路中的多分支冲突已统一整合，Issue #3/#4 能力已在主线生效。

[Unreleased]: https://github.com/jiaozhu/emorad/compare/v1.0.4...HEAD
[1.0.4]: https://github.com/jiaozhu/emorad/compare/v1.0.3...v1.0.4
