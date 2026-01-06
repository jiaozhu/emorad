# Emorad

**Emorad** - 一个功能强大的 Java 反编译工具，专为 Spring Boot JAR、WAR 文件和 Tomcat 部署目录设计，支持跨平台运行。

> 名称来源: **E**xplore **M**ore **O**f **R**everse **A**nd **D**ecompile

## 功能特点

### 核心功能
- **全格式支持**: Spring Boot JAR、WAR、普通JAR、CLASS文件、Tomcat部署目录
- **智能处理**: 自动识别文件类型并选择最佳处理策略
- **嵌套JAR**: 完整支持Spring Boot的BOOT-INF/lib嵌套JAR结构
- **多核并发**: 充分利用多核CPU，显著提升反编译速度

### 智能过滤
- **业务代码优先**: 自动跳过 Spring、Tomcat 等框架包，只反编译业务代码
- **包含过滤器**: 指定只处理特定包前缀的类
- **排除过滤器**: 跳过不需要的第三方包
- **JAR名称过滤**: 按关键字筛选 lib 目录下的 JAR
- **配置文件复制**: 自动复制配置文件到输出目录
- **跳过依赖**: 自动跳过 lib 目录下的依赖 JAR

### 用户体验
- **零配置**: 自动下载并管理CFR反编译器
- **跨平台**: 完美支持Windows、macOS、Linux
- **中文界面**: 详细的中文提示和错误信息
- **实时进度**: 进度条显示，一目了然

### 报告系统
- **HTML报告**: 可视化报告，支持浏览器查看
- **JSON报告**: 机器可读的详细数据
- **实时统计**: 成功率、耗时、错误信息完整记录

## 安装要求

### 基础要求
- **Java环境**: JDK 8 或更高版本
- **Go环境**: Go 1.21+ (仅编译时需要)

### 自动化安装
工具会自动下载并管理CFR反编译器，无需手动安装。

## 快速开始

### 编译项目

```bash
# 克隆项目
git clone https://github.com/jiaozhu/emorad.git
cd emorad

# 使用 Make 编译当前平台
make build

# 编译所有平台
make all

# 或直接使用 Go
go build -o pkg/emorad ./cmd/emorad
```

### 基本使用

```bash
# 反编译Spring Boot JAR（自动过滤框架包）
emorad app.jar

# 反编译WAR文件
emorad app.war

# 反编译Tomcat部署目录
emorad /path/to/tomcat/webapps/myapp

# 反编译单个CLASS文件
emorad MyClass.class
```

## 命令行参数

| 参数 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--output` | `-o` | 输出目录 | 当前目录下的 `src` 目录 |
| `--workers` | `-w` | 并发工作器数量 | CPU核心数 |
| `--include` | `-i` | 只处理匹配的包前缀，逗号分隔 | 无（处理所有） |
| `--exclude` | `-e` | 排除匹配的包前缀，追加到默认列表 | 无 |
| `--jar-include` | `-j` | 只处理名称包含指定关键字的 lib JAR | 无 |
| `--copy-resources` | `-r` | 复制配置文件到 resources 目录 | `false` |
| `--skip-libs` | - | 跳过 lib 目录下的依赖 JAR | `true` |
| `--no-default-exclude` | - | 不使用默认的框架包排除列表 | `false` |
| `--version` | `-v` | 显示版本信息 | - |
| `--help` | `-h` | 显示帮助信息 | - |

### 默认排除的框架包

工具默认会自动跳过以下框架包，只反编译业务代码：

```
org/springframework/  org/apache/       com/fasterxml/
org/hibernate/        org/mybatis/      ch/qos/logback/
org/slf4j/           com/google/        javax/
jakarta/             org/aspectj/       org/yaml/
com/zaxxer/          org/jboss/         io/netty/
com/alibaba/         org/thymeleaf/     org/bouncycastle/
```

## 使用示例

### 只反编译业务代码（推荐）

```bash
# 只反编译 com.mycompany 包下的代码
emorad -i "com.mycompany" app.jar

# 反编译多个业务包
emorad -i "com.mycompany,com.partner" app.jar
```

### 追加排除规则

```bash
# 在默认排除列表基础上，额外排除 com.thirdparty
emorad -e "com.thirdparty" app.jar
```

### 处理依赖库

```bash
# 禁用跳过依赖库，处理所有 JAR
emorad --skip-libs=false app.jar

