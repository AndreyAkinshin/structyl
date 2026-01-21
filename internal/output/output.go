// Package output provides formatted output utilities for the CLI.
//
// # Design Note: Singleton Pattern
//
// CLI commands typically use a package-level Writer instance created via New().
// This singleton pattern is intentional for CLI applications where:
//   - Output configuration (color, verbosity) is set once at startup
//   - Thread safety is not a concern (CLI is single-threaded)
//   - Simplifies command handlers that need output access
//
// For testing, use NewWithWriters to inject custom io.Writers and capture output.
// Tests should create isolated Writer instances rather than modifying the global.
//
// Write errors are intentionally ignored throughout this package. CLI output
// failures (broken pipe, closed terminal) are non-recoverable and should not
// affect exit codes or program flow.
package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/AndreyAkinshin/structyl/internal/model"
	"github.com/AndreyAkinshin/structyl/internal/testparser"
)

// Writer handles CLI output formatting.
type Writer struct {
	out     io.Writer
	err     io.Writer
	color   bool
	quiet   bool
	verbose bool
}

// New creates a new Writer with default settings.
func New() *Writer {
	return &Writer{
		out:   os.Stdout,
		err:   os.Stderr,
		color: isTerminal(),
	}
}

// NewWithWriters creates a Writer with custom io.Writers (for testing).
func NewWithWriters(out, err io.Writer, color bool) *Writer {
	return &Writer{
		out:   out,
		err:   err,
		color: color,
	}
}

// SetQuiet enables or disables quiet mode.
func (w *Writer) SetQuiet(quiet bool) {
	w.quiet = quiet
}

// SetVerbose enables or disables verbose mode.
func (w *Writer) SetVerbose(verbose bool) {
	w.verbose = verbose
}

// IsVerbose returns true if verbose mode is enabled.
func (w *Writer) IsVerbose() bool {
	return w.verbose
}

// styled wraps text with ANSI style codes if color is enabled.
// Returns plain text when color output is disabled.
func (w *Writer) styled(style, text string) string {
	if w.color {
		return style + text + reset
	}
	return text
}

// Debug prints a debug message (only in verbose mode).
func (w *Writer) Debug(format string, args ...interface{}) {
	if !w.verbose {
		return
	}
	msg := fmt.Sprintf(format, args...)
	if w.color {
		w.Println("%s[debug]%s %s", dim, reset, msg)
	} else {
		w.Println("[debug] %s", msg)
	}
}

// Print formats and writes to the output stream.
// Write errors are intentionally ignored: CLI output failures (broken pipe,
// closed terminal) are non-recoverable and should not affect the exit code.
func (w *Writer) Print(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(w.out, format, args...)
}

// Println writes a line to stdout.
func (w *Writer) Println(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(w.out, format+"\n", args...)
}

// Error writes to stderr.
func (w *Writer) Error(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(w.err, format, args...)
}

// Errorln writes a line to stderr.
func (w *Writer) Errorln(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(w.err, format+"\n", args...)
}

// Info prints an info message (skipped in quiet mode).
func (w *Writer) Info(format string, args ...interface{}) {
	if w.quiet {
		return
	}
	w.Println(format, args...)
}

// Success prints a success message.
func (w *Writer) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	w.Println("%s", w.styled(green, msg))
}

// Warning prints a warning message.
func (w *Writer) Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf("warning: "+format, args...)
	w.Errorln("%s", w.styled(yellow, msg))
}

// TargetStart prints the start of a target command with enhanced visibility.
func (w *Writer) TargetStart(target, command string) {
	if w.quiet {
		return
	}
	// Empty line for visual separation
	w.Println("")
	label := fmt.Sprintf("─── [%s] %s ───", target, command)
	w.Println("%s", w.styled(bold+cyan, label))
}

// TargetSuccess prints target command success.
func (w *Writer) TargetSuccess(target, command string) {
	if w.quiet {
		return
	}
	if w.color {
		w.Println(green+"[%s]"+reset+" %s "+green+"✓"+reset, target, command)
	} else {
		w.Println("[%s] %s done", target, command)
	}
}

// TargetFailed prints target command failure.
func (w *Writer) TargetFailed(target, command string, err error) {
	if w.color {
		w.Errorln(red+"[%s] %s failed:"+reset+" %v", target, command, err)
	} else {
		w.Errorln("[%s] %s failed: %v", target, command, err)
	}
}

