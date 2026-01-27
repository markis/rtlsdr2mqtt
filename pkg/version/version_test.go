package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestInfo(t *testing.T) {
	info := Info()

	if info == "" {
		t.Error("Info() returned empty string")
	}

	if !strings.Contains(info, ApplicationName) {
		t.Errorf("Info() should contain application name '%s', got: %s", ApplicationName, info)
	}

	if !strings.Contains(info, Version) {
		t.Errorf("Info() should contain version '%s', got: %s", Version, info)
	}

	expected := ApplicationName + " " + Version
	if info != expected {
		t.Errorf("Expected '%s', got '%s'", expected, info)
	}
}

func TestBuildInfo(t *testing.T) {
	buildInfo := BuildInfo()

	if buildInfo == "" {
		t.Error("BuildInfo() returned empty string")
	}

	if !strings.Contains(buildInfo, ApplicationName) {
		t.Errorf("BuildInfo() should contain application name '%s', got: %s", ApplicationName, buildInfo)
	}

	if !strings.Contains(buildInfo, Version) {
		t.Errorf("BuildInfo() should contain version '%s', got: %s", Version, buildInfo)
	}

	if !strings.Contains(buildInfo, runtime.Version()) {
		t.Errorf("BuildInfo() should contain runtime version '%s', got: %s", runtime.Version(), buildInfo)
	}

	if !strings.Contains(buildInfo, "built with") {
		t.Errorf("BuildInfo() should contain 'built with', got: %s", buildInfo)
	}
}

func TestConstants(t *testing.T) {
	if ApplicationName == "" {
		t.Error("ApplicationName should not be empty")
	}

	if Version == "" {
		t.Error("Version should not be empty")
	}

	if ApplicationName != "rtlsdr2mqtt" {
		t.Errorf("Expected ApplicationName 'rtlsdr2mqtt', got '%s'", ApplicationName)
	}
}