# 不使用默认排除列表，只排除指定包
emorad --no-default-exclude -e "org.springframework" app.jar
```

### JAR 名称过滤

```bash
# 只反编译名称包含 "myapp" 或 "common" 的 lib JAR
emorad -j "myapp,common" app.jar

# 结合包含过滤使用
emorad -i "com.mycompany" -j "myapp" app.jar
```

### 复制配置文件

```bash
# 反编译并复制配置文件到 resources 目录
emorad -r app.jar

# 结合 JAR 过滤和配置文件复制
emorad -j "myapp" -r app.jar

# 完整示例：指定业务包、JAR过滤、复制配置文件
emorad -i "com.mycompany" -j "myapp,common" -r app.jar
```

### 自定义输出和并发

```bash
# 自定义输出目录
emorad -o /custom/output app.jar

# 调整并发数
emorad -w 4 app.jar
```

### Tomcat部署目录

```bash
# 方式1: 在部署目录中直接运行
cd /opt/tomcat/webapps/myapp
emorad

# 方式2: 指定部署目录
emorad /opt/tomcat/webapps/myapp

# 结合包含过滤
emorad -i "com.mycompany" /opt/tomcat/webapps/myapp
```

## 输出说明

### 目录结构

```
输出目录/
├── com/example/              # 反编译的源代码(保持包结构)
│   └── MyClass.java
├── resources/                # 配置文件(使用 -r 参数启用)
│   ├── application.yml
│   ├── application.properties
│   └── mapper/
│       └── UserMapper.xml
└── reports/                  # 反编译报告
    ├── report-20240101-120000.html
    └── report-20240101-120000.json
```

### 支持的配置文件类型

使用 `-r` 参数时，会自动复制以下类型的配置文件：

```
.properties  .yml  .yaml  .xml  .json
.conf  .config  .txt  .sql  .sh
```

### 报告文件

#### HTML报告
- **可视化展示**: 精美的Web界面
- **统计图表**: 成功率、耗时等统计
- **详细列表**: 每个文件的处理状态和错误信息
- **浏览器查看**: 双击即可打开

#### JSON报告
- **机器可读**: 方便自动化处理
- **完整数据**: 所有处理结果的详细记录
- **易于集成**: 可集成到CI/CD流程

## 项目结构

```
emorad/
├── cmd/emorad/           # 主程序入口
├── internal/
│   ├── cfr/              # CFR 反编译器管理
│   ├── decompile/        # 反编译逻辑
│   ├── processor/        # 文件处理器
│   └── report/           # 报告生成
├── docs/                 # 文档
├── pkg/                  # 编译输出
└── scripts/              # 脚本
```

## 编译构建

### 使用 Makefile（推荐）

```bash
# 编译当前平台
make build

# 编译所有平台
make all

# 编译特定平台
make darwin-arm64   # macOS Apple Silicon
make darwin-amd64   # macOS Intel
make linux-amd64    # Linux x86_64
make linux-arm64    # Linux ARM64
make windows-amd64  # Windows x86_64

# 编译文件输出到 pkg 目录
# 例如: pkg/emorad-darwin-arm64

# 清理构建产物
make clean

# 运行测试
make test

# 查看帮助
make help
```

### 手动编译

```bash
# 当前平台
go build -o pkg/emorad ./cmd/emorad

# 交叉编译
GOOS=linux GOARCH=amd64 go build -o pkg/emorad-linux-amd64 ./cmd/emorad
GOOS=darwin GOARCH=arm64 go build -o pkg/emorad-darwin-arm64 ./cmd/emorad
GOOS=windows GOARCH=amd64 go build -o pkg/emorad-windows-amd64.exe ./cmd/emorad
```

## 故障排除

### Java环境问题

```bash
# 检查Java是否安装
java -version

# 如果未安装，请访问:
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

## 性能指标

| 项目 | 指标 |
|------|------|
| 并发处理 | 利用所有CPU核心 |
| 内存占用 | <100MB (小型项目) |
| 处理速度 | ~100-500 files/s |
| 支持大小 | 无限制 |

## 贡献

欢迎提交Issue和Pull Request!

## 许可证

MIT License

---

**Powered by CFR Decompiler** - https://www.benf.org/other/cfr/