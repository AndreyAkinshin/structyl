// Package output provides formatted output utilities for the CLI.
package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Writer handles CLI output formatting.
type Writer struct {
	out   io.Writer
	err   io.Writer
	color bool
	quiet bool
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

// Print writes to stdout.
func (w *Writer) Print(format string, args ...interface{}) {
	fmt.Fprintf(w.out, format, args...)
}

// Println writes a line to stdout.
func (w *Writer) Println(format string, args ...interface{}) {
	fmt.Fprintf(w.out, format+"\n", args...)
}

// Error writes to stderr.
func (w *Writer) Error(format string, args ...interface{}) {
	fmt.Fprintf(w.err, format, args...)
}

// Errorln writes a line to stderr.
func (w *Writer) Errorln(format string, args ...interface{}) {
	fmt.Fprintf(w.err, format+"\n", args...)
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
	if w.color {
		w.Println("\033[32m"+format+"\033[0m", args...)
	} else {
		w.Println(format, args...)
	}
}

// Warning prints a warning message.
func (w *Writer) Warning(format string, args ...interface{}) {
	if w.color {
		w.Errorln("\033[33mwarning: "+format+"\033[0m", args...)
	} else {
		w.Errorln("warning: "+format, args...)
	}
}

// TargetStart prints the start of a target command with enhanced visibility.
func (w *Writer) TargetStart(target, command string) {
	if w.quiet {
		return
	}
	// Empty line for visual separation
	w.Println("")
	label := fmt.Sprintf("─── [%s] %s ───", target, command)
	if w.color {
		w.Println("%s%s%s", bold+cyan, label, reset)
	} else {
		w.Println("%s", label)
	}
}

// TargetSuccess prints target command success.
func (w *Writer) TargetSuccess(target, command string) {
	if w.quiet {
		return
	}
	if w.color {
		w.Println("\033[32m[%s]\033[0m %s \033[32m✓\033[0m", target, command)
	} else {
		w.Println("[%s] %s done", target, command)
	}
}

// TargetFailed prints target command failure.
func (w *Writer) TargetFailed(target, command string, err error) {
	if w.color {
		w.Errorln("\033[31m[%s] %s failed:\033[0m %v", target, command, err)
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
	if w.color {
		w.Println("\033[1m=== %s ===\033[0m", title)
	} else {
		w.Println("=== %s ===", title)
	}
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
	if w.color {
		w.Println("%s%s%s", colorTitle, title, reset)
	} else {
		w.Println("%s", title)
	}
}

// HelpSection formats a section header (e.g., "Meta Commands:").
func (w *Writer) HelpSection(title string) {
	w.Println("")
	if w.color {
		w.Println("%s%s%s", colorSection, title, reset)
	} else {
		w.Println("%s", title)
	}
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

// WarningSimple prints a warning message without the "warning:" prefix.
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

// colorPlaceholders highlights <placeholder> patterns in text.
func (w *Writer) colorPlaceholders(text string) string {
	var result strings.Builder
	i := 0
	for i < len(text) {
		if text[i] == '<' {
			// Find closing >
			end := strings.Index(text[i:], ">")
			if end != -1 {
				// Found a placeholder
				placeholder := text[i : i+end+1]
				result.WriteString(reset)
				result.WriteString(colorPlaceholder)
				result.WriteString(placeholder)
				result.WriteString(reset)
				i += end + 1
				continue
			}
		}
		result.WriteByte(text[i])
		i++
	}
	return result.String()
}
