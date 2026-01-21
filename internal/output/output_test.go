package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/AndreyAkinshin/structyl/internal/testparser"
)

// newTestWriter creates a Writer with captured output for testing.
func newTestWriter() (*Writer, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	w := &Writer{
		out:   stdout,
		err:   stderr,
		color: false, // Disable color for predictable test output
		quiet: false,
	}
	return w, stdout, stderr
}

func TestNew(t *testing.T) {
	t.Parallel()
	w := New()
	if w == nil {
		t.Fatal("New() returned nil")
	}
	if w.out == nil {
		t.Error("out writer is nil")
	}
	if w.err == nil {
		t.Error("err writer is nil")
	}
}

func TestNewWithWriters(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	w := NewWithWriters(stdout, stderr, true)
	if w == nil {
		t.Fatal("NewWithWriters() returned nil")
	}
	if w.out != stdout {
		t.Error("out writer not set correctly")
	}
	if w.err != stderr {
		t.Error("err writer not set correctly")
	}
	if !w.color {
		t.Error("color should be true when passed true")
	}
}

func TestNewWithWriters_NoColor(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	w := NewWithWriters(stdout, stderr, false)
	if w.color {
		t.Error("color should be false when passed false")
	}
}

func TestNewWithWriters_WritesToCustomWriters(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	w := NewWithWriters(stdout, stderr, false)

	// Write to stdout via Println
	w.Println("test message")
	if !strings.Contains(stdout.String(), "test message") {
		t.Errorf("Println() did not write to custom stdout, got %q", stdout.String())
	}

	// Write to stderr via ErrorPrefix
	w.ErrorPrefix("test error")
	if !strings.Contains(stderr.String(), "test error") {
		t.Errorf("ErrorPrefix() did not write to custom stderr, got %q", stderr.String())
	}
}

func TestWriter_SetQuiet(t *testing.T) {
	t.Parallel()
	w, _, _ := newTestWriter()

	w.SetQuiet(true)
	if !w.quiet {
		t.Error("SetQuiet(true) did not set quiet")
	}

	w.SetQuiet(false)
	if w.quiet {
		t.Error("SetQuiet(false) did not unset quiet")
	}
}

func TestWriter_SetVerbose(t *testing.T) {
	t.Parallel()
	w, _, _ := newTestWriter()

	w.SetVerbose(true)
	if !w.verbose {
		t.Error("SetVerbose(true) did not set verbose")
	}
	if !w.IsVerbose() {
		t.Error("IsVerbose() = false after SetVerbose(true)")
	}

	w.SetVerbose(false)
	if w.verbose {
		t.Error("SetVerbose(false) did not unset verbose")
	}
	if w.IsVerbose() {
		t.Error("IsVerbose() = true after SetVerbose(false)")
	}
}