// Section prints a section header.
func (w *Writer) Section(title string) {
	if w.quiet {
		return
	}
	w.Println("")
	header := fmt.Sprintf("=== %s ===", title)
	w.Println("%s", w.styled(bold, header))
}

// List prints a list of items.
func (w *Writer) List(items []string) {
	for _, item := range items {
		w.Println("  - %s", item)
	}
}

// Table prints a simple table.
func (w *Writer) Table(headers []string, rows [][]string) {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	// Note: Extra columns in rows beyond headers are silently ignored.
	// This allows flexible row data without strict schema enforcement.
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	var headerParts []string
	for i, h := range headers {
		headerParts = append(headerParts, fmt.Sprintf("%-*s", widths[i], h))
	}
	w.Println(strings.Join(headerParts, "  "))

	// Print separator
	var sepParts []string
	for _, width := range widths {
		sepParts = append(sepParts, strings.Repeat("-", width))
	}
	w.Println(strings.Join(sepParts, "  "))

	// Print rows
	for _, row := range rows {
		var rowParts []string
		for i, cell := range row {
			if i < len(widths) {
				rowParts = append(rowParts, fmt.Sprintf("%-*s", widths[i], cell))
			}
		}
		w.Println(strings.Join(rowParts, "  "))
	}
}

// isTerminal returns true if stdout is a terminal.
func isTerminal() bool {
	// Simple check - could be enhanced with golang.org/x/term
	if fi, _ := os.Stdout.Stat(); fi != nil {
		return (fi.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

// ANSI color codes.
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
)

// Semantic color roles for help output.
const (
	colorTitle       = bold + cyan   // Main title/brand
	colorSection     = bold + yellow // Section headers
	colorCommand     = bold + cyan   // Commands and subcommands
	colorPlaceholder = green         // Placeholders like <target>, <ver>
	colorFlag        = yellow        // Flags like --docker
	colorDescription = dim           // Help text descriptions
	colorExample     = cyan          // Example commands
	colorEnvVar      = yellow        // Environment variables
)

// HelpTitle formats the main help title line.
func (w *Writer) HelpTitle(title string) {
	w.Println("%s", w.styled(colorTitle, title))
}

// HelpSection formats a section header (e.g., "Meta Commands:").
func (w *Writer) HelpSection(title string) {
	w.Println("")
	w.Println("%s", w.styled(colorSection, title))
}

// HelpCommand formats a command with its description.
func (w *Writer) HelpCommand(name, description string, width int) {
	if w.color {
		coloredName := w.colorPlaceholders(name)
		// Calculate display width (name without ANSI codes)
		padding := width - len(name)
		if padding < 0 {
			padding = 0
		}
		w.Println("  %s%s%s%s  %s%s%s", colorCommand, coloredName, reset, strings.Repeat(" ", padding), colorDescription, description, reset)
	} else {
		w.Println("  %-*s  %s", width, name, description)
	}
}

// HelpSubCommand formats a sub-command or flag with indented description.
func (w *Writer) HelpSubCommand(name, description string, width int) {
	if w.color {
		w.Println("    %s%-*s%s  %s%s%s", colorFlag, width, name, reset, colorDescription, description, reset)
	} else {
		w.Println("    %-*s  %s", width, name, description)
	}
}

// HelpFlag formats a flag with its description.
func (w *Writer) HelpFlag(name, description string, width int) {
	if w.color {
		coloredName := w.colorPlaceholders(name)
		padding := width - len(name)
		if padding < 0 {
			padding = 0
		}
		w.Println("  %s%s%s%s  %s%s%s", colorFlag, coloredName, reset, strings.Repeat(" ", padding), colorDescription, description, reset)
	} else {
		w.Println("  %-*s  %s", width, name, description)
	}
}

// HelpExample formats an example command with description.
func (w *Writer) HelpExample(command, description string) {
	if w.color {
		w.Println("  %s%s%s", colorExample, command, reset)
		if description != "" {
			w.Println("      %s%s%s", colorDescription, description, reset)
		}
	} else {
		w.Println("  %s", command)
		if description != "" {
			w.Println("      %s", description)
		}
	}
}

// HelpUsage formats usage lines.
func (w *Writer) HelpUsage(usage string) {
	if w.color {
		colored := w.colorPlaceholders(usage)
		w.Println("  %s", colored)
	} else {
		w.Println("  %s", usage)
	}
}

// HelpEnvVar formats an environment variable.
func (w *Writer) HelpEnvVar(name, description string, width int) {
	if w.color {
		w.Println("  %s%-*s%s  %s%s%s", colorEnvVar, width, name, reset, colorDescription, description, reset)
	} else {
		w.Println("  %-*s  %s", width, name, description)
	}
}

// Step prints a numbered step message with color.
func (w *Writer) Step(num int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		w.Println("%s%d.%s %s", cyan, num, reset, msg)
	} else {
		w.Println("%d. %s", num, msg)
	}
}

// StepDetail prints an indented detail line under a step.
func (w *Writer) StepDetail(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		w.Println("   %s- %s%s", dim, msg, reset)
	} else {
		w.Println("   - %s", msg)
	}
}

