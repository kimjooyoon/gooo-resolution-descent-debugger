package debugger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func LoadSource(path string) (Source, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Source{}, nil, err
	}
	source, err := ParseSource(string(raw))
	if err != nil {
		return Source{}, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := ValidateSource(source); err != nil {
		return Source{}, nil, err
	}
	return source, raw, nil
}

func ParseSource(input string) (Source, error) {
	var source Source
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	for lineNumber, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			continue
		}
		values := map[string]string{}
		switch tokens[0] {
		case "gooo", "authority", "precedence", "unknown_fields":
		default:
			parsed, parseErr := keyValues(tokens[1:])
			if parseErr != nil {
				return Source{}, fmt.Errorf("line %d: %w", lineNumber+1, parseErr)
			}
			values = parsed
		}
		intValue := func(key string) (int, error) {
			value, ok := values[key]
			if !ok {
				return 0, fmt.Errorf("line %d: missing %s", lineNumber+1, key)
			}
			parsed, parseErr := strconv.Atoi(value)
			if parseErr != nil {
				return 0, fmt.Errorf("line %d: %s must be an integer", lineNumber+1, key)
			}
			return parsed, nil
		}

		switch tokens[0] {
		case "gooo":
			if len(tokens) != 3 || tokens[1] != "resolution_descent_debugger" || tokens[2] != "v1" {
				return Source{}, fmt.Errorf("line %d: invalid gooo header", lineNumber+1)
			}
			source.Schema = SourceSchema
		case "authority":
			if len(tokens) != 2 {
				return Source{}, fmt.Errorf("line %d: invalid authority declaration", lineNumber+1)
			}
			source.Authority = tokens[1]
		case "origin":
			source.Origin = Origin{Source: values["source"], Contract: values["contract"], Fixture: values["fixture"]}
		case "resolution_lattice":
			source.ResolutionLevels = splitCSV(values["levels"])
		case "probe_capability":
			source.Capabilities = append(source.Capabilities, ProbeCapability{ID: values["id"], Input: values["input"], Observation: values["observation"]})
		case "probe_effect":
			writes, parseErr := intValue("repository_writes")
			if parseErr != nil {
				return Source{}, parseErr
			}
			source.Effects = append(source.Effects, ProbeEffect{ID: values["id"], Kind: values["kind"], RepositoryWrites: writes})
		case "descent_rule":
			ordinal, parseErr := intValue("ordinal")
			if parseErr != nil {
				return Source{}, parseErr
			}
			source.DescentRules = append(source.DescentRules, DescentRule{Ordinal: ordinal, From: values["from"], To: values["to"], Capability: values["capability"], Effect: values["effect"]})
		case "claim_transition":
			ordinal, parseErr := intValue("ordinal")
			if parseErr != nil {
				return Source{}, parseErr
			}
			source.Transitions = append(source.Transitions, ClaimTransitionRule{Ordinal: ordinal, From: values["from"], To: values["to"], Evidence: values["evidence"]})
		case "decision":
			source.Statuses = splitCSV(values["statuses"])
		case "precedence":
			if len(tokens) != 2 {
				return Source{}, fmt.Errorf("line %d: invalid precedence declaration", lineNumber+1)
			}
			source.Precedence = strings.Split(tokens[1], ">")
		case "unknown_fields":
			if len(tokens) != 2 {
				return Source{}, fmt.Errorf("line %d: invalid unknown field declaration", lineNumber+1)
			}
			source.UnknownFields = strings.Split(tokens[1], ",")
		case "fixed_denominator":
			cases, parseErr := intValue("cases")
			if parseErr != nil {
				return Source{}, parseErr
			}
			source.Denominator = DenominatorDecl{ID: values["id"], Cases: cases, Unit: values["unit"]}
		case "scenario":
			ordinal, parseErr := intValue("ordinal")
			if parseErr != nil {
				return Source{}, parseErr
			}
			source.Scenarios = append(source.Scenarios, Scenario{Ordinal: ordinal, ID: values["id"], Expected: values["expected"], Fixture: values["fixture"]})
		default:
			return Source{}, fmt.Errorf("line %d: unknown declaration %q", lineNumber+1, tokens[0])
		}
	}
	return source, nil
}

