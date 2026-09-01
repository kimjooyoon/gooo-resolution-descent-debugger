package debugger

import (
	"path/filepath"
	"testing"
)

func TestFixedConformanceVector(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	output := t.TempDir()
	report, err := ExecuteConformance(root, output, filepath.Join(root, ".gooo", "resolution-descent-debugger.gooo"), filepath.Join(root, "contracts", "denominator-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionRefuted || report.Summary.CasesTotal != 7 || report.Summary.Closed != 3 || report.Summary.Unknown != 2 || report.Summary.Refuted != 2 {
		t.Fatalf("unexpected vector result: %#v", report.Summary)
	}
	if report.Summary.UnknownFrontierBefore != 9 || report.Summary.UnknownFrontierAfter != 3 {
		t.Fatalf("unexpected frontier cardinalities: %#v", report.Summary)
	}
	if report.Summary.ReplayComparisons != 1 || report.Summary.ReplayMismatches != 0 {
		t.Fatalf("unexpected replay result: %#v", report.Summary)
	}
}