// Action prints an action message (what the CLI is doing).
func (w *Writer) Action(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		w.Println("%s%s%s", cyan, msg, reset)
	} else {
		w.Println("%s", msg)
	}
}

// ErrorPrefix prints an error message with structyl prefix to stderr.
func (w *Writer) ErrorPrefix(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		w.Errorln("%sstructyl:%s %s", red, reset, msg)
	} else {
		w.Errorln("structyl: %s", msg)
	}
}

// WarningSimple prints a warning message with "warning:" prefix colored but message uncolored.
// Use this for user-facing warnings where the message should stand out without full-line coloring.
func (w *Writer) WarningSimple(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		w.Errorln("%swarning:%s %s", yellow, reset, msg)
	} else {
		w.Errorln("warning: %s", msg)
	}
}

// SummaryHeader prints a summary section header.
func (w *Writer) SummaryHeader(title string) {
	w.Println("")
	if w.color {
		w.Println("%s=== %s ===%s", bold+cyan, title, reset)
	} else {
		w.Println("=== %s ===", title)
	}
	w.Println("")
}

// SummaryItem prints a labeled summary item with value.
func (w *Writer) SummaryItem(label, value string) {
	if w.color {
		w.Println("  %s%s:%s %s", dim, label, reset, value)
	} else {
		w.Println("  %s: %s", label, value)
	}
}

// SummaryPassed prints a passed/success items summary.
func (w *Writer) SummaryPassed(label, value string) {
	if w.color {
		w.Println("  %s%s:%s %s%s%s", dim, label, reset, green, value, reset)
	} else {
		w.Println("  %s: %s", label, value)
	}
}

// SummaryFailed prints a failed items summary.
func (w *Writer) SummaryFailed(label, value string) {
	if w.color {
		w.Println("  %s%s:%s %s%s%s", dim, label, reset, red, value, reset)
	} else {
		w.Println("  %s: %s", label, value)
	}
}

// FinalSuccess prints a final success message.
func (w *Writer) FinalSuccess(format string, args ...interface{}) {
	w.Println("")
	msg := fmt.Sprintf(format, args...)
	if w.color {
		w.Println("%s%s%s", green, msg, reset)
	} else {
		w.Println("%s", msg)
	}
}

// FinalFailure prints a final failure message.
func (w *Writer) FinalFailure(format string, args ...interface{}) {
	w.Println("")
	msg := fmt.Sprintf(format, args...)
	if w.color {
		w.Println("%s%s%s", red, msg, reset)
	} else {
		w.Println("%s", msg)
	}
}

// DryRunStart prints the dry run header.
func (w *Writer) DryRunStart() {
	w.Println("")
	if w.color {
		w.Println("%s=== DRY RUN ===%s", bold+yellow, reset)
	} else {
		w.Println("=== DRY RUN ===")
	}
	w.Println("")
}

// DryRunEnd prints the dry run footer.
func (w *Writer) DryRunEnd() {
	w.Println("")
	if w.color {
		w.Println("%s=== END DRY RUN ===%s", bold+yellow, reset)
	} else {
		w.Println("=== END DRY RUN ===")
	}
}

// PhaseHeader prints a CI/build phase header.
func (w *Writer) PhaseHeader(phase string) {
	w.Println("")
	if w.color {
		w.Println("%s=== %s ===%s", bold+blue, phase, reset)
	} else {
		w.Println("=== %s ===", phase)
	}
}

