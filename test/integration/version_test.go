package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/version"
)

// Integration tests for version functionality with real fixtures.
// Unit tests for version validation, parsing, bump, and compare are in
// internal/version/version_test.go. These tests focus on file I/O with
// real project fixtures.

func TestVersionRead(t *testing.T) {
	fixtureDir := filepath.Join(fixturesDir(), "multi-language")
	versionPath := filepath.Join(fixtureDir, "VERSION")

	v, err := version.Read(versionPath)
	if err != nil {
		t.Fatalf("failed to read version: %v", err)
	}

	if v != "1.2.3" {
		t.Errorf("expected version %q, got %q", "1.2.3", v)
	}
}

func TestVersionReadMissing(t *testing.T) {
	fixtureDir := filepath.Join(fixturesDir(), "minimal")
	versionPath := filepath.Join(fixtureDir, "VERSION")

	_, err := version.Read(versionPath)
	if err == nil {
		t.Error("expected error when reading missing VERSION file")
	}
}

func TestVersionWriteAndRead(t *testing.T) {
	tmpDir := t.TempDir()
	versionPath := filepath.Join(tmpDir, "VERSION")

	// Write version
	err := version.Write(versionPath, "2.0.0")
	if err != nil {
		t.Fatalf("failed to write version: %v", err)
	}

	// Read version
	v, err := version.Read(versionPath)
	if err != nil {
		t.Fatalf("failed to read version: %v", err)
	}

	if v != "2.0.0" {
		t.Errorf("expected version %q, got %q", "2.0.0", v)
	}
}

func TestVersionWriteInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	versionPath := filepath.Join(tmpDir, "VERSION")

	err := version.Write(versionPath, "invalid")
	if err == nil {
		t.Error("expected error when writing invalid version")
	}

	// Ensure file was not created
	if _, err := os.Stat(versionPath); !os.IsNotExist(err) {
		t.Error("expected version file to not be created for invalid version")
	}
}
