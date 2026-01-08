package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectConfig IDEA 项目生成配置
type ProjectConfig struct {
	ProjectName string // 项目名称
	OutputDir   string // 输出根目录
	SrcDir      string // 源代码目录
	LibsDir     string // 依赖库目录
}

// GenerateIDEAProject 生成完整的 IDEA 项目结构
func GenerateIDEAProject(config *ProjectConfig) error {
	ideaDir := filepath.Join(config.OutputDir, ".idea")
	if err := os.MkdirAll(ideaDir, 0755); err != nil {
		return fmt.Errorf("创建 .idea 目录失败: %v", err)
	}

	// 生成 .iml 模块文件
	if err := generateIMLFile(config); err != nil {
		return err
	}

	// 生成 modules.xml
	if err := generateModulesXML(config); err != nil {
		return err
	}

	// 生成 misc.xml (JDK 配置)
	if err := generateMiscXML(config); err != nil {
		return err
	}

	return nil
}

// generateIMLFile 生成 .iml 模块文件
func generateIMLFile(config *ProjectConfig) error {
	// 收集所有 JAR 依赖
	var jarEntries strings.Builder
	libsDir := filepath.Join(config.OutputDir, "libs")

	if _, err := os.Stat(libsDir); err == nil {
		err = filepath.Walk(libsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".jar") {
				relPath, _ := filepath.Rel(config.OutputDir, path)
				relPath = filepath.ToSlash(relPath)
				jarEntries.WriteString(fmt.Sprintf(`      <root url="jar://$MODULE_DIR$/%s!/" />
`, relPath))
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	imlContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<module type="JAVA_MODULE" version="4">
  <component name="NewModuleRootManager" inherit-compiler-output="true">
    <exclude-output />
    <content url="file://$MODULE_DIR$">
      <sourceFolder url="file://$MODULE_DIR$/src" isTestSource="false" />
      <excludeFolder url="file://$MODULE_DIR$/reports" />
    </content>
    <orderEntry type="inheritedJdk" />
    <orderEntry type="sourceFolder" forTests="false" />
    <orderEntry type="module-library">
      <library>
        <CLASSES>
%s        </CLASSES>
        <JAVADOC />
        <SOURCES />
      </library>
    </orderEntry>
  </component>
</module>
`, jarEntries.String())

	imlPath := filepath.Join(config.OutputDir, config.ProjectName+".iml")
	return os.WriteFile(imlPath, []byte(imlContent), 0644)
}

// generateModulesXML 生成 modules.xml
func generateModulesXML(config *ProjectConfig) error {
	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<project version="4">
  <component name="ProjectModuleManager">
    <modules>
      <module fileurl="file://$PROJECT_DIR$/%s.iml" filepath="$PROJECT_DIR$/%s.iml" />
    </modules>
  </component>
</project>
`, config.ProjectName, config.ProjectName)

	xmlPath := filepath.Join(config.OutputDir, ".idea", "modules.xml")
	return os.WriteFile(xmlPath, []byte(content), 0644)
}

// generateMiscXML 生成 misc.xml (JDK 配置)
func generateMiscXML(config *ProjectConfig) error {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<project version="4">
  <component name="ProjectRootManager" version="2" languageLevel="JDK_11" default="true" project-jdk-name="11" project-jdk-type="JavaSDK">
    <output url="file://$PROJECT_DIR$/out" />
  </component>
</project>
`
	xmlPath := filepath.Join(config.OutputDir, ".idea", "misc.xml")
	return os.WriteFile(xmlPath, []byte(content), 0644)
}

// CopyLibJars 复制依赖 JAR 到 libs 目录
func CopyLibJars(jarPaths []string, outputDir string) (int, error) {
	libsDir := filepath.Join(outputDir, "libs")
	if err := os.MkdirAll(libsDir, 0755); err != nil {
		return 0, fmt.Errorf("创建 libs 目录失败: %v", err)
	}

	copied := 0
	for _, jarPath := range jarPaths {
		jarName := filepath.Base(jarPath)
		destPath := filepath.Join(libsDir, jarName)

		// 跳过已存在的文件
		if _, err := os.Stat(destPath); err == nil {
			continue
		}

		srcFile, err := os.Open(jarPath)
		if err != nil {
			continue
		}

		destFile, err := os.Create(destPath)
		if err != nil {
			srcFile.Close()
			continue
		}

		if _, err := destFile.ReadFrom(srcFile); err == nil {
			copied++
		}

		srcFile.Close()
		destFile.Close()
	}

	return copied, nil
}