// TargetInfo prints target information line.
func (w *Writer) TargetInfo(name, targetType, title string) {
	if w.color {
		w.Println("%s%s%s (%s): %s", cyan+bold, name, reset, targetType, title)
	} else {
		w.Println("%s (%s): %s", name, targetType, title)
	}
}

// TargetDetail prints an indented target detail.
func (w *Writer) TargetDetail(label, value string) {
	if w.color {
		w.Println("  %s%s:%s %s", dim, label, reset, value)
	} else {
		w.Println("  %s: %s", label, value)
	}
}

// ValidationSuccess prints a validation success message.
func (w *Writer) ValidationSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		w.Println("%s%s%s %s", green, "✓", reset, msg)
	} else {
		w.Println("%s", msg)
	}
}

// Hint prints a hint message for the user.
func (w *Writer) Hint(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		w.Println("%s%s%s", dim, msg, reset)
	} else {
		w.Println("%s", msg)
	}
}

// UpdateNotification prints an update notification message.
func (w *Writer) UpdateNotification(version string) {
	if w.color {
		w.Errorln("%sstructyl %s available. Run 'structyl upgrade' to update.%s", dim, version, reset)
	} else {
		w.Errorln("structyl %s available. Run 'structyl upgrade' to update.", version)
	}
}

// SummaryAction prints an action item with status indicator, name, duration, and optional error.
// Used for detailed summaries showing individual targets or phases.
func (w *Writer) SummaryAction(name string, success bool, duration string, errMsg string) {
	if w.color {
		if success {
			w.Print("    %s✓%s %-12s %s%s%s", green, reset, name, dim, duration, reset)
		} else {
			w.Print("    %s✗%s %-12s %s%s%s", red, reset, name, dim, duration, reset)
			if errMsg != "" {
				w.Print("  %s(%s)%s", dim, errMsg, reset)
			}
		}
	} else {
		if success {
			w.Print("    + %-12s %s", name, duration)
		} else {
			w.Print("    x %-12s %s", name, duration)
			if errMsg != "" {
				w.Print("  (%s)", errMsg)
			}
		}
	}
	w.Print("\n")
}

// SummarySectionLabel prints a label for a summary section (e.g., "Targets:" or "Phases:").
func (w *Writer) SummarySectionLabel(label string) {
	if w.color {
		w.Println("  %s%s%s", dim, label, reset)
	} else {
		w.Println("  %s", label)
	}
}

// TaskResult is an alias for the shared model type.
type TaskResult = model.TaskResult

// TaskRunSummary is an alias for the shared model type.
type TaskRunSummary = model.TaskRunSummary

// FormatTestCounts formats test counts for display.
// Returns empty string if counts are nil or not parsed.
func FormatTestCounts(counts *testparser.TestCounts) string {
	if counts == nil || !counts.Parsed {
		return ""
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("%d passed", counts.Passed))
	if counts.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", counts.Failed))
	}
	if counts.Skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", counts.Skipped))
	}

	return strings.Join(parts, ", ")
}

// PrintTaskSummary prints a summary of task execution.
func (w *Writer) PrintTaskSummary(taskName string, summary *TaskRunSummary) {
	w.SummaryHeader(taskName + " Summary")

	// Print detailed task listing
	w.SummarySectionLabel("Tasks:")
	for _, t := range summary.Tasks {
		w.printTaskResultLine(t)
	}
	w.Println("")

	// Print aggregated test counts if available
	if summary.TestCounts != nil && summary.TestCounts.Parsed {
		w.printTestCountsSummary(summary.TestCounts)
	}

	// Print summary details
	w.SummaryItem("Total Tasks", fmt.Sprintf("%d", len(summary.Tasks)))
	w.SummaryPassed("Passed", fmt.Sprintf("%d", summary.Passed))
	if summary.Failed > 0 {
		// Collect failed task names
		var failedNames []string
		for _, t := range summary.Tasks {
			if !t.Success {
				failedNames = append(failedNames, t.Name)
			}
		}
		w.SummaryFailed("Failed", fmt.Sprintf("%d (%s)", summary.Failed, strings.Join(failedNames, ", ")))
	}
	w.SummaryItem("Duration", FormatDuration(summary.TotalDuration))

	// Final message
	if summary.Failed == 0 {
		w.FinalSuccess("All %d tasks completed successfully.", len(summary.Tasks))
	} else {
		w.FinalFailure("%d of %d tasks failed.", summary.Failed, len(summary.Tasks))
	}
}

