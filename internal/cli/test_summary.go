package cli

import (
	"fmt"
	"os"

	"github.com/AndreyAkinshin/structyl/internal/testparser"
)

// cmdTestSummary parses go test -json output and prints a summary.
func cmdTestSummary(args []string) int {
	// Check for help
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		printTestSummaryUsage()
		return 0
	}

	// Determine input source
	var input *os.File
	var err error

	if len(args) > 0 && args[0] != "-" {
		// Read from file
		input, err = os.Open(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "structyl test-summary: %v\n", err)
			return 1
		}
		defer func() { _ = input.Close() }()
	} else {
		// Read from stdin
		input = os.Stdin
	}

	// Parse the JSON output
	parser := &testparser.JSONParser{}
	counts := parser.ParseJSON(input)

	if !counts.Parsed {
		fmt.Fprintln(os.Stderr, "structyl test-summary: no test results found in input")
		fmt.Fprintln(os.Stderr, "hint: use 'go test -json ./...' to produce JSON output")
		return 1
	}

	// Print summary
	printTestSummary(&counts)

	// Return non-zero if there were failures
	if counts.Failed > 0 {
		return 1
	}
	return 0
}

// printTestSummary prints a formatted test summary.
func printTestSummary(counts *testparser.TestCounts) {
	out.Println("")
	out.SummaryHeader("Test Summary")

	// Print counts
	out.SummaryPassed("Passed", fmt.Sprintf("%d", counts.Passed))
	if counts.Failed > 0 {
		out.SummaryFailed("Failed", fmt.Sprintf("%d", counts.Failed))
	}
	if counts.Skipped > 0 {
		out.SummaryItem("Skipped", fmt.Sprintf("%d", counts.Skipped))
	}
	out.SummaryItem("Total", fmt.Sprintf("%d", counts.Total))

	// Print failed test details
	if len(counts.FailedTests) > 0 {
		out.Println("")
		out.SummarySectionLabel("Failed Tests:")
		for _, ft := range counts.FailedTests {
			if ft.Reason != "" {
				out.SummaryFailed("  "+ft.Name, ft.Reason)
			} else {
				out.SummaryFailed("  "+ft.Name, "")
			}
		}
	}

	out.Println("")

	// Final message
	if counts.Failed == 0 {
		out.FinalSuccess("All %d tests passed.", counts.Total)
	} else {
		out.FinalFailure("%d of %d tests failed.", counts.Failed, counts.Total)
	}
}

func printTestSummaryUsage() {
	out.HelpTitle("structyl test-summary - parse and summarize go test -json output")
	out.HelpSection("Usage:")
	out.HelpUsage("go test -json ./... | structyl test-summary")
	out.HelpUsage("go test -json ./... 2>&1 | tee test.json && structyl test-summary test.json")
	out.HelpSection("Description:")
	out.Println("  Parses go test -json output and prints a clear summary of test results,")
	out.Println("  highlighting any failed tests with their failure reasons.")
	out.Println("")
	out.HelpSection("Examples:")
	out.HelpExample("go test -json ./... | structyl test-summary", "Parse from stdin")
	out.HelpExample("structyl test-summary test-output.json", "Parse from file")
	out.Println("")
}
