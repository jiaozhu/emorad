# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- 新增零参数自动扫描模式：在当前目录自动识别并反编译 JAR/WAR/CLASS 与 Tomcat/WebLogic（含 `APP-INF`）部署结构。
- 新增智能业务 JAR 保留策略：默认过滤常见公共依赖，同时保留命中应用特征词的业务 JAR。
- 新增配置文件机制：
  - `--config` 显式指定配置文件
  - 自动加载 `./.emorad.yaml` 或 `~/.emorad/config.yaml`
  - 支持 `common_jar_prefixes` / `app_jar_hints` / `jar_include`
- 新增日志文档与企业级示例配置：
  - `docs/LOGGING.md`
  - `docs/emorad.config.example.yaml`

### Changed
- 进度显示优化为单行横向进度条（0%-100%），减少大规模反编译时终端刷屏。
- 静默逐个 class 与逐个 JAR 的处理中输出，保留关键摘要与结构化事件。
- 参数体系优化：
  - 新增 `--exclude-mode (append|replace|none)`
  - `--no-default-exclude` 标记为兼容弃用
  - `--output` 默认路径按 `--idea-project` 自适应
- Makefile 输出改为专业简洁风格，去除 emoji 与装饰性符号。

### Fixed
- 修复 `skip-libs` 语义与行为不一致问题。
- 修复路径匹配跨平台兼容性问题（Windows 风格路径）。

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