// printTaskResultLine prints a single task result with optional test counts.
func (w *Writer) printTaskResultLine(t TaskResult) {
	duration := FormatDuration(t.Duration)
	testCountsStr := FormatTestCounts(t.TestCounts)

	w.printTaskStatus(t.Name, t.Success, duration)
	w.printTaskSuffix(testCountsStr, t.Error)
	w.Print("\n")
}

// printTaskStatus prints the status indicator, name, and duration for a task.
func (w *Writer) printTaskStatus(name string, success bool, duration string) {
	if w.color {
		indicator := green + "✓" + reset
		if !success {
			indicator = red + "✗" + reset
		}
		w.Print("    %s %-12s %s%s%s", indicator, name, dim, duration, reset)
	} else {
		indicator := "+"
		if !success {
			indicator = "x"
		}
		w.Print("    %s %-12s %s", indicator, name, duration)
	}
}

// printTaskSuffix prints the test counts or error suffix for a task.
func (w *Writer) printTaskSuffix(testCountsStr string, err error) {
	if testCountsStr != "" {
		if w.color {
			w.Print("    %s(%s)%s", dim, testCountsStr, reset)
		} else {
			w.Print("    (%s)", testCountsStr)
		}
		return
	}
	if err != nil {
		if w.color {
			w.Print("  %s(%s)%s", dim, err.Error(), reset)
		} else {
			w.Print("  (%s)", err.Error())
		}
	}
}

// printTestCountsSummary prints the aggregated test counts summary line.
func (w *Writer) printTestCountsSummary(counts *testparser.TestCounts) {
	if counts == nil || !counts.Parsed {
		return
	}

	var parts []string

	if counts.Passed > 0 {
		if w.color {
			parts = append(parts, fmt.Sprintf("%s%d passed%s", green, counts.Passed, reset))
		} else {
			parts = append(parts, fmt.Sprintf("%d passed", counts.Passed))
		}
	}

	if counts.Failed > 0 {
		if w.color {
			parts = append(parts, fmt.Sprintf("%s%d failed%s", red, counts.Failed, reset))
		} else {
			parts = append(parts, fmt.Sprintf("%d failed", counts.Failed))
		}
	}

	if counts.Skipped > 0 {
		if w.color {
			parts = append(parts, fmt.Sprintf("%s%d skipped%s", yellow, counts.Skipped, reset))
		} else {
			parts = append(parts, fmt.Sprintf("%d skipped", counts.Skipped))
		}
	}

	if len(parts) > 0 {
		w.Println("  Tests: %s", strings.Join(parts, ", "))
		w.Println("")
	}

	// Print failed test details if any
	w.printFailedTestDetails(counts.FailedTests)
}

// printFailedTestDetails prints detailed information about failed tests.
func (w *Writer) printFailedTestDetails(failedTests []testparser.FailedTest) {
	if len(failedTests) == 0 {
		return
	}

	w.SummarySectionLabel("Failed Tests:")
	for _, ft := range failedTests {
		if w.color {
			if ft.Reason != "" {
				w.Println("    %s✗%s %s: %s%s%s", red, reset, ft.Name, dim, ft.Reason, reset)
			} else {
				w.Println("    %s✗%s %s", red, reset, ft.Name)
			}
		} else {
			if ft.Reason != "" {
				w.Println("    x %s: %s", ft.Name, ft.Reason)
			} else {
				w.Println("    x %s", ft.Name)
			}
		}
	}
	w.Println("")
}

// FormatDuration formats a duration in a human-readable way.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", m, s)
}

// colorPlaceholders highlights <placeholder> patterns in text.
func (w *Writer) colorPlaceholders(text string) string {
	var result strings.Builder
	for {
		start := strings.Index(text, "<")
		if start == -1 {
			result.WriteString(text)
			break
		}
		end := strings.Index(text[start:], ">")
		if end == -1 {
			result.WriteString(text)
			break
		}
		result.WriteString(text[:start])
		result.WriteString(reset)
		result.WriteString(colorPlaceholder)
		result.WriteString(text[start : start+end+1])
		result.WriteString(reset)
		text = text[start+end+1:]
	}
	return result.String()
}