func ValidateSource(source Source) error {
	if source.Schema != SourceSchema || source.Authority != "metacode" {
		return fmt.Errorf(".gooo must declare %s and metacode authority", SourceSchema)
	}
	if !equalStrings(source.ResolutionLevels, ResolutionLevels) {
		return fmt.Errorf("resolution lattice must be STAGE,STEP,REASON,BLOCKED_BY")
	}
	if source.Origin.Source == "" || source.Origin.Contract == "" || source.Origin.Fixture != "caller_owned" {
		return fmt.Errorf("origin must bind source, contract, and caller_owned fixture")
	}
	if !equalStrings(source.Statuses, RequiredStatuses) || !equalStrings(source.Precedence, RequiredPrecedence) {
		return fmt.Errorf("decision statuses or precedence are not declared exactly")
	}
	if !equalStrings(source.UnknownFields, RequiredUnknownFields) {
		return fmt.Errorf("UNKNOWN must declare the required six fields in order")
	}
	if source.Denominator.ID == "" || source.Denominator.Cases != 7 || source.Denominator.Unit != "scenario" || len(source.Scenarios) != source.Denominator.Cases {
		return fmt.Errorf("fixed denominator must declare exactly seven scenarios")
	}
	capabilities := map[string]bool{}
	for _, capability := range source.Capabilities {
		if capability.ID == "" || capability.Input != "caller_owned_fixture" || capability.Observation == "" || capabilities[capability.ID] {
			return fmt.Errorf("invalid or duplicate probe capability %q", capability.ID)
		}
		capabilities[capability.ID] = true
	}
	if !capabilities["fixture_lookup"] || !capabilities["source_digest_check"] {
		return fmt.Errorf("required fixture lookup and source digest capabilities are missing")
	}
	effects := map[string]ProbeEffect{}
	for _, effect := range source.Effects {
		if effect.ID == "" || effect.Kind == "" || effect.RepositoryWrites < 0 || effects[effect.ID].ID != "" {
			return fmt.Errorf("invalid or duplicate probe effect %q", effect.ID)
		}
		effects[effect.ID] = effect
	}
	if effects["read_only"].Kind != "READ_ONLY" || effects["read_only"].RepositoryWrites != 0 || effects["forbidden_write"].Kind != "FORBIDDEN" || effects["forbidden_write"].RepositoryWrites != 1 {
		return fmt.Errorf("required read_only and forbidden_write effects are missing")
	}
	seenRules := map[int]bool{}
	for _, rule := range source.DescentRules {
		if rule.Ordinal < 1 || seenRules[rule.Ordinal] || rule.From == "" || rule.To == "" || !contains(ResolutionLevels, rule.From) || !contains(ResolutionLevels, rule.To) || !capabilities[rule.Capability] || effects[rule.Effect].ID == "" {
			return fmt.Errorf("invalid descent rule %d", rule.Ordinal)
		}
		seenRules[rule.Ordinal] = true
	}
	if len(source.DescentRules) != 4 {
		return fmt.Errorf("exactly four descent rules are required")
	}
	seenTransitions := map[int]bool{}
	for _, transition := range source.Transitions {
		if transition.Ordinal < 1 || seenTransitions[transition.Ordinal] || transition.From != LifecycleOpen || (transition.To != LifecycleOpen && transition.To != LifecycleDischarged && transition.To != LifecycleRefuted) || transition.Evidence == "" {
			return fmt.Errorf("invalid claim transition %d", transition.Ordinal)
		}
		seenTransitions[transition.Ordinal] = true
	}
	if len(source.Transitions) != 3 {
		return fmt.Errorf("exactly three claim transitions are required")
	}
	seenScenarios := map[string]bool{}
	for index, scenario := range source.Scenarios {
		if scenario.Ordinal != index+1 || scenario.ID == "" || seenScenarios[scenario.ID] || !contains(RequiredStatuses, scenario.Expected) || scenario.Fixture == "" {
			return fmt.Errorf("invalid scenario %q", scenario.ID)
		}
		seenScenarios[scenario.ID] = true
	}
	return nil
}

func LoadDenominator(path string) (Denominator, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Denominator{}, nil, err
	}
	var denominator Denominator
	if err := json.Unmarshal(raw, &denominator); err != nil {
		return Denominator{}, nil, fmt.Errorf("parse denominator: %w", err)
	}
	if err := ValidateDenominator(denominator); err != nil {
		return Denominator{}, nil, err
	}
	return denominator, raw, nil
}

