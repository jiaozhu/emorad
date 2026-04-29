package processor

import (
	"path/filepath"
	"testing"
)

func TestShouldProcessJar_SkipLibsBehavior(t *testing.T) {
	cfg := NewDefaultFilterConfig()
	cfg.SkipLibs = true

	if cfg.ShouldProcessJar("/tmp/app/BOOT-INF/lib/spring-core.jar") {
		t.Fatalf("expected lib jar to be skipped when skip-libs=true and jar-include is empty")
	}

	if !cfg.ShouldProcessJar("/tmp/app/BOOT-INF/classes/app.jar") {
		t.Fatalf("expected non-lib jar path to be processed")
	}
}

func TestShouldProcessJar_WithJarIncludeAndSkipLibs(t *testing.T) {
	cfg := NewDefaultFilterConfig()
	cfg.SkipLibs = true
	cfg.JarIncludes = []string{"myapp"}
	cfg.JarMatchMode = "contains"

	if !cfg.ShouldProcessJar("/tmp/app/BOOT-INF/lib/myapp-common-1.0.jar") {
		t.Fatalf("expected matched lib jar to be processed")
	}

	if cfg.ShouldProcessJar("/tmp/app/BOOT-INF/lib/spring-core.jar") {
		t.Fatalf("expected unmatched lib jar to be skipped")
	}
}

func TestShouldProcessJar_WindowsStylePath(t *testing.T) {
	cfg := NewDefaultFilterConfig()
	cfg.SkipLibs = true

	if cfg.ShouldProcessJar(`C:\tmp\app\BOOT-INF\lib\spring-core.jar`) {
		t.Fatalf("expected Windows-style lib jar path to be recognized and skipped")
	}
}

func TestExtractPackageName_WindowsStylePath(t *testing.T) {
	pkg := ExtractPackageName(`C:\tmp\x\BOOT-INF\classes\com\example\Demo.class`)
	if pkg != "com/example" {
		t.Fatalf("unexpected package name: %s", pkg)
	}
}

func TestShouldProcessClass_FilterDefaultFramework(t *testing.T) {
	cfg := NewDefaultFilterConfig()
	base := "/tmp/app"
	classPath := filepath.Join(base, "BOOT-INF", "classes", "org", "springframework", "A.class")
	if cfg.ShouldProcessClass(classPath, base) {
		t.Fatalf("expected framework class to be filtered by default excludes")
	}
}

func TestShouldProcessClass_BusinessClassInTomcat(t *testing.T) {
	cfg := NewDefaultFilterConfig()
	base := "/tmp/tomcat/webapps/myapp"
	classPath := filepath.Join(base, "WEB-INF", "classes", "com", "acme", "App.class")
	if !cfg.ShouldProcessClass(classPath, base) {
		t.Fatalf("expected business class to be processed")
	}
}

func TestShouldProcessJar_AutoBusinessJarInWebInfLib(t *testing.T) {
	cfg := NewDefaultFilterConfig()
	cfg.SkipLibs = true
	cfg.AppJarHints = []string{"sso", "client"}

	if !cfg.ShouldProcessJar("/tmp/app/WEB-INF/lib/sso_client_185.07bak.jar") {
		t.Fatalf("expected business jar in WEB-INF/lib to be processed by app hints")
	}
}

func TestShouldProcessJar_AutoCommonJarFiltered(t *testing.T) {
	cfg := NewDefaultFilterConfig()
	cfg.SkipLibs = true
	cfg.AppJarHints = []string{"sso", "client"}

	if cfg.ShouldProcessJar("/tmp/app/WEB-INF/lib/spring-core-3.0.5.RELEASE.jar") {
		t.Fatalf("expected common framework jar to be filtered")
	}
}
