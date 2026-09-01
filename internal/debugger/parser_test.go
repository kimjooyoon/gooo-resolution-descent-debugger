package debugger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAuthoritativeSource(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, ".gooo", "resolution-descent-debugger.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	source, err := ParseSource(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSource(source); err != nil {
		t.Fatal(err)
	}
	if len(source.Scenarios) != 7 || len(source.DescentRules) != 4 || len(source.Transitions) != 3 {
		t.Fatalf("unexpected source shape: scenarios=%d rules=%d transitions=%d", len(source.Scenarios), len(source.DescentRules), len(source.Transitions))
	}
}

func TestUnknownTupleRequiresAllFields(t *testing.T) {
	valid := UnknownRecord{Stage: "stage", Step: "step", Reason: "reason", UnknownClass: "DIRECT_MISSING", NextOperation: "probe", BlockedBy: []string{"id"}}
	if !validUnknown(valid) {
		t.Fatal("complete UNKNOWN tuple was rejected")
	}
	valid.BlockedBy = nil
	if validUnknown(valid) {
		t.Fatal("empty blocked_by was accepted")
	}
}
