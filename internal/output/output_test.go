package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"
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

func TestWriter_SetQuiet(t *testing.T) {
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

func TestWriter_Print(t *testing.T) {
	w, stdout, _ := newTestWriter()

	w.Print("hello %s", "world")

	if got := stdout.String(); got != "hello world" {
		t.Errorf("Print() = %q, want %q", got, "hello world")
	}
}

func TestWriter_Println(t *testing.T) {
	w, stdout, _ := newTestWriter()

	w.Println("hello %s", "world")

	if got := stdout.String(); got != "hello world\n" {
		t.Errorf("Println() = %q, want %q", got, "hello world\n")
	}
}

func TestWriter_Error(t *testing.T) {
	w, _, stderr := newTestWriter()

	w.Error("error %d", 42)

	if got := stderr.String(); got != "error 42" {
		t.Errorf("Error() = %q, want %q", got, "error 42")
	}
}

func TestWriter_Errorln(t *testing.T) {
	w, _, stderr := newTestWriter()

	w.Errorln("error %d", 42)

	if got := stderr.String(); got != "error 42\n" {
		t.Errorf("Errorln() = %q, want %q", got, "error 42\n")
	}
}

func TestWriter_Info(t *testing.T) {
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
	tests := []struct {
		name   string
		quiet  bool
		color  bool
		expect string
	}{
		{"normal without color", false, false, "[rs] build\n"},
		{"normal with color", false, true, "\033[1m[rs]\033[0m build\n"},
		{"quiet mode", true, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	testErr := errors.New("compilation error")

	tests := []struct {
		name   string
		color  bool
		expect string
	}{
		{"without color", false, "[rs] build failed: compilation error\n"},
		{"with color", true, "\033[31m[rs] build failed:\033[0m compilation error\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	w, stdout, _ := newTestWriter()

	w.List([]string{"item1", "item2", "item3"})

	expected := "  - item1\n  - item2\n  - item3\n"
	if got := stdout.String(); got != expected {
		t.Errorf("List() = %q, want %q", got, expected)
	}
}

func TestWriter_List_Empty(t *testing.T) {
	w, stdout, _ := newTestWriter()

	w.List([]string{})

	if got := stdout.String(); got != "" {
		t.Errorf("List() with empty slice = %q, want empty", got)
	}
}

func TestWriter_Table(t *testing.T) {
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
