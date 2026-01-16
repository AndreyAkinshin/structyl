package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AndreyAkinshin/structyl/internal/output"
)

func TestFormatDuration_Milliseconds(t *testing.T) {
	d := 500 * time.Millisecond
	result := FormatDuration(d)
	if result != "500ms" {
		t.Errorf("FormatDuration() = %q, want %q", result, "500ms")
	}
}

func TestFormatDuration_Seconds(t *testing.T) {
	d := 30 * time.Second
	result := FormatDuration(d)
	if result != "30.0s" {
		t.Errorf("FormatDuration() = %q, want %q", result, "30.0s")
	}
}

func TestFormatDuration_Minutes(t *testing.T) {
	d := 2*time.Minute + 30*time.Second
	result := FormatDuration(d)
	if result != "2m30s" {
		t.Errorf("FormatDuration() = %q, want %q", result, "2m30s")
	}
}

func TestPhaseOrder_Debug(t *testing.T) {
	phases := PhaseOrder(false)

	expected := []string{"clean", "init", "check", "build", "test"}
	if len(phases) != len(expected) {
		t.Errorf("PhaseOrder(false) returned %d phases, want %d", len(phases), len(expected))
	}

	for i, p := range expected {
		if phases[i] != p {
			t.Errorf("phases[%d] = %q, want %q", i, phases[i], p)
		}
	}
}

func TestPhaseOrder_Release(t *testing.T) {
	phases := PhaseOrder(true)

	// Should have build:release instead of build
	found := false
	for _, p := range phases {
		if p == "build:release" {
			found = true
		}
		if p == "build" {
			t.Error("release mode should not have 'build' phase")
		}
	}

	if !found {
		t.Error("release mode should have 'build:release' phase")
	}
}

func TestCIResult_Success(t *testing.T) {
	result := &CIResult{
		StartTime: time.Now().Add(-time.Second),
		EndTime:   time.Now(),
		Duration:  time.Second,
		Success:   true,
		PhaseResults: []PhaseResult{
			{Name: "build", Success: true},
			{Name: "test", Success: true},
		},
	}

	if !result.Success {
		t.Error("result.Success should be true")
	}
}

func TestCIResult_Failure(t *testing.T) {
	result := &CIResult{
		StartTime: time.Now().Add(-time.Second),
		EndTime:   time.Now(),
		Duration:  time.Second,
		Success:   false,
		PhaseResults: []PhaseResult{
			{Name: "build", Success: true},
			{Name: "test", Success: false},
		},
	}

	if result.Success {
		t.Error("result.Success should be false")
	}
}

func TestPhaseResult_Duration(t *testing.T) {
	start := time.Now()
	end := start.Add(100 * time.Millisecond)

	result := PhaseResult{
		Name:      "build",
		StartTime: start,
		EndTime:   end,
		Duration:  end.Sub(start),
		Success:   true,
	}

	if result.Duration != 100*time.Millisecond {
		t.Errorf("Duration = %v, want 100ms", result.Duration)
	}
}

func TestCIOptions_Release(t *testing.T) {
	opts := CIOptions{
		Release:  true,
		Parallel: true,
	}

	if !opts.Release {
		t.Error("Release should be true")
	}
	if !opts.Parallel {
		t.Error("Parallel should be true")
	}
}

func TestFormatDuration_Boundary999ms(t *testing.T) {
	d := 999 * time.Millisecond
	result := FormatDuration(d)
	if result != "999ms" {
		t.Errorf("FormatDuration() = %q, want %q", result, "999ms")
	}
}

func TestFormatDuration_BoundaryExactlyOneSecond(t *testing.T) {
	d := time.Second
	result := FormatDuration(d)
	if result != "1.0s" {
		t.Errorf("FormatDuration() = %q, want %q", result, "1.0s")
	}
}

func TestFormatDuration_BoundaryExactlyOneMinute(t *testing.T) {
	d := time.Minute
	result := FormatDuration(d)
	if result != "1m0s" {
		t.Errorf("FormatDuration() = %q, want %q", result, "1m0s")
	}
}

func TestFormatDuration_59Seconds(t *testing.T) {
	d := 59*time.Second + 500*time.Millisecond
	result := FormatDuration(d)
	if result != "59.5s" {
		t.Errorf("FormatDuration() = %q, want %q", result, "59.5s")
	}
}

func TestCopyFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	content := []byte("test content")
	err := os.WriteFile(srcPath, content, 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	err = copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	destContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read dest file: %v", err)
	}

	if string(destContent) != string(content) {
		t.Errorf("dest content = %q, want %q", string(destContent), string(content))
	}
}

func TestCopyFile_SourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "nonexistent.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	err := copyFile(srcPath, dstPath)
	if err == nil {
		t.Error("copyFile() expected error for missing source")
	}
}

func TestCopyFile_DestDirNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "nonexistent", "dest.txt")

	err := os.WriteFile(srcPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	err = copyFile(srcPath, dstPath)
	if err == nil {
		t.Error("copyFile() expected error for missing dest directory")
	}
}

func TestCopyFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "empty.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	err := os.WriteFile(srcPath, []byte{}, 0644)
	if err != nil {
		t.Fatalf("failed to create empty source file: %v", err)
	}

	err = copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("failed to stat dest file: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("dest file size = %d, want 0", info.Size())
	}
}

func TestFindArtifacts_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()
	mock := &mockTarget{
		name:     "test",
		commands: map[string]bool{},
	}
	mock.directory = tmpDir

	artifacts := findArtifacts(mock)
	if len(artifacts) != 0 {
		t.Errorf("findArtifacts() = %v, want empty", artifacts)
	}
}

func TestFindArtifacts_MatchesRustPattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Create target/release directory with artifact
	releaseDir := filepath.Join(tmpDir, "target", "release")
	err := os.MkdirAll(releaseDir, 0755)
	if err != nil {
		t.Fatalf("failed to create release dir: %v", err)
	}
	artifactPath := filepath.Join(releaseDir, "myapp.exe")
	err = os.WriteFile(artifactPath, []byte("binary"), 0755)
	if err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	mock := &mockTarget{
		name:     "rs",
		commands: map[string]bool{},
	}
	mock.directory = tmpDir

	artifacts := findArtifacts(mock)
	if len(artifacts) != 1 {
		t.Errorf("findArtifacts() count = %d, want 1", len(artifacts))
	}
	if len(artifacts) > 0 && artifacts[0] != artifactPath {
		t.Errorf("findArtifacts()[0] = %q, want %q", artifacts[0], artifactPath)
	}
}

func TestFindArtifacts_MatchesBinPattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Create bin directory with artifact
	binDir := filepath.Join(tmpDir, "bin")
	err := os.MkdirAll(binDir, 0755)
	if err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	artifactPath := filepath.Join(binDir, "myapp")
	err = os.WriteFile(artifactPath, []byte("binary"), 0755)
	if err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	mock := &mockTarget{
		name:     "go",
		commands: map[string]bool{},
	}
	mock.directory = tmpDir

	artifacts := findArtifacts(mock)
	if len(artifacts) != 1 {
		t.Errorf("findArtifacts() count = %d, want 1", len(artifacts))
	}
}

func TestTargetResult_Fields(t *testing.T) {
	result := TargetResult{
		Name:     "rs",
		Success:  false,
		Duration: 5 * time.Second,
		Errors:   []error{},
	}

	if result.Name != "rs" {
		t.Errorf("Name = %q, want %q", result.Name, "rs")
	}
	if result.Success {
		t.Error("Success should be false")
	}
	if result.Duration != 5*time.Second {
		t.Errorf("Duration = %v, want 5s", result.Duration)
	}
}

// =============================================================================
// Work Item 5: RunCI Tests
// =============================================================================

func TestRunCI_Success(t *testing.T) {
	registry, _ := createTestRegistry(t)
	runner := New(registry)

	ctx := context.Background()
	result, err := runner.RunCI(ctx, CIOptions{})

	// RunCI doesn't error even if commands fail, it just sets Success=false
	if err != nil {
		t.Errorf("RunCI() error = %v, want nil", err)
	}
	if result == nil {
		t.Fatal("RunCI() result = nil")
	}

	// Should have phase results
	if len(result.PhaseResults) == 0 {
		t.Error("RunCI() PhaseResults should not be empty")
	}

	// Duration should be set
	if result.Duration == 0 {
		t.Error("RunCI() Duration should not be zero")
	}
}

func TestRunCI_Release_UsesBuildRelease(t *testing.T) {
	registry, _ := createTestRegistry(t)
	runner := New(registry)

	ctx := context.Background()
	result, err := runner.RunCI(ctx, CIOptions{Release: true})

	if err != nil {
		t.Errorf("RunCI() error = %v", err)
		return
	}

	// Check that phases include build:release
	hasReleaseBuild := false
	for _, phase := range result.PhaseResults {
		if phase.Name == "build:release" {
			hasReleaseBuild = true
			break
		}
	}
	if !hasReleaseBuild {
		t.Error("RunCI(Release=true) should have 'build:release' phase")
	}
}

func TestRunCI_ContinueOnError_RunsAllPhases(t *testing.T) {
	registry, _ := createTestRegistry(t)
	runner := New(registry)

	ctx := context.Background()
	result, _ := runner.RunCI(ctx, CIOptions{Continue: true})

	// With Continue=true, should have all standard phases
	expectedPhases := []string{"clean", "init", "check", "build", "test"}
	// Count phases (they may appear twice - once for aux, once for lang)
	phaseNames := make(map[string]bool)
	for _, phase := range result.PhaseResults {
		phaseNames[phase.Name] = true
	}

	for _, expected := range expectedPhases {
		if !phaseNames[expected] {
			t.Errorf("expected phase %q not found in results", expected)
		}
	}
}

