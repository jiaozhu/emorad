# 🚀 Emorad 部署和使用指南

## ✅ 问题修复说明

**修复的问题**: 之前版本在处理包含多个Spring Boot JAR文件的目录时，无法正确识别和反编译JAR文件。

**修复内容**:
- ✅ 增强了目录处理器，现在能够正确识别目录中的所有JAR/WAR文件
- ✅ 添加了扫描结果统计，清晰显示找到的文件数量
- ✅ 优化了输出目录排除逻辑，避免重复处理
- ✅ 改进了错误提示，当没有找到文件时会明确告知

## 📦 Windows部署步骤

### 1. 获取可执行文件

从 `build` 目录复制 `emorad-windows-amd64.exe` 到你的Windows系统。

```
build/emorad-windows-amd64.exe  →  复制到 Windows 系统
```

### 2. 放置可执行文件

建议将文件重命名并放置到以下位置之一：

**方式1: 放到PATH目录** (推荐)
```cmd
# 复制到 C:\Windows\System32\ (需要管理员权限)
copy emorad-windows-amd64.exe C:\Windows\System32\emorad.exe

# 或者放到用户目录
mkdir %USERPROFILE%\bin
copy emorad-windows-amd64.exe %USERPROFILE%\bin\emorad.exe
# 然后将 %USERPROFILE%\bin 添加到PATH环境变量
```

**方式2: 放到项目目录**
```cmd
# 直接放到包含JAR文件的目录
copy emorad-windows-amd64.exe D:\projects\myapp\emorad.exe
```

### 3. 验证安装

```cmd
# 方式1安装后
emorad --version

# 方式2安装后
D:\projects\myapp\emorad.exe --version
```

## 🎯 使用示例

### 场景1: 反编译目录中的所有JAR文件

假设你有多个 Spring Boot JAR 文件在 `D:\projects\myapp` 目录中：

```cmd
# 切换到目录
cd D:\projects\myapp

# 执行反编译
emorad.exe .

# 或者指定完整路径
emorad.exe D:\projects\myapp
```

**预期输出**:
```
🚀 开始反编译...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📦 初始化反编译器...
✓ Java环境检测成功
✓ 使用CFR JAR: C:\Users\YourName\.emorad\cfr\cfr-0.152.jar
📁 检测到目录,使用目录处理器
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

正在处理目录: D:\projects\myapp
📊 扫描结果: 4个JAR, 0个WAR, 0个CLASS文件
处理JAR文件: app-core-0.0.1.jar
处理JAR文件: app-web-0.0.1.jar
处理JAR文件: app-service-0.0.1.jar
处理JAR文件: app-api-0.0.1.jar
...
反编译进度: 100.0% (1234/1234)

✓ 反编译完成！
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
              反编译报告摘要
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📂 输入路径: D:\projects\myapp
📁 输出路径: D:\projects\myapp\src
⏱️  总耗时: 45.23 秒

📊 文件统计:
   • 总文件数: 1234
   • 成功数量: 1230
   • 失败数量: 4
   • 成功率: 99.68%

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📄 详细报告已保存到: D:\projects\myapp\src\reports/
```

### 场景2: 反编译单个JAR文件

```cmd
# 反编译指定JAR
emorad.exe myapp-0.0.1.jar

# 指定输出目录
emorad.exe -o D:\decompiled myapp-0.0.1.jar
```

### 场景3: 反编译WAR文件

```cmd
# 反编译WAR文件
emorad.exe myapp.war

# 自定义输出和并发数
emorad.exe -o D:\output -w 8 myapp.war
```

## 📁 输出结构

反编译后的目录结构:

```
D:\projects\myapp\
├── src/                                    # 反编译的源代码
│   ├── com/
│   │   └── example/
│   │       └── MyClass.java
│   └── reports/                            # 详细报告
│       ├── report-20251108-113000.html     # 可视化HTML报告
│       └── report-20251108-113000.json     # 详细JSON数据
├── app-core-0.0.1.jar                      # 原始JAR文件
├── app-web-0.0.1.jar
├── app-service-0.0.1.jar
└── app-api-0.0.1.jar
```

## 🔍 查看报告

### HTML报告
```cmd
# 在浏览器中打开HTML报告
start src\reports\report-20251108-113000.html
```

HTML报告包含:
- 📊 可视化统计图表
- 📋 每个文件的处理状态
- ⏱️ 详细的耗时信息
- ❌ 失败文件的错误信息

### JSON报告
可用于自动化处理和数据分析:
```json
{
  "inputPath": "D:\\projects\\myapp",
  "outputPath": "D:\\projects\\myapp\\src",
  "totalFiles": 1234,
  "successCount": 1230,
  "failureCount": 4,
  "results": [...]
}
```

## ⚙️ 高级选项

```cmd
# 自定义并发数(默认使用所有CPU核心)
emorad.exe -w 4 .

# 自定义输出目录
emorad.exe -o D:\output .

# 组合使用
emorad.exe -o D:\decompiled -w 8 .

# 查看帮助
emorad.exe --help

# 查看版本
emorad.exe --version
```

## 🐛 常见问题

### 1. 提示"Java未安装"

**解决方法**:
```cmd
# 检查Java是否安装
java -version

# 如果没有安装,下载安装JDK
# https://www.oracle.com/java/technologies/downloads/
```

### 2. 找不到任何文件

**可能原因**:
- JAR文件在子目录中(程序会递归扫描,应该能找到)
- 文件扩展名不是.jar或.war
- 文件在输出目录src下(会自动排除)

**解决方法**:
```cmd
# 检查文件是否存在
dir /s *.jar

# 直接指定JAR文件
emorad.exe path\to\your.jar
```

### 3. CFR下载失败

**解决方法**:
```cmd
# 手动下载CFR并放置到:
# C:\Users\你的用户名\.emorad\cfr\cfr-0.152.jar

# 或安装系统CFR(如果有brew for Windows)
brew install cfr-decompiler
```

### 4. 权限问题

**解决方法**:
```cmd
# 以管理员身份运行命令提示符
# 右键点击"命令提示符" -> "以管理员身份运行"
```

## 📊 性能优化建议

### 大型项目优化

```cmd
# 增加并发数(根据CPU核心数)
emorad.exe -w 16 .

# 使用SSD存储输出目录
emorad.exe -o D:\SSD\output .
```

### 批量处理

创建批处理脚本 `batch-decompile.bat`:
```batch
@echo off
setlocal enabledelayedexpansion

set EMORAD=emorad.exe
set OUTPUT_BASE=D:\decompiled

for %%f in (*.jar) do (
    echo Processing %%f...
    %EMORAD% -o "%OUTPUT_BASE%\%%~nf" "%%f"
)

echo All done!
pause
```

## 💡 最佳实践

1. **首次使用**: 先在小项目上测试
2. **大型项目**: 使用 `-w` 参数调整并发数
3. **CI/CD集成**: 使用JSON报告进行自动化分析
4. **保留原始文件**: 反编译不会修改原始JAR/WAR文件
5. **定期清理**: 删除不需要的反编译结果释放空间

## 📞 支持

遇到问题?
- 📧 提交Issue: https://github.com/jiaozhu/emorad/issues
- 📖 查看文档: README.md
- 🔍 查看日志: src/reports/ 目录下的报告文件
