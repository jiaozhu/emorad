# 🎯 Emorad

**Emorad** - 一个功能强大的 Java 反编译工具，专为 Spring Boot JAR、WAR 文件和 Tomcat 部署目录设计，支持跨平台运行。

> 名称来源: **E**xplore **M**ore **O**f **R**everse **A**nd **D**ecompile

## ✨ 功能特点

### 🚀 核心功能
- **全格式支持**: Spring Boot JAR、WAR、普通JAR、CLASS文件、Tomcat部署目录
- **智能处理**: 自动识别文件类型并选择最佳处理策略
- **嵌套JAR**: 完整支持Spring Boot的BOOT-INF/lib嵌套JAR结构
- **多核并发**: 充分利用多核CPU,显著提升反编译速度

### 🎨 用户体验
- **零配置**: 自动下载并管理CFR反编译器
- **跨平台**: 完美支持Windows、macOS、Linux
- **中文界面**: 详细的中文提示和错误信息
- **实时进度**: 彩色进度条显示,一目了然

### 📊 报告系统
- **HTML报告**: 精美的可视化报告,支持浏览器查看
- **JSON报告**: 机器可读的详细数据
- **实时统计**: 成功率、耗时、错误信息完整记录

## 📦 安装要求

### 基础要求
- **Java环境**: JDK 8 或更高版本
- **Go环境**: Go 1.21+ (仅编译时需要)

### 自动化安装
工具会自动下载并管理CFR反编译器,无需手动安装!

### 手动安装CFR (可选)
```bash
# macOS (使用 Homebrew)
brew install cfr-decompiler

# 或从官网下载
# https://www.benf.org/other/cfr/
```

## 🚀 快速开始

### 1. 编译项目

```bash
# 克隆项目
git clone https://github.com/jiaozhu/emorad.git
cd emorad

# 编译
go build -o emorad

# Windows用户
go build -o emorad.exe
```

### 2. 基本使用

```bash
# 反编译Spring Boot JAR
emorad app.jar

# 反编译WAR文件
emorad app.war

# 反编译Tomcat部署目录
emorad /path/to/tomcat/webapps/myapp

# 反编译单个CLASS文件
emorad MyClass.class
```

### 3. 高级选项

```bash
# 自定义输出目录
emorad -o /custom/output app.jar

# 调整并发数(默认使用所有CPU核心)
emorad -w 4 app.jar

# 在当前Tomcat部署目录中使用
cd /path/to/tomcat/webapps/myapp
emorad
```

## 📁 输出说明

### 目录结构
```
输出目录/
├── src/                    # 反编译的源代码(保持包结构)
│   └── com/example/
│       └── MyClass.java
└── reports/                # 反编译报告
    ├── report-20240101-120000.html
    └── report-20240101-120000.json
```

### 报告文件

#### HTML报告
- 📊 **可视化展示**: 精美的Web界面
- 📈 **统计图表**: 成功率、耗时等统计
- 🔍 **详细列表**: 每个文件的处理状态和错误信息
- 💻 **浏览器查看**: 双击即可打开

#### JSON报告
- 🔧 **机器可读**: 方便自动化处理
- 📝 **完整数据**: 所有处理结果的详细记录
- 🔗 **易于集成**: 可集成到CI/CD流程

## 💡 使用示例

### Spring Boot应用
```bash
# 反编译Spring Boot JAR,包括所有依赖
emorad myapp-0.0.1-SNAPSHOT.jar

# 输出目录: myapp-0.0.1-SNAPSHOT/src/
# - BOOT-INF/classes下的业务代码
# - BOOT-INF/lib下的依赖JAR(递归处理)
```

### Tomcat部署
```bash
# 方式1: 在部署目录中直接运行
cd /opt/tomcat/webapps/myapp
emorad

# 方式2: 指定部署目录
emorad /opt/tomcat/webapps/myapp

# 输出目录: /opt/tomcat/webapps/myapp/src/
```

### WAR文件
```bash
# 反编译WAR文件
emorad myapp.war

# 输出目录: myapp/src/
# - WEB-INF/classes下的业务代码
# - WEB-INF/lib下的依赖JAR
```

## 🔧 高级功能

### 自动CFR管理
工具会自动处理CFR反编译器:
1. ✅ 优先使用系统安装的`cfr-decompiler`命令
2. ✅ 如果没有,自动下载CFR JAR到`~/.emorad/cfr/`
3. ✅ 使用Java运行CFR JAR进行反编译

### 智能类型识别
工具自动识别输入类型:
- 📦 JAR文件 → JAR处理器(支持嵌套JAR)
- 📦 WAR文件 → WAR处理器
- 📄 CLASS文件 → CLASS处理器
- 📁 目录 → 目录处理器(自动检测Tomcat结构)

### 并发处理优化
- 🚀 默认使用所有CPU核心
- 🚀 智能任务分配
- 🚀 进度实时显示

## ⚙️ Windows支持

### 编译Windows版本
```bash
# 在任意平台编译Windows版本
GOOS=windows GOARCH=amd64 go build -o emorad.exe

# 或使用build脚本
./scripts/build-all.sh
```

### Windows使用
```cmd
# 命令提示符
emorad.exe app.jar

# PowerShell
.\emorad.exe app.jar

# 拖放支持
# 直接将JAR/WAR文件拖到emorad.exe图标上
```

## 🐛 故障排除

### Java环境问题
```bash
# 检查Java是否安装
java -version

# 如果未安装,请访问:
# https://www.java.com/
```

### CFR下载失败
```bash
# 手动下载CFR并放置到:
# ~/.emorad/cfr/cfr-0.152.jar

# 或安装系统CFR
brew install cfr-decompiler  # macOS
```

### 权限问题 (Linux/macOS)
```bash
# 添加执行权限
chmod +x emorad
```

## 📊 性能指标

| 项目 | 指标 |
|------|------|
| 并发处理 | 利用所有CPU核心 |
| 内存占用 | <100MB (小型项目) |
| 处理速度 | ~100-500 files/s |
| 支持大小 | 无限制 |

## 🤝 贡献

欢迎提交Issue和Pull Request!

## 📄 许可证

MIT License

---

**Powered by CFR Decompiler** - https://www.benf.org/other/cfr/ 