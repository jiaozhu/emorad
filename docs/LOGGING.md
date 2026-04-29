# Logging Guide

本文档说明 Emorad 在 `--log-format=json` 模式下的结构化日志事件，便于接入 ELK、ClickHouse、Loki 等平台。

## 启用方式

```bash
emorad app.jar --log-format json
```

每行输出一条 JSON，基础字段：
- `ts`: RFC3339 时间戳
- `level`: `info` / `warn` / `error`
- `event`: 事件名

## 核心流程事件

- `start_decompile`: 开始任务（含 `input`、`workers`）
- `processor_selected`: 处理器选择（`type`: `directory|jar|war|class`）
- `process_started`: 进入主处理阶段
- `process_completed`: 主处理完成（`duration_seconds`）
- `decompile_finished`: 任务结束（`status`）
- `decompile_failed`: 任务失败（`error`）
- `decompile_cancelled`: 任务取消（`error`）

## 扫描/过滤/资源事件

- `directory_scan_completed`: 目录扫描完成（`jar_count`、`war_count`、`class_count`）
- `class_filter_applied`: class 过滤结果（`before`、`after`）
- `resources_copied`: 配置文件复制数量（`count`）
- `lib_jars_copied`: 依赖 JAR 复制数量（`count`）
- `unicode_postprocess_completed`: Unicode 后处理统计（`processed_files`、`modified_files`、`duration_seconds`）

## 嵌套 JAR 与失败事件

- `nested_jar_skipped`: 嵌套 JAR 被跳过（`jar`、`depth`）
- `nested_jar_depth_limit_reached`: 达到深度上限（`max_depth`、`skipped`）
- `nested_jar_failed`: 嵌套 JAR 处理失败（`jar`、`depth`、`error`）
- `jar_process_failed`: JAR 处理失败（`jar`、`error`）
- `war_process_failed`: WAR 处理失败（`war`、`error`）
- `class_decompile_failed`: class 反编译失败（`class_file`、`error`）

## 示例

```json
{"ts":"2026-04-29T10:00:00+08:00","level":"info","event":"start_decompile","input":"/data/app.jar","workers":8}
{"ts":"2026-04-29T10:00:00+08:00","level":"info","event":"processor_selected","type":"jar"}
{"ts":"2026-04-29T10:00:01+08:00","level":"info","event":"class_filter_applied","before":18230,"after":2410}
{"ts":"2026-04-29T10:00:05+08:00","level":"error","event":"class_decompile_failed","class_file":"Foo.class","error":"exit status 1"}
{"ts":"2026-04-29T10:00:12+08:00","level":"info","event":"decompile_finished","status":"completed"}
```

## 接入建议

- 用 `event` 作为主聚合维度，`level` 作为告警过滤维度。
- 对失败事件（`*_failed`）建立告警规则，并保留 `error` 原文用于排障。
- 对 `process_completed.duration_seconds`、`class_filter_applied.before/after` 建立趋势看板。
