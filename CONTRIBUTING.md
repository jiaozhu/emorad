# 贡献指南

感谢您对 Emorad 项目的关注！我们欢迎各种形式的贡献。

## 如何贡献

### 报告问题

如果您发现了 bug 或有功能建议，请[创建 Issue](https://github.com/jiaozhu/emorad/issues/new)。

在提交 Issue 时，请尽量包含以下信息：

- **Bug 报告**：
  - 操作系统和版本
  - Go 版本（如果从源码编译）
  - Java 版本
  - 详细的重现步骤
  - 预期行为与实际行为
  - 相关的错误日志

- **功能建议**：
  - 清晰描述您希望的功能
  - 说明这个功能的使用场景
  - 如果可能，提供实现思路

### 提交代码

1. **Fork 仓库**

   ```bash
   git clone https://github.com/jiaozhu/emorad.git
   cd emorad
   ```

2. **创建分支**

   ```bash
   git checkout -b feature/your-feature-name
   # 或
   git checkout -b fix/your-bug-fix
   ```

3. **进行修改**

   - 确保代码符合项目的代码风格
   - 运行 `make fmt` 格式化代码
   - 运行 `make vet` 进行代码检查
   - 运行 `make test` 确保测试通过

4. **提交更改**

   ```bash
   git add .
   git commit -m "feat: 添加新功能描述"
   # 或
   git commit -m "fix: 修复问题描述"
   ```

   提交消息建议遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：
   - `feat:` 新功能
   - `fix:` Bug 修复
   - `docs:` 文档更新
   - `refactor:` 代码重构
   - `test:` 测试相关
   - `chore:` 构建/工具相关

5. **推送并创建 Pull Request**

   ```bash
   git push origin feature/your-feature-name
   ```

   然后在 GitHub 上创建 Pull Request。

## 开发环境设置

### 前置要求

- Go 1.21 或更高版本
- Java 8 或更高版本（运行时需要）
- Make（可选，用于构建）

### 构建项目

```bash
# 编译当前平台
make build

# 编译所有平台
make all

# 运行测试
make test
```

## 代码规范

- 使用 `gofmt` 格式化代码
- 遵循 Go 官方的[代码审查建议](https://go.dev/wiki/CodeReviewComments)
- 为公开的函数和类型添加文档注释
- 保持函数简洁，职责单一

## 许可证

通过向本项目贡献代码，您同意您的贡献将按照项目的 [MIT License](LICENSE) 进行许可。

## 联系方式

如有任何问题，欢迎通过以下方式联系：

- Issues: https://github.com/jiaozhu/emorad/issues
- Email: weijie@opsrelay.dev
