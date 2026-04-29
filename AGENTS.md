# Repository Guidelines

## 项目结构与模块组织
Emorad 是一个用于 Java 制品反编译的 Go 命令行项目。

- `cmd/emorad/main.go`：程序入口与 CLI 参数编排。
- `internal/`：私有实现包。
- `internal/decompile`：反编译主流程。
- `internal/cfr`：CFR 反编译器管理。
- `internal/processor`：结果后处理与项目结构辅助逻辑。
- `internal/report`：HTML/JSON 报告生成。
- `pkg/`：构建产物目录（`make build`/`make all` 输出）。
- `scripts/`：本地与发布相关脚本。
- `docs/`：部署与运维文档。

## 构建、测试与开发命令
优先使用 `Makefile` 目标，项目默认参数已内置。

- `make build`：构建当前平台二进制到 `pkg/emorad`。
- `make all`：清理后进行多平台交叉编译（macOS/Linux/Windows）。
- `make test`：运行全部 Go 测试（`go test -v ./...`）。
- `make vet`：运行静态检查（`go vet ./...`）。
- `make fmt`：格式化全部包（`go fmt ./...`）。
- `make clean`：清理 `pkg/` 等构建产物。

Go 直接构建示例：`go build -o pkg/emorad ./cmd/emorad`。

## 代码风格与命名约定
- 遵循 Go 标准格式，提交前执行 `make fmt`。
- 包名保持简短、小写、名词化（如 `processor`、`report`）。
- 导出标识符使用 `CamelCase`，内部辅助函数使用 `camelCase`。
- 保持函数职责单一，按包边界分离 CLI、业务流程与报告逻辑。

## 测试规范
- 测试框架：Go 内置 `testing`。
- 测试文件与实现同目录，命名为 `*_test.go`（示例：`internal/processor/unicode_test.go`）。
- 测试函数建议使用 `TestXxxBehavior` 风格，体现具体场景。
- 提交 PR 前至少运行 `make test`；修复缺陷时补充回归测试。

## 提交与 Pull Request 规范
- 提交信息遵循 Conventional Commits（当前历史常见：`feat:`、`fix:`、`docs:`、`ci:`、`chore:`）。
- 每次提交保持单一目的、变更原子化。
- PR 应包含：变更摘要、关联 Issue（如有）、测试证明（`make test`/`make vet`）、涉及 CLI 或报告展示时附示例输出或截图。

## 安全与配置提示
- 不要提交客户反编译产物、临时输出或任何密钥信息。
- 排查构建/运行问题前先确认本地版本满足要求（Go 1.21+、JDK 8+）。