func TestWriter_Debug(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		verbose bool
		color   bool
		expect  string
	}{
		{"verbose without color", true, false, "[debug] test message\n"},
		{"verbose with color", true, true, "\033[2m[debug]\033[0m test message\n"},
		{"not verbose", false, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.verbose = tt.verbose
			w.color = tt.color

			w.Debug("test %s", "message")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("Debug() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_Print(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	w.Print("hello %s", "world")

	if got := stdout.String(); got != "hello world" {
		t.Errorf("Print() = %q, want %q", got, "hello world")
	}
}

func TestWriter_Println(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	w.Println("hello %s", "world")

	if got := stdout.String(); got != "hello world\n" {
		t.Errorf("Println() = %q, want %q", got, "hello world\n")
	}
}

func TestWriter_Error(t *testing.T) {
	t.Parallel()
	w, _, stderr := newTestWriter()

	w.Error("error %d", 42)

	if got := stderr.String(); got != "error 42" {
		t.Errorf("Error() = %q, want %q", got, "error 42")
	}
}

func TestWriter_Errorln(t *testing.T) {
	t.Parallel()
	w, _, stderr := newTestWriter()

	w.Errorln("error %d", 42)

	if got := stderr.String(); got != "error 42\n" {
		t.Errorf("Errorln() = %q, want %q", got, "error 42\n")
	}
}

func TestWriter_Info(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		quiet  bool
		expect string
	}{
		{"normal mode", false, "info message\n"},
		{"quiet mode", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.quiet = tt.quiet

			w.Info("info %s", "message")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("Info() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "done\n"},
		{"with color", true, "\033[32mdone\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.Success("done")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("Success() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_Warning(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "warning: caution\n"},
		{"with color", true, "\033[33mwarning: caution\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _, stderr := newTestWriter()
			w.color = tt.color

			w.Warning("caution")

			if got := stderr.String(); got != tt.expect {
				t.Errorf("Warning() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_TargetStart(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		quiet  bool
		color  bool
		expect string
	}{
		{"normal without color", false, false, "\n─── [rs] build ───\n"},
		{"normal with color", false, true, "\n\033[1m\033[36m─── [rs] build ───\033[0m\n"},
		{"quiet mode", true, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.quiet = tt.quiet
			w.color = tt.color

			w.TargetStart("rs", "build")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("TargetStart() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_TargetSuccess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		quiet  bool
		color  bool
		expect string
	}{
		{"normal without color", false, false, "[rs] build done\n"},
		{"normal with color", false, true, "\033[32m[rs]\033[0m build \033[32m✓\033[0m\n"},
		{"quiet mode", true, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.quiet = tt.quiet
			w.color = tt.color

			w.TargetSuccess("rs", "build")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("TargetSuccess() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_TargetFailed(t *testing.T) {
	t.Parallel()
	testErr := errors.New("compilation error")

	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "[rs] build: compilation error\n"},
		{"with color", true, "\033[31m[rs] build:\033[0m compilation error\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _, stderr := newTestWriter()
			w.color = tt.color

			w.TargetFailed("rs", "build", testErr)

			if got := stderr.String(); got != tt.expect {
				t.Errorf("TargetFailed() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_Section(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		quiet  bool
		color  bool
		expect string
	}{
		{"normal without color", false, false, "\n=== Build ===\n"},
		{"normal with color", false, true, "\n\033[1m=== Build ===\033[0m\n"},
		{"quiet mode", true, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.quiet = tt.quiet
			w.color = tt.color

			w.Section("Build")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("Section() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_List(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	w.List([]string{"item1", "item2", "item3"})

	expected := "  - item1\n  - item2\n  - item3\n"
	if got := stdout.String(); got != expected {
		t.Errorf("List() = %q, want %q", got, expected)
	}
}

func TestWriter_List_Empty(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	w.List([]string{})

	if got := stdout.String(); got != "" {
		t.Errorf("List() with empty slice = %q, want empty", got)
	}
}

func TestWriter_Table(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	headers := []string{"Name", "Type", "Status"}
	rows := [][]string{
		{"rs", "language", "ok"},
		{"py", "language", "ok"},
	}

	w.Table(headers, rows)

	output := stdout.String()

	// Verify headers present
	if !strings.Contains(output, "Name") {
		t.Error("Table() missing header 'Name'")
	}
	if !strings.Contains(output, "Type") {
		t.Error("Table() missing header 'Type'")
	}
	if !strings.Contains(output, "Status") {
		t.Error("Table() missing header 'Status'")
	}

	// Verify rows present
	if !strings.Contains(output, "rs") {
		t.Error("Table() missing row 'rs'")
	}
	if !strings.Contains(output, "py") {
		t.Error("Table() missing row 'py'")
	}

	// Verify separator line exists
	if !strings.Contains(output, "---") {
		t.Error("Table() missing separator line")
	}
}

func TestWriter_Table_VaryingWidths(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	headers := []string{"A", "LongHeader"}
	rows := [][]string{
		{"short", "x"},
		{"verylongvalue", "y"},
	}

	w.Table(headers, rows)

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) < 3 {
		t.Fatalf("Table() expected at least 3 lines, got %d", len(lines))
	}

	// Column width should accommodate longest value
	// "verylongvalue" is 13 chars, "LongHeader" is 10 chars
	headerLine := lines[0]
	if !strings.Contains(headerLine, "A") {
		t.Error("Table() header line missing 'A'")
	}
}

func TestWriter_Table_Empty(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	headers := []string{"Name", "Value"}
	rows := [][]string{}

	w.Table(headers, rows)

	output := stdout.String()

	// Should still print headers and separator
	if !strings.Contains(output, "Name") {
		t.Error("Table() with empty rows should still print headers")
	}
}

func TestWriter_Table_RowShorterThanHeaders(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	headers := []string{"A", "B", "C"}
	rows := [][]string{
		{"1", "2"}, // Missing third column
	}

	w.Table(headers, rows)

	// Should not panic and should handle gracefully
	output := stdout.String()
	if !strings.Contains(output, "1") {
		t.Error("Table() should handle short rows gracefully")
	}
}

// Tests for Help methods

func TestWriter_HelpTitle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "structyl v1.0\n"},
		{"with color", true, "\033[1m\033[36mstructyl v1.0\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.HelpTitle("structyl v1.0")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("HelpTitle() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_HelpSection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "\nCommands:\n"},
		{"with color", true, "\n\033[1m\033[33mCommands:\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.HelpSection("Commands:")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("HelpSection() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_HelpCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		cmd    string
		desc   string
		width  int
		expect string
	}{
		{"without color", false, "build", "Build targets", 10, "  build       Build targets\n"},
		{"with color", true, "build", "Build targets", 10, "  \033[1m\033[36mbuild\033[0m       \033[2mBuild targets\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.HelpCommand(tt.cmd, tt.desc, tt.width)

			if got := stdout.String(); got != tt.expect {
				t.Errorf("HelpCommand() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_HelpCommand_WithPlaceholder(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()
	w.color = true

	w.HelpCommand("run <task>", "Run a task", 12)

	output := stdout.String()
	// Should contain the placeholder highlighting
	if !strings.Contains(output, "\033[32m<task>\033[0m") {
		t.Errorf("HelpCommand() should highlight placeholder, got %q", output)
	}
}

func TestWriter_HelpSubCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "    --verbose  Enable verbose output\n"},
		{"with color", true, "    \033[33m--verbose\033[0m  \033[2mEnable verbose output\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.HelpSubCommand("--verbose", "Enable verbose output", 9)

			if got := stdout.String(); got != tt.expect {
				t.Errorf("HelpSubCommand() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_HelpFlag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "  --docker    Run in Docker\n"},
		{"with color", true, "  \033[33m--docker\033[0m    \033[2mRun in Docker\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.HelpFlag("--docker", "Run in Docker", 10)

			if got := stdout.String(); got != tt.expect {
				t.Errorf("HelpFlag() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_HelpFlag_WithPlaceholder(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()
	w.color = true

	w.HelpFlag("--target <name>", "Target name", 16)

	output := stdout.String()
	// Should contain the placeholder highlighting
	if !strings.Contains(output, "\033[32m<name>\033[0m") {
		t.Errorf("HelpFlag() should highlight placeholder, got %q", output)
	}
}

func TestWriter_HelpExample(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		cmd    string
		desc   string
		expect string
	}{
		{"without color no desc", false, "structyl build", "", "  structyl build\n"},
		{"without color with desc", false, "structyl build", "Build all", "  structyl build\n      Build all\n"},
		{"with color no desc", true, "structyl build", "", "  \033[36mstructyl build\033[0m\n"},
		{"with color with desc", true, "structyl build", "Build all", "  \033[36mstructyl build\033[0m\n      \033[2mBuild all\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.HelpExample(tt.cmd, tt.desc)

			if got := stdout.String(); got != tt.expect {
				t.Errorf("HelpExample() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_HelpUsage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		usage  string
		expect string
	}{
		{"without color", false, "structyl <command>", "  structyl <command>\n"},
		{"with color", true, "structyl <command>", "  structyl \033[0m\033[32m<command>\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.HelpUsage(tt.usage)

			if got := stdout.String(); got != tt.expect {
				t.Errorf("HelpUsage() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_HelpEnvVar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "  STRUCTYL_ROOT    Project root path\n"},
		{"with color", true, "  \033[33mSTRUCTYL_ROOT  \033[0m  \033[2mProject root path\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.HelpEnvVar("STRUCTYL_ROOT", "Project root path", 15)

			if got := stdout.String(); got != tt.expect {
				t.Errorf("HelpEnvVar() = %q, want %q", got, tt.expect)
			}
		})
	}
}

// Tests for Summary methods

func TestWriter_SummaryHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "\n=== Build Summary ===\n\n"},
		{"with color", true, "\n\033[1m\033[36m=== Build Summary ===\033[0m\n\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.SummaryHeader("Build Summary")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("SummaryHeader() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_SummaryItem(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "  Duration: 1.5s\n"},
		{"with color", true, "  \033[2mDuration:\033[0m 1.5s\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.SummaryItem("Duration", "1.5s")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("SummaryItem() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_SummaryPassed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "  Passed: 5\n"},
		{"with color", true, "  \033[2mPassed:\033[0m \033[32m5\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.SummaryPassed("Passed", "5")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("SummaryPassed() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_SummaryFailed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "  Failed: 2\n"},
		{"with color", true, "  \033[2mFailed:\033[0m \033[31m2\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.SummaryFailed("Failed", "2")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("SummaryFailed() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_SummaryAction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		color   bool
		success bool
		errMsg  string
		expect  string
	}{
		{"success without color", false, true, "", "    + build        1.5s\n"},
		{"success with color", true, true, "", "    \033[32m✓\033[0m build        \033[2m1.5s\033[0m\n"},
		{"failure without color", false, false, "", "    x build        1.5s\n"},
		{"failure with color", true, false, "", "    \033[31m✗\033[0m build        \033[2m1.5s\033[0m\n"},
		{"failure with error without color", false, false, "exit 1", "    x build        1.5s  (exit 1)\n"},
		{"failure with error with color", true, false, "exit 1", "    \033[31m✗\033[0m build        \033[2m1.5s\033[0m  \033[2m(exit 1)\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.SummaryAction("build", tt.success, "1.5s", tt.errMsg)

			if got := stdout.String(); got != tt.expect {
				t.Errorf("SummaryAction() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_SummarySectionLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "  Tasks:\n"},
		{"with color", true, "  \033[2mTasks:\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.SummarySectionLabel("Tasks:")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("SummarySectionLabel() = %q, want %q", got, tt.expect)
			}
		})
	}
}

// Tests for Step/Action methods

func TestWriter_Step(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "1. Build the project\n"},
		{"with color", true, "\033[36m1.\033[0m Build the project\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.Step(1, "Build the %s", "project")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("Step() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_StepDetail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "   - Running tests\n"},
		{"with color", true, "   \033[2m- Running tests\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.StepDetail("Running %s", "tests")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("StepDetail() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_Action(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "Building rs target\n"},
		{"with color", true, "\033[36mBuilding rs target\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.Action("Building %s target", "rs")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("Action() = %q, want %q", got, tt.expect)
			}
		})
	}
}

// Tests for Status methods

func TestWriter_ErrorPrefix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "structyl: target not found\n"},
		{"with color", true, "\033[31mstructyl:\033[0m target not found\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _, stderr := newTestWriter()
			w.color = tt.color

			w.ErrorPrefix("target not %s", "found")

			if got := stderr.String(); got != tt.expect {
				t.Errorf("ErrorPrefix() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_WarningSimple(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "warning: deprecated feature\n"},
		{"with color", true, "\033[33mwarning:\033[0m deprecated feature\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _, stderr := newTestWriter()
			w.color = tt.color

			w.WarningSimple("deprecated %s", "feature")

			if got := stderr.String(); got != tt.expect {
				t.Errorf("WarningSimple() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_FinalSuccess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "\nAll 3 tasks passed.\n"},
		{"with color", true, "\n\033[32mAll 3 tasks passed.\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.FinalSuccess("All %d tasks passed.", 3)

			if got := stdout.String(); got != tt.expect {
				t.Errorf("FinalSuccess() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_FinalFailure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "\n2 of 5 tasks failed.\n"},
		{"with color", true, "\n\033[31m2 of 5 tasks failed.\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.FinalFailure("%d of %d tasks failed.", 2, 5)

			if got := stdout.String(); got != tt.expect {
				t.Errorf("FinalFailure() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_ValidationSuccess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "Config is valid\n"},
		{"with color", true, "\033[32m✓\033[0m Config is valid\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.ValidationSuccess("Config is %s", "valid")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("ValidationSuccess() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_Hint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "Run 'structyl help' for more info\n"},
		{"with color", true, "\033[2mRun 'structyl help' for more info\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.Hint("Run 'structyl help' for more %s", "info")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("Hint() = %q, want %q", got, tt.expect)
			}
		})
	}
}

// Tests for DryRun/Phase/Target methods

func TestWriter_DryRunStart(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "\n=== DRY RUN ===\n\n"},
		{"with color", true, "\n\033[1m\033[33m=== DRY RUN ===\033[0m\n\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.DryRunStart()

			if got := stdout.String(); got != tt.expect {
				t.Errorf("DryRunStart() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_DryRunEnd(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "\n=== END DRY RUN ===\n"},
		{"with color", true, "\n\033[1m\033[33m=== END DRY RUN ===\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.DryRunEnd()

			if got := stdout.String(); got != tt.expect {
				t.Errorf("DryRunEnd() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_PhaseHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "\n=== Build Phase ===\n"},
		{"with color", true, "\n\033[1m\033[34m=== Build Phase ===\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.PhaseHeader("Build Phase")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("PhaseHeader() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_TargetInfo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "rs (language): Rust Library\n"},
		{"with color", true, "\033[36m\033[1mrs\033[0m (language): Rust Library\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.TargetInfo("rs", "language", "Rust Library")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("TargetInfo() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_TargetDetail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "  Directory: ./src/rs\n"},
		{"with color", true, "  \033[2mDirectory:\033[0m ./src/rs\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, stdout, _ := newTestWriter()
			w.color = tt.color

			w.TargetDetail("Directory", "./src/rs")

			if got := stdout.String(); got != tt.expect {
				t.Errorf("TargetDetail() = %q, want %q", got, tt.expect)
			}
		})
	}
}

// Tests for Utility functions

func TestFormatTestCounts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		counts *testparser.TestCounts
		expect string
	}{
		{"nil counts", nil, ""},
		{"not parsed", &testparser.TestCounts{Parsed: false}, ""},
		{"passed only", &testparser.TestCounts{Parsed: true, Passed: 10}, "10 passed"},
		{"passed and failed", &testparser.TestCounts{Parsed: true, Passed: 8, Failed: 2}, "8 passed, 2 failed"},
		{"all counts", &testparser.TestCounts{Parsed: true, Passed: 5, Failed: 2, Skipped: 3}, "5 passed, 2 failed, 3 skipped"},
		{"passed and skipped", &testparser.TestCounts{Parsed: true, Passed: 7, Skipped: 1}, "7 passed, 1 skipped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatTestCounts(tt.counts)
			if got != tt.expect {
				t.Errorf("FormatTestCounts() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		duration time.Duration
		expect   string
	}{
		{"milliseconds", 500 * time.Millisecond, "500ms"},
		{"under second", 999 * time.Millisecond, "999ms"},
		{"one second", time.Second, "1.0s"},
		{"seconds", 5500 * time.Millisecond, "5.5s"},
		{"under minute", 59 * time.Second, "59.0s"},
		{"one minute", time.Minute, "1m0s"},
		{"minutes and seconds", 2*time.Minute + 30*time.Second, "2m30s"},
		{"exact minutes", 5 * time.Minute, "5m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatDuration(tt.duration)
			if got != tt.expect {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.duration, got, tt.expect)
			}
		})
	}
}

func TestWriter_ColorPlaceholders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"no placeholders", "build all", "build all"},
		{"single placeholder", "<target>", "\033[0m\033[32m<target>\033[0m"},
		{"placeholder in text", "run <task> now", "run \033[0m\033[32m<task>\033[0m now"},
		{"multiple placeholders", "<cmd> <arg>", "\033[0m\033[32m<cmd>\033[0m \033[0m\033[32m<arg>\033[0m"},
		{"unclosed bracket", "test < value", "test < value"},
		{"utf8 no placeholder", "构建 проект", "构建 проект"},
		{"utf8 with placeholder", "构建 <target> 项目", "构建 \033[0m\033[32m<target>\033[0m 项目"},
		{"emoji no placeholder", "🚀 build 🎉", "🚀 build 🎉"},
		{"emoji with placeholder", "🚀 <task> 🎉", "🚀 \033[0m\033[32m<task>\033[0m 🎉"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _, _ := newTestWriter()
			w.color = true

			got := w.colorPlaceholders(tt.input)
			if got != tt.expect {
				t.Errorf("colorPlaceholders(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

// Tests for PrintTaskSummary

func TestWriter_PrintTaskSummary_AllPassed(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	summary := &TaskRunSummary{
		Tasks: []TaskResult{
			{Name: "build", Success: true, Duration: time.Second},
			{Name: "test", Success: true, Duration: 2 * time.Second},
		},
		TotalDuration: 3 * time.Second,
		Passed:        2,
		Failed:        0,
		TestCounts:    nil,
	}

	w.PrintTaskSummary("CI", summary)

	output := stdout.String()

	// Should contain summary header
	if !strings.Contains(output, "CI Summary") {
		t.Error("PrintTaskSummary() missing summary header")
	}

	// Should contain task names
	if !strings.Contains(output, "build") {
		t.Error("PrintTaskSummary() missing task 'build'")
	}
	if !strings.Contains(output, "test") {
		t.Error("PrintTaskSummary() missing task 'test'")
	}

	// Should contain success indicators
	if !strings.Contains(output, "+") {
		t.Error("PrintTaskSummary() missing success indicators")
	}

	// Should contain passed count
	if !strings.Contains(output, "2") {
		t.Error("PrintTaskSummary() missing passed count")
	}

	// Should contain success message
	if !strings.Contains(output, "completed successfully") {
		t.Error("PrintTaskSummary() missing success message")
	}
}

func TestWriter_PrintTaskSummary_WithFailures(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	summary := &TaskRunSummary{
		Tasks: []TaskResult{
			{Name: "build", Success: true, Duration: time.Second},
			{Name: "test", Success: false, Duration: 2 * time.Second, Error: errors.New("exit 1")},
		},
		TotalDuration: 3 * time.Second,
		Passed:        1,
		Failed:        1,
		TestCounts:    nil,
	}

	w.PrintTaskSummary("CI", summary)

	output := stdout.String()

	// Should contain failure indicator
	if !strings.Contains(output, "x") {
		t.Error("PrintTaskSummary() missing failure indicator")
	}

	// Should contain failed task name in failure message
	if !strings.Contains(output, "test") {
		t.Error("PrintTaskSummary() missing failed task name")
	}

	// Should contain failure message
	if !strings.Contains(output, "failed") {
		t.Error("PrintTaskSummary() missing failure message")
	}
}

func TestWriter_PrintTaskSummary_WithTestCounts(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	taskCounts := &testparser.TestCounts{
		Parsed:  true,
		Passed:  10,
		Failed:  2,
		Skipped: 1,
	}

	summary := &TaskRunSummary{
		Tasks: []TaskResult{
			{
				Name:       "test",
				Success:    false,
				Duration:   time.Second,
				TestCounts: taskCounts,
			},
		},
		TotalDuration: time.Second,
		Passed:        0,
		Failed:        1,
		TestCounts:    taskCounts,
	}

	w.PrintTaskSummary("Test", summary)

	output := stdout.String()

	// Should contain test counts
	if !strings.Contains(output, "10 passed") {
		t.Error("PrintTaskSummary() missing passed test count")
	}
	if !strings.Contains(output, "2 failed") {
		t.Error("PrintTaskSummary() missing failed test count")
	}
	if !strings.Contains(output, "1 skipped") {
		t.Error("PrintTaskSummary() missing skipped test count")
	}
}

func TestWriter_PrintTaskSummary_EmptyTasks(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	summary := &TaskRunSummary{
		Tasks:         []TaskResult{},
		TotalDuration: 0,
		Passed:        0,
		Failed:        0,
		TestCounts:    nil,
	}

	w.PrintTaskSummary("Empty", summary)

	output := stdout.String()

	// Should still print summary header
	if !strings.Contains(output, "Empty Summary") {
		t.Error("PrintTaskSummary() missing summary header for empty tasks")
	}

	// Should show 0 tasks
	if !strings.Contains(output, "0") {
		t.Error("PrintTaskSummary() should show 0 for empty tasks")
	}
}

func TestWriter_UpdateNotification(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		color   bool
		version string
		expect  string
	}{
		{"without color", false, "1.2.3", "structyl 1.2.3 available. Run 'structyl upgrade' to update.\n"},
		{"with color", true, "2.0.0", "\033[2mstructyl 2.0.0 available. Run 'structyl upgrade' to update.\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _, stderr := newTestWriter()
			w.color = tt.color

			w.UpdateNotification(tt.version)

			if got := stderr.String(); got != tt.expect {
				t.Errorf("UpdateNotification() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWriter_PrintTaskSummary_WithFailedTests(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	taskCounts := &testparser.TestCounts{
		Parsed:  true,
		Passed:  8,
		Failed:  2,
		Skipped: 0,
		FailedTests: []testparser.FailedTest{
			{Name: "TestFoo", Reason: "assertion failed"},
			{Name: "TestBar", Reason: ""},
		},
	}

	summary := &TaskRunSummary{
		Tasks: []TaskResult{
			{
				Name:       "test",
				Success:    false,
				Duration:   time.Second,
				TestCounts: taskCounts,
				Error:      errors.New("exit 1"),
			},
		},
		TotalDuration: time.Second,
		Passed:        0,
		Failed:        1,
		TestCounts:    taskCounts,
	}

	w.PrintTaskSummary("Test", summary)

	output := stdout.String()

	// Should contain failed tests section
	if !strings.Contains(output, "Failed Tests") {
		t.Error("PrintTaskSummary() missing 'Failed Tests' section")
	}

	// Should contain failed test names
	if !strings.Contains(output, "TestFoo") {
		t.Error("PrintTaskSummary() missing failed test name 'TestFoo'")
	}
	if !strings.Contains(output, "TestBar") {
		t.Error("PrintTaskSummary() missing failed test name 'TestBar'")
	}

	// Should contain failure reason when available
	if !strings.Contains(output, "assertion failed") {
		t.Error("PrintTaskSummary() missing failure reason")
	}
}

func TestWriter_PrintTaskSummary_WithFailedTests_Color(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()
	w.color = true

	taskCounts := &testparser.TestCounts{
		Parsed:  true,
		Passed:  5,
		Failed:  1,
		Skipped: 2,
		FailedTests: []testparser.FailedTest{
			{Name: "TestFailure", Reason: "timeout"},
		},
	}

	summary := &TaskRunSummary{
		Tasks: []TaskResult{
			{
				Name:       "test",
				Success:    false,
				Duration:   time.Second,
				TestCounts: taskCounts,
			},
		},
		TotalDuration: time.Second,
		Passed:        0,
		Failed:        1,
		TestCounts:    taskCounts,
	}

	w.PrintTaskSummary("Test", summary)

	output := stdout.String()

	// Should contain colored indicators for passed/failed/skipped
	if !strings.Contains(output, "\033[32m") { // green for passed
		t.Error("PrintTaskSummary() missing green color for passed tests")
	}
	if !strings.Contains(output, "\033[31m") { // red for failed
		t.Error("PrintTaskSummary() missing red color for failed tests")
	}
	if !strings.Contains(output, "\033[33m") { // yellow for skipped
		t.Error("PrintTaskSummary() missing yellow color for skipped tests")
	}
}

func TestWriter_printFailedTestDetails_Empty(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	// Access via PrintTaskSummary with no failed tests
	taskCounts := &testparser.TestCounts{
		Parsed:      true,
		Passed:      10,
		Failed:      0,
		FailedTests: nil,
	}

	summary := &TaskRunSummary{
		Tasks: []TaskResult{
			{
				Name:       "test",
				Success:    true,
				Duration:   time.Second,
				TestCounts: taskCounts,
			},
		},
		TotalDuration: time.Second,
		Passed:        1,
		Failed:        0,
		TestCounts:    taskCounts,
	}

	w.PrintTaskSummary("Test", summary)

	output := stdout.String()

	// Should NOT contain "Failed Tests" section
	if strings.Contains(output, "Failed Tests") {
		t.Error("PrintTaskSummary() should not show 'Failed Tests' section when no tests failed")
	}
}

func TestWriter_printTaskResultLine_WithError(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	summary := &TaskRunSummary{
		Tasks: []TaskResult{
			{
				Name:     "build",
				Success:  false,
				Duration: 2 * time.Second,
				Error:    errors.New("compilation error"),
			},
		},
		TotalDuration: 2 * time.Second,
		Passed:        0,
		Failed:        1,
	}

	w.PrintTaskSummary("Build", summary)

	output := stdout.String()

	// Should contain the error message in parentheses
	if !strings.Contains(output, "(compilation error)") {
		t.Error("PrintTaskSummary() missing error message in task line")
	}
}

func TestWriter_printTaskResultLine_WithTestCounts_NoError(t *testing.T) {
	t.Parallel()
	w, stdout, _ := newTestWriter()

	taskCounts := &testparser.TestCounts{
		Parsed: true,
		Passed: 15,
		Failed: 0,
	}

	summary := &TaskRunSummary{
		Tasks: []TaskResult{
			{
				Name:       "test",
				Success:    true,
				Duration:   time.Second,
				TestCounts: taskCounts,
			},
		},
		TotalDuration: time.Second,
		Passed:        1,
		Failed:        0,
		TestCounts:    taskCounts,
	}

	w.PrintTaskSummary("Test", summary)

	output := stdout.String()

	// Should show test counts, not error
	if !strings.Contains(output, "15 passed") {
		t.Error("PrintTaskSummary() missing test counts")
	}
}

func TestWriter_HelpCommand_NegativePadding(t *testing.T) {
	t.Parallel()
	// Test when command name is longer than width (negative padding)
	w, stdout, _ := newTestWriter()
	w.color = true

	w.HelpCommand("very-long-command-name", "Description", 5)

	output := stdout.String()
	if !strings.Contains(output, "very-long-command-name") {
		t.Error("HelpCommand() should handle long command names")
	}
}

func TestWriter_HelpFlag_NegativePadding(t *testing.T) {
	t.Parallel()
	// Test when flag name is longer than width (negative padding)
	w, stdout, _ := newTestWriter()
	w.color = true

	w.HelpFlag("--very-long-flag-name", "Description", 5)

	output := stdout.String()
	if !strings.Contains(output, "very-long-flag-name") {
		t.Error("HelpFlag() should handle long flag names")
	}
}