func ValidateDenominator(denominator Denominator) error {
	if denominator.Schema != "gooo/resolution-descent-debugger/denominator/v1" || denominator.DenominatorID != "resolution-descent-debugger-v1" || denominator.ScenarioCount != 7 || !denominator.RootReadmeInventoryExcluded || !equalStrings(denominator.UnknownFields, RequiredUnknownFields) || !equalStrings(denominator.Precedence, RequiredPrecedence) {
		return fmt.Errorf("denominator declaration is incomplete")
	}
	if denominator.Authority.Source != "metacode" || denominator.Authority.InputRepositoryWrites != 0 || denominator.Authority.AutomaticCommit != 0 || denominator.Authority.AutomaticPush != 0 || denominator.Authority.AutomaticMerge != 0 || denominator.Authority.AutomaticRelease != 0 {
		return fmt.Errorf("denominator authority boundary is invalid")
	}
	if len(denominator.Scenarios) != 7 {
		return fmt.Errorf("denominator must contain seven scenarios")
	}
	for index, scenario := range denominator.Scenarios {
		if scenario.Ordinal != index+1 || scenario.ID == "" || !contains(RequiredStatuses, scenario.Expected) {
			return fmt.Errorf("invalid denominator scenario %d", index+1)
		}
	}
	return nil
}

func ValidateBindings(source Source, denominator Denominator) error {
	if source.Denominator.ID != denominator.DenominatorID || source.Denominator.Cases != denominator.ScenarioCount || len(source.Scenarios) != len(denominator.Scenarios) {
		return fmt.Errorf("source and denominator identity mismatch")
	}
	for index, scenario := range source.Scenarios {
		expected := denominator.Scenarios[index]
		if scenario.Ordinal != expected.Ordinal || scenario.ID != expected.ID || scenario.Expected != expected.Expected {
			return fmt.Errorf("scenario %d does not match fixed denominator", index+1)
		}
	}
	return nil
}

func LoadFixture(path string) (Fixture, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Fixture{}, nil, err
	}
	var fixture Fixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return Fixture{}, nil, fmt.Errorf("parse fixture %s: %w", path, err)
	}
	if err := ValidateFixture(fixture); err != nil {
		return Fixture{}, nil, err
	}
	return fixture, raw, nil
}

func ValidateFixture(fixture Fixture) error {
	if fixture.Schema != FixtureSchema || fixture.CaseID == "" || fixture.Subject == "" || fixture.Claim.ID == "" || fixture.Claim.State != LifecycleOpen || fixture.Probe.ID == "" || fixture.Probe.Operation == "" || fixture.Probe.Capability == "" || fixture.Probe.Effect == "" || fixture.Probe.Observation == "" || fixture.Probe.Evidence == "" {
		return fmt.Errorf("fixture is incomplete or not OPEN")
	}
	if !contains(ResolutionLevels, fixture.StartResolution) || !contains(ResolutionLevels, fixture.Claim.Resolution) {
		return fmt.Errorf("fixture resolution is outside the declared lattice")
	}
	if fixture.Claim.Stage == "" || fixture.Claim.Step == "" || fixture.Claim.Reason == "" || fixture.Claim.UnknownClass == "" || fixture.Claim.NextOperation == "" || len(fixture.Claim.BlockedBy) == 0 {
		return fmt.Errorf("fixture UNKNOWN claim does not carry the six required fields")
	}
	if fixture.Probe.TargetResolution != "" && !contains(ResolutionLevels, fixture.Probe.TargetResolution) {
		return fmt.Errorf("probe target resolution is outside the declared lattice")
	}
	return nil
}

func DigestBytes(raw []byte) string {
	digest := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func DigestValue(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return DigestBytes(raw), nil
}

func keyValues(tokens []string) (map[string]string, error) {
	values := make(map[string]string, len(tokens))
	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("expected key=value token, got %q", token)
		}
		values[parts[0]] = strings.Trim(parts[1], "\"")
	}
	return values, nil
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sortedCopy(values []string) []string {
	result := append([]string(nil), values...)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j] < result[i] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