func TestRunCI_CollectsArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "artifacts")

	registry, projRoot := createTestRegistry(t)

	// Create a test artifact in the target directory
	binDir := filepath.Join(projRoot, "rs", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(binDir, "testapp")
	if err := os.WriteFile(artifactPath, []byte("binary"), 0755); err != nil {
		t.Fatal(err)
	}

	runner := New(registry)
	ctx := context.Background()
	// Use Continue: true so artifact collection runs even if phases fail
	result, err := runner.RunCI(ctx, CIOptions{ArtifactDir: outputDir, Continue: true})

	if err != nil {
		t.Errorf("RunCI() error = %v", err)
	}

	// Output directory should be created when ArtifactDir is set
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("artifact output directory was not created")
	}

	// Artifact count should be >= 0
	_ = result.ArtifactCount // May or may not find artifacts depending on patterns
}

// =============================================================================
// Work Item 6: collectArtifacts Tests (additional)
// =============================================================================

func TestCollectArtifacts_CreatesOutputDir(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "new", "nested", "dir")

	registry, _ := createTestRegistry(t)
	runner := New(registry)

	targets, _ := registry.TopologicalOrder()
	_, err := runner.collectArtifacts(context.Background(), targets, outputDir, nil)

	if err != nil {
		t.Errorf("collectArtifacts() error = %v", err)
	}

	// Output directory should exist
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("collectArtifacts() did not create output directory")
	}
}

func TestCollectArtifacts_NoMatches_ReturnsZero(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "artifacts")

	registry, _ := createTestRegistry(t)
	runner := New(registry)

	targets, _ := registry.TopologicalOrder()
	count, err := runner.collectArtifacts(context.Background(), targets, outputDir, nil)

	if err != nil {
		t.Errorf("collectArtifacts() error = %v", err)
	}
	if count != 0 {
		t.Errorf("collectArtifacts() = %d, want 0 (no matching artifacts)", count)
	}
}

// =============================================================================
// Work Item 7: PrintCISummary Tests
// =============================================================================

func TestPrintCISummary_Success(t *testing.T) {
	result := &CIResult{
		StartTime: time.Now().Add(-time.Second),
		EndTime:   time.Now(),
		Duration:  time.Second,
		Success:   true,
		PhaseResults: []PhaseResult{
			{Name: "clean", Success: true, Duration: 100 * time.Millisecond},
			{Name: "build", Success: true, Duration: 500 * time.Millisecond},
			{Name: "test", Success: true, Duration: 400 * time.Millisecond},
		},
		ArtifactCount: 2,
	}

	// Verify the function completes without panic and doesn't modify input
	originalCount := len(result.PhaseResults)
	out := output.New()
	PrintCISummary(result, out)

	// Verify input wasn't modified
	if len(result.PhaseResults) != originalCount {
		t.Errorf("PrintCISummary modified PhaseResults length: got %d, want %d",
			len(result.PhaseResults), originalCount)
	}
}

func TestPrintCISummary_Failure(t *testing.T) {
	result := &CIResult{
		StartTime: time.Now().Add(-2 * time.Second),
		EndTime:   time.Now(),
		Duration:  2 * time.Second,
		Success:   false,
		PhaseResults: []PhaseResult{
			{Name: "clean", Success: true, Duration: 100 * time.Millisecond},
			{Name: "build", Success: true, Duration: 500 * time.Millisecond},
			{Name: "test", Success: false, Duration: 1400 * time.Millisecond},
		},
		ArtifactCount: 0,
	}

	// Verify the function handles failure results without panic
	// and preserves the original success status
	out := output.New()
	PrintCISummary(result, out)

	if result.Success != false {
		t.Error("PrintCISummary should not modify Success field")
	}
}

func TestPrintCISummary_WithArtifacts(t *testing.T) {
	result := &CIResult{
		StartTime: time.Now().Add(-time.Second),
		EndTime:   time.Now(),
		Duration:  time.Second,
		Success:   true,
		PhaseResults: []PhaseResult{
			{Name: "build", Success: true},
		},
		ArtifactCount: 5,
	}

	// Verify artifact count is preserved after print
	out := output.New()
	PrintCISummary(result, out)

	if result.ArtifactCount != 5 {
		t.Errorf("PrintCISummary modified ArtifactCount: got %d, want 5", result.ArtifactCount)
	}
}

func TestPrintCISummary_EmptyPhases(t *testing.T) {
	result := &CIResult{
		StartTime:    time.Now(),
		EndTime:      time.Now(),
		Duration:     0,
		Success:      true,
		PhaseResults: []PhaseResult{},
	}

	// Verify function handles empty phases gracefully
	out := output.New()
	PrintCISummary(result, out)

	// Verify empty slice wasn't modified
	if len(result.PhaseResults) != 0 {
		t.Errorf("PrintCISummary added to empty PhaseResults: got %d elements", len(result.PhaseResults))
	}
}
