package version

import (
	"testing"
)

func TestGetInfo(t *testing.T) {
	info := GetInfo()

	if info.Version == "" {
		t.Error("Version should not be empty")
	}

	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}

	if info.GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}

	if info.BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}
}

func TestVersionDefaults(t *testing.T) {
	// Test default values when not set via ldflags
	if Version == "" {
		t.Error("Version should have a default value")
	}

	if GitCommit == "" {
		t.Error("GitCommit should have a default value")
	}

	if BuildTime == "" {
		t.Error("BuildTime should have a default value")
	}
}
