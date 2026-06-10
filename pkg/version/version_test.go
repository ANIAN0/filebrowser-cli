package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	s := String()
	if s == "" {
		t.Error("String() should not be empty")
	}
	if !strings.Contains(s, Version) {
		t.Errorf("String() = %q, want to contain Version = %q", s, Version)
	}
}

func TestStringWithBuildTime(t *testing.T) {
	// 保存原始值
	origVersion := Version
	origCommit := Commit
	origBuildTime := BuildTime
	defer func() {
		Version = origVersion
		Commit = origCommit
		BuildTime = origBuildTime
	}()

	// 测试有 BuildTime 的情况
	Version = "v1.0.0"
	Commit = "abc123"
	BuildTime = "2024-01-01T00:00:00Z"

	s := String()
	if !strings.Contains(s, "v1.0.0") {
		t.Errorf("String() = %q, want to contain v1.0.0", s)
	}
	if !strings.Contains(s, "abc123") {
		t.Errorf("String() = %q, want to contain abc123", s)
	}
	if !strings.Contains(s, "built") {
		t.Errorf("String() = %q, want to contain 'built'", s)
	}
}

func TestStringWithoutBuildTime(t *testing.T) {
	// 保存原始值
	origVersion := Version
	origCommit := Commit
	origBuildTime := BuildTime
	defer func() {
		Version = origVersion
		Commit = origCommit
		BuildTime = origBuildTime
	}()

	// 测试没有 BuildTime 的情况
	Version = "v1.0.0"
	Commit = "abc123"
	BuildTime = ""

	s := String()
	if !strings.Contains(s, "v1.0.0") {
		t.Errorf("String() = %q, want to contain v1.0.0", s)
	}
	if !strings.Contains(s, "abc123") {
		t.Errorf("String() = %q, want to contain abc123", s)
	}
	if strings.Contains(s, "built") {
		t.Errorf("String() = %q, should not contain 'built' when BuildTime is empty", s)
	}
}

func TestDefaults(t *testing.T) {
	if Version == "" {
		t.Error("Version default should not be empty")
	}
	if Commit == "" {
		t.Error("Commit default should not be empty")
	}
}

func TestGet(t *testing.T) {
	// 保存原始值
	origVersion := Version
	origCommit := Commit
	origBuildTime := BuildTime
	defer func() {
		Version = origVersion
		Commit = origCommit
		BuildTime = origBuildTime
	}()

	// 设置测试值
	Version = "v2.0.0"
	Commit = "def456"
	BuildTime = "2024-06-01T12:00:00Z"

	info := Get()
	if info.Version != "v2.0.0" {
		t.Errorf("Get().Version = %q, want v2.0.0", info.Version)
	}
	if info.Commit != "def456" {
		t.Errorf("Get().Commit = %q, want def456", info.Commit)
	}
	if info.BuildTime != "2024-06-01T12:00:00Z" {
		t.Errorf("Get().BuildTime = %q, want 2024-06-01T12:00:00Z", info.BuildTime)
	}
}