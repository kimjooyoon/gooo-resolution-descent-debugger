package debugger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func Lower(source Source, sourcePath, contractPath, sourceDigest, contractDigest string) SemanticIR {
	return SemanticIR{
		Schema:           IRSchema,
		SourcePath:       sourcePath,
		SourceDigest:     sourceDigest,
		ContractPath:     contractPath,
		ContractDigest:   contractDigest,
		ResolutionLevels: append([]string(nil), source.ResolutionLevels...),
		Capabilities:     append([]ProbeCapability(nil), source.Capabilities...),
		Effects:          append([]ProbeEffect(nil), source.Effects...),
		DescentRules:     append([]DescentRule(nil), source.DescentRules...),
		Transitions:      append([]ClaimTransitionRule(nil), source.Transitions...),
		Scenarios:        append([]Scenario(nil), source.Scenarios...),
	}
}

func Execute(root, output string, source Source, sourcePath, contractPath string, sourceRaw, contractRaw []byte, fixturePaths []string) (Report, error) {
	if err := ensureCallerOutput(root, output); err != nil {
		return Report{}, err
	}
	if err := ValidateSource(source); err != nil {
		return Report{}, err
	}
	sourceDigest := DigestBytes(sourceRaw)
	contractDigest := DigestBytes(contractRaw)
	ir := Lower(source, sourcePath, contractPath, sourceDigest, contractDigest)
	started := time.Now()
	caseReports := make([]CaseReport, 0, len(fixturePaths))
	for _, fixturePath := range fixturePaths {
		fixture, raw, err := LoadFixture(fixturePath)
		if err != nil {
			return Report{}, err
		}
		caseReport, err := executeCase(source, ir, fixture, raw)
		if err != nil {
			return Report{}, fmt.Errorf("case %s: %w", fixture.CaseID, err)
		}
		caseReports = append(caseReports, caseReport)
	}
	if len(caseReports) != len(source.Scenarios) {
		return Report{}, fmt.Errorf("expected %d fixtures, observed %d", len(source.Scenarios), len(caseReports))
	}
	summary := summarize(caseReports)
	decision, improvement := finalDecisionAndImprovement(caseReports, sourceDigest, contractDigest)
	inv, err := inventory(root)
	if err != nil {
		return Report{}, err
	}
	report := Report{
		Schema:           ReportSchema,
		Decision:         decision,
		SourcePath:       sourcePath,
		SourceDigest:     sourceDigest,
		ContractPath:     contractPath,
		ContractDigest:   contractDigest,
		ToolchainDigest:  ToolchainDigest,
		RunnerDigest:     RunnerDigest,
		ResolutionLevels: append([]string(nil), source.ResolutionLevels...),
		Precedence:       append([]string(nil), source.Precedence...),
		UnknownFields:    append([]string(nil), source.UnknownFields...),
		Cases:            caseReports,
		Summary:          summary,
		Authority:        zeroAuthority(),
		Improvement:      improvement,
		Inventory:        inv,
		RepositoryWrites: 0,
	}
	report.Metrics = Metrics{
		Schema:           MetricsSchema,
		Inventory:        inv,
		Generated:        generatedMetricForCases(caseReports),
		Stages:           StageMetrics{Conformance: StageMetric{WallMS: int(time.Since(started).Milliseconds()), PeakRSSKiB: processRSSKiB()}},
		Tests:            TestMetrics{Total: summary.TestsTotal, Selected: summary.TestsSelected, Executed: summary.TestsExecuted, Reused: summary.TestsReused, Failed: summary.TestsFailed, Unknown: summary.TestsUnknown},
		RepositoryWrites: 0,
		Authority:        zeroAuthority(),
	}
	if err := validateReport(source, report); err != nil {
		return Report{}, err
	}
	if err := writeOutputs(output, ir, report); err != nil {
		return Report{}, err
	}
	return report, nil
}

func ExecuteConformance(root, output, sourcePath, contractPath string) (ConformanceReport, error) {
	source, sourceRaw, err := LoadSource(sourcePath)
	if err != nil {
		return ConformanceReport{}, err
	}
	denominator, contractRaw, err := LoadDenominator(contractPath)
	if err != nil {
		return ConformanceReport{}, err
	}
	if err := ValidateBindings(source, denominator); err != nil {
		return ConformanceReport{}, err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return ConformanceReport{}, err
	}
	fixturePaths := make([]string, 0, len(source.Scenarios))
	for _, scenario := range source.Scenarios {
		fixturePath := scenario.Fixture
		if !filepath.IsAbs(fixturePath) {
			fixturePath = filepath.Join(rootAbs, fixturePath)
		}
		fixturePaths = append(fixturePaths, fixturePath)
	}
	report, err := Execute(root, output, source, sourcePath, contractPath, sourceRaw, contractRaw, fixturePaths)
	if err != nil {
		return ConformanceReport{}, err
	}
	for index, scenario := range source.Scenarios {
		if report.Cases[index].CaseID != scenario.ID || report.Cases[index].Expected != scenario.Expected {
			return ConformanceReport{}, fmt.Errorf("case order does not match .gooo scenario %d", index+1)
		}
	}
	conformance := ConformanceReport{Schema: ConformanceSchema, Decision: report.Decision, FixedDenominator: denominator.ScenarioCount, Summary: report.Summary, Cases: report.Cases, Authority: report.Authority, RepositoryWrites: 0}
	if err := WriteJSON(filepath.Join(output, "conformance-report.json"), conformance); err != nil {
		return ConformanceReport{}, err
	}
	return conformance, nil
}

func ExecuteSingle(root, output, sourcePath, contractPath, fixturePath string) (Report, error) {
	source, sourceRaw, err := LoadSource(sourcePath)
	if err != nil {
		return Report{}, err
	}
	_, contractRaw, err := LoadDenominator(contractPath)
	if err != nil {
		return Report{}, err
	}
	return Execute(root, output, source, sourcePath, contractPath, sourceRaw, contractRaw, []string{fixturePath})
}

func executeCase(source Source, ir SemanticIR, fixture Fixture, fixtureRaw []byte) (CaseReport, error) {
	if !containsScenario(source.Scenarios, fixture.CaseID) {
		return CaseReport{}, fmt.Errorf("fixture is not in the fixed denominator")
	}
	probeIR := ProbeIR{
		Schema:           ProbeIRSchema,
		CaseID:           fixture.CaseID,
		ClaimID:          fixture.Claim.ID,
		Operation:        fixture.Probe.Operation,
		Capability:       fixture.Probe.Capability,
		Effect:           fixture.Probe.Effect,
		FromResolution:   fixture.Claim.Resolution,
		ToResolution:     fixture.Probe.TargetResolution,
		Observation:      fixture.Probe.Observation,
		ExpectedIDs:      sortedCopy(fixture.Probe.ExpectedIDs),
		ObservedIDs:      sortedCopy(fixture.Probe.ObservedIDs),
		NarrowedFrontier: sortedCopy(fixture.Probe.NarrowedFrontier),
		SourceDigest:     ir.SourceDigest,
		ContractDigest:   ir.ContractDigest,
		FixtureDigest:    DigestBytes(fixtureRaw),
		RunnerDigest:     RunnerDigest,
		ToolchainDigest:  ToolchainDigest,
	}
	evidence, err := runTypedProbe(source, ir, fixture, probeIR)
	if err != nil {
		return CaseReport{}, err
	}
	decision, transition := transitionFor(fixture, evidence)
	if fixture.Probe.Observation == ObservationReplay {
		first, firstErr := runTypedProbe(source, ir, fixture, probeIR)
		if firstErr != nil {
			return CaseReport{}, firstErr
		}
		if first.EvidenceDigest != evidence.EvidenceDigest || first.Accepted != evidence.Accepted {
			return CaseReport{}, fmt.Errorf("replay evidence is not byte-equivalent")
		}
	}
	afterBlockedBy := []string{}
	if decision == DecisionUnknown {
		afterBlockedBy = sortedCopy(fixture.Claim.BlockedBy)
	}
	caseReport := CaseReport{
		Ordinal:                          scenarioOrdinal(source.Scenarios, fixture.CaseID),
		CaseID:                           fixture.CaseID,
		Expected:                         scenarioExpected(source.Scenarios, fixture.CaseID),
		Subject:                          fixture.Subject,
		StartResolution:                  fixture.StartResolution,
		Decision:                         decision,
		ClosureBasis:                     closureBasis(decision, evidence),
		OriginalClaim:                    fixture.Claim,
		Claims:                           []Claim{fixture.Claim},
		ProbeIR:                          probeIR,
		ProbeEvidence:                    evidence,
		ClaimTransitions:                 []ClaimTransition{transition},
		BeforeUnknownFrontierCardinality: len(fixture.Claim.BlockedBy),
		AfterUnknownFrontierCardinality:  len(afterBlockedBy),
		BeforeBlockedBy:                  sortedCopy(fixture.Claim.BlockedBy),
		AfterBlockedBy:                   afterBlockedBy,
		NarrowedFrontier:                 sortedCopy(fixture.Probe.NarrowedFrontier),
		ReplayEqual:                      fixture.Probe.Observation != ObservationReplay || evidence.Accepted,
		Improvement:                      caseImprovement(fixture.CaseID, fixture.Claim, decision, ir, evidence, len(afterBlockedBy)),
	}
	if fixture.Probe.Observation == ObservationReplay {
		caseReport.ReplayDigest = evidence.EvidenceDigest
	}
	return caseReport, nil
}

func runTypedProbe(source Source, ir SemanticIR, fixture Fixture, probeIR ProbeIR) (ProbeEvidence, error) {
	capability, foundCapability := findCapability(source, probeIR.Capability)
	effect, foundEffect := findEffect(source, probeIR.Effect)
	if !foundCapability || !foundEffect {
		return ProbeEvidence{}, fmt.Errorf("probe capability or effect is undeclared")
	}
	evidence := ProbeEvidence{
		ID:               probeIR.CaseID + ":" + probeIR.Operation,
		Observation:      probeIR.Observation,
		Capability:       probeIR.Capability,
		Effect:           probeIR.Effect,
		ObservedIDs:      append([]string(nil), probeIR.ObservedIDs...),
		ExpectedIDs:      append([]string(nil), probeIR.ExpectedIDs...),
		NarrowedFrontier: append([]string(nil), probeIR.NarrowedFrontier...),
		InputDigest:      probeIR.FixtureDigest,
		RunnerDigest:     probeIR.RunnerDigest,
		ToolchainDigest:  probeIR.ToolchainDigest,
	}
	if capability.Input != "caller_owned_fixture" {
		return ProbeEvidence{}, fmt.Errorf("probe capability does not bind caller-owned fixture")
	}
	if effect.RepositoryWrites != 0 || effect.Kind != "READ_ONLY" || probeIR.Effect != "read_only" || probeIR.Observation == ObservationForbidden {
		evidence.Contradiction = true
		evidence.Reason = "FORBIDDEN_PROBE_EFFECT"
		return finishEvidence(evidence)
	}
	if probeIR.Observation == ObservationStale {
		evidence.Contradiction = true
		evidence.Reason = "STALE_SOURCE_DIGEST_CONTRADICTS_OPEN_CLAIM"
		return finishEvidence(evidence)
	}
	if probeIR.Observation == ObservationAmbiguous {
		evidence.Reason = "AMBIGUOUS_PROBE_EVIDENCE"
		return finishEvidence(evidence)
	}
	if probeIR.Observation == ObservationUnbounded {
		evidence.Reason = "NO_LOWER_RESOLUTION_NODE"
		return finishEvidence(evidence)
	}
	if probeIR.Observation != ObservationMatch && probeIR.Observation != ObservationDependencyMatch && probeIR.Observation != ObservationReplay {
		return ProbeEvidence{}, fmt.Errorf("unknown typed probe observation %q", probeIR.Observation)
	}
	if !hasDescentRule(source, probeIR.FromResolution, probeIR.ToResolution, probeIR.Capability, probeIR.Effect) {
		return ProbeEvidence{}, fmt.Errorf("probe descent is not declared by .gooo")
	}
	if !equalStrings(sortedCopy(probeIR.ExpectedIDs), sortedCopy(probeIR.ObservedIDs)) {
		evidence.Reason = "PROBE_IDENTITY_INCOMPLETE"
		return finishEvidence(evidence)
	}
	evidence.Accepted = true
	evidence.Reason = "MATCHING_TYPED_PROBE_EVIDENCE"
	return finishEvidence(evidence)
}

func finishEvidence(evidence ProbeEvidence) (ProbeEvidence, error) {
	digest, err := DigestValue(struct {
		ID            string   `json:"id"`
		Observation   string   `json:"observation"`
		Accepted      bool     `json:"accepted"`
		Contradiction bool     `json:"contradiction"`
		Capability    string   `json:"capability"`
		Effect        string   `json:"effect"`
		ObservedIDs   []string `json:"observed_ids"`
		ExpectedIDs   []string `json:"expected_ids"`
		Reason        string   `json:"reason"`
		InputDigest   string   `json:"input_digest"`
	}{evidence.ID, evidence.Observation, evidence.Accepted, evidence.Contradiction, evidence.Capability, evidence.Effect, evidence.ObservedIDs, evidence.ExpectedIDs, evidence.Reason, evidence.InputDigest})
	if err != nil {
		return ProbeEvidence{}, err
	}
	evidence.EvidenceDigest = digest
	return evidence, nil
}

func transitionFor(fixture Fixture, evidence ProbeEvidence) (string, ClaimTransition) {
	decision := DecisionUnknown
	to := LifecycleOpen
	reason := evidence.Reason
	next := fixture.Claim.NextOperation
	blockedBy := sortedCopy(fixture.Claim.BlockedBy)
	if evidence.Contradiction {
		decision = DecisionRefuted
		to = LifecycleRefuted
		if evidence.Reason == "FORBIDDEN_PROBE_EFFECT" {
			next = "REVOKE_FORBIDDEN_PROBE_EFFECT"
		} else {
			next = "RESTORE_CURRENT_SOURCE_BINDING"
		}
		blockedBy = []string{}
	} else if evidence.Accepted {
		decision = DecisionClosed
		to = LifecycleDischarged
		reason = "LOWER_RESOLUTION_PROBE_ACCEPTED"
		next = "NONE"
		blockedBy = []string{}
	}
	transition := ClaimTransition{
		Sequence:      1,
		ClaimID:       fixture.Claim.ID,
		From:          LifecycleOpen,
		To:            to,
		EvidenceID:    evidence.ID,
		Stage:         "PROBE",
		Step:          "EXECUTE_TYPED_PROBE",
		Reason:        reason,
		UnknownClass:  fixture.Claim.UnknownClass,
		NextOperation: next,
		BlockedBy:     blockedBy,
		AppendOnly:    true,
	}
	return decision, transition
}

func caseImprovement(caseID string, claim Claim, decision string, ir SemanticIR, evidence ProbeEvidence, after int) Improvement {
	pair := ExactPair{ScenarioID: caseID, SourceDigest: ir.SourceDigest, ContractDigest: ir.ContractDigest, ToolchainDigest: ToolchainDigest, RunnerDigest: RunnerDigest, ExactIdentity: true, Before: len(claim.BlockedBy), After: after}
	if decision == DecisionClosed && evidence.Accepted && pair.After < pair.Before {
		return Improvement{State: DecisionClosed, ExactPair: true, Pair: pair, Reason: "EXACT_SAME_SCENARIO_SOURCE_CONTRACT_TOOLCHAIN_RUNNER_PAIR"}
	}
	if decision == DecisionRefuted {
		return Improvement{State: DecisionRefuted, ExactPair: pair.ExactIdentity, Pair: pair, Reason: evidence.Reason}
	}
	unknown := UnknownRecord{Stage: "IMPROVEMENT", Step: "COMPARE_EXACT_BEFORE_AFTER_PAIR", Reason: "IMPROVEMENT_NOT_DISCHARGED_BY_PROBE", UnknownClass: "INCOMPLETE_EVIDENCE", NextOperation: "PROVIDE_ACCEPTED_TYPED_PROBE", BlockedBy: sortedCopy(claim.BlockedBy)}
	return Improvement{State: DecisionUnknown, ExactPair: pair.ExactIdentity, Pair: pair, Reason: unknown.Reason, Unknown: &unknown}
}

func closureBasis(decision string, evidence ProbeEvidence) string {
	switch decision {
	case DecisionClosed:
		return "ACCEPTED_TYPED_PROBE_EVIDENCE"
	case DecisionRefuted:
		return "EXPLICIT_PROBE_CONTRADICTION"
	default:
		return "OPEN_CLAIM_RETAINED_AFTER_INSUFFICIENT_PROBE"
	}
}

func summarize(cases []CaseReport) Summary {
	var summary Summary
	summary.CasesTotal = len(cases)
	summary.TestsTotal = len(cases)
	summary.TestsSelected = len(cases)
	summary.TestsExecuted = len(cases)
	for _, caseReport := range cases {
		summary.UnknownFrontierBefore += caseReport.BeforeUnknownFrontierCardinality
		summary.UnknownFrontierAfter += caseReport.AfterUnknownFrontierCardinality
		summary.ProbeExecutions++
		switch caseReport.Decision {
		case DecisionClosed:
			summary.Closed++
		case DecisionUnknown:
			summary.Unknown++
			summary.TestsUnknown++
		case DecisionRefuted:
			summary.Refuted++
			summary.TestsFailed++
		}
		if caseReport.ProbeIR.Observation == ObservationReplay {
			summary.ReplayComparisons++
			if !caseReport.ReplayEqual {
				summary.ReplayMismatches++
			}
		}
	}
	return summary
}

func finalDecisionAndImprovement(cases []CaseReport, sourceDigest, contractDigest string) (string, Improvement) {
	decision := DecisionClosed
	var chosen *CaseReport
	for index := range cases {
		caseReport := &cases[index]
		if caseReport.Decision == DecisionRefuted {
			decision = DecisionRefuted
			if chosen == nil || chosen.Decision != DecisionRefuted {
				chosen = caseReport
			}
		} else if caseReport.Decision == DecisionUnknown && decision != DecisionRefuted {
			decision = DecisionUnknown
			if chosen == nil {
				chosen = caseReport
			}
		}
	}
	if chosen == nil {
		return DecisionClosed, Improvement{State: DecisionClosed, ExactPair: true, Reason: "ALL_CASES_HAVE_EXACT_ACCEPTED_PROBES", Pair: ExactPair{SourceDigest: sourceDigest, ContractDigest: contractDigest, ToolchainDigest: ToolchainDigest, RunnerDigest: RunnerDigest, ExactIdentity: true}}
	}
	if decision == DecisionRefuted {
		return decision, Improvement{State: DecisionRefuted, ExactPair: chosen.Improvement.ExactPair, Pair: chosen.Improvement.Pair, Reason: "REFUTED_CASE_PRECEDENCE"}
	}
	unknown := UnknownRecord{Stage: "IMPROVEMENT", Step: "COMPARE_EXACT_BEFORE_AFTER_PAIR", Reason: "CASE_SET_RETAINS_UNKNOWN_FRONTIER", UnknownClass: "INCOMPLETE_EVIDENCE", NextOperation: "PROVIDE_ACCEPTED_TYPED_PROBES", BlockedBy: sortedCopy(chosen.AfterBlockedBy)}
	if len(unknown.BlockedBy) == 0 {
		unknown.BlockedBy = sortedCopy(chosen.BeforeBlockedBy)
	}
	return decision, Improvement{State: DecisionUnknown, ExactPair: chosen.Improvement.ExactPair, Pair: chosen.Improvement.Pair, Reason: unknown.Reason, Unknown: &unknown}
}

func validateReport(source Source, report Report) error {
	if report.RepositoryWrites != 0 || report.Authority.RepositoryWrites != 0 || report.Authority.InputRepositoryWrites != 0 || len(report.Cases) != len(source.Scenarios) {
		return fmt.Errorf("report violates zero-write or fixed denominator boundary")
	}
	for _, caseReport := range report.Cases {
		if caseReport.Decision != caseReport.Expected {
			return fmt.Errorf("fixed case %s expected %s but observed %s", caseReport.CaseID, caseReport.Expected, caseReport.Decision)
		}
		if len(caseReport.Claims) != 1 || caseReport.OriginalClaim.State != LifecycleOpen || len(caseReport.ClaimTransitions) != 1 || caseReport.ClaimTransitions[0].From != LifecycleOpen || !caseReport.ClaimTransitions[0].AppendOnly {
			return fmt.Errorf("case %s does not preserve append-only OPEN claim", caseReport.CaseID)
		}
		if caseReport.AfterUnknownFrontierCardinality != len(caseReport.AfterBlockedBy) || caseReport.BeforeUnknownFrontierCardinality != len(caseReport.BeforeBlockedBy) {
			return fmt.Errorf("case %s has inconsistent frontier cardinality", caseReport.CaseID)
		}
		if caseReport.Decision == DecisionUnknown {
			if len(caseReport.AfterBlockedBy) == 0 || !validUnknown(UnknownRecord{Stage: caseReport.OriginalClaim.Stage, Step: caseReport.OriginalClaim.Step, Reason: caseReport.OriginalClaim.Reason, UnknownClass: caseReport.OriginalClaim.UnknownClass, NextOperation: caseReport.OriginalClaim.NextOperation, BlockedBy: caseReport.AfterBlockedBy}) {
				return fmt.Errorf("UNKNOWN case %s does not carry six fields", caseReport.CaseID)
			}
			if caseReport.ClaimTransitions[0].To != LifecycleOpen {
				return fmt.Errorf("UNKNOWN case %s must retain OPEN lifecycle", caseReport.CaseID)
			}
		}
		if caseReport.Decision == DecisionClosed && (caseReport.ClaimTransitions[0].To != LifecycleDischarged || caseReport.ClosureBasis == "") {
			return fmt.Errorf("CLOSED case %s lacks accepted transition evidence", caseReport.CaseID)
		}
		if caseReport.Decision == DecisionRefuted && caseReport.ClaimTransitions[0].To != LifecycleRefuted {
			return fmt.Errorf("REFUTED case %s lacks contradiction transition", caseReport.CaseID)
		}
	}
	return nil
}

func validUnknown(unknown UnknownRecord) bool {
	return unknown.Stage != "" && unknown.Step != "" && unknown.Reason != "" && unknown.UnknownClass != "" && unknown.NextOperation != "" && len(unknown.BlockedBy) > 0
}

func containsScenario(scenarios []Scenario, id string) bool {
	for _, scenario := range scenarios {
		if scenario.ID == id {
			return true
		}
	}
	return false
}

func scenarioOrdinal(scenarios []Scenario, id string) int {
	for _, scenario := range scenarios {
		if scenario.ID == id {
			return scenario.Ordinal
		}
	}
	return 0
}

func scenarioExpected(scenarios []Scenario, id string) string {
	for _, scenario := range scenarios {
		if scenario.ID == id {
			return scenario.Expected
		}
	}
	return ""
}

func findCapability(source Source, id string) (ProbeCapability, bool) {
	for _, capability := range source.Capabilities {
		if capability.ID == id {
			return capability, true
		}
	}
	return ProbeCapability{}, false
}

func findEffect(source Source, id string) (ProbeEffect, bool) {
	for _, effect := range source.Effects {
		if effect.ID == id {
			return effect, true
		}
	}
	return ProbeEffect{}, false
}

func hasDescentRule(source Source, from, to, capability, effect string) bool {
	for _, rule := range source.DescentRules {
		if rule.From == from && rule.To == to && rule.Capability == capability && rule.Effect == effect {
			return true
		}
	}
	return false
}

func zeroAuthority() Authority {
	return Authority{VerificationAuthority: "GITHUB_ACTIONS", RepositoryWrites: 0, InputRepositoryWrites: 0, LocalTestExecutions: 0, LocalBuildExecutions: 0, LocalVetExecutions: 0, LocalConformanceExecutions: 0, LocalIntegrationExecutions: 0, AutomaticCommit: 0, AutomaticPush: 0, AutomaticMerge: 0, AutomaticRelease: 0}
}

func ensureCallerOutput(root, output string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	outputAbs, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	if pathWithin(rootAbs, outputAbs) {
		return fmt.Errorf("output must be caller-owned and outside input repository")
	}
	if err := os.MkdirAll(outputAbs, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(outputAbs)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("caller-owned output must start empty: %s", outputAbs)
	}
	return nil
}

func writeOutputs(output string, ir SemanticIR, report Report) error {
	if err := WriteJSON(filepath.Join(output, "semantic-ir.json"), ir); err != nil {
		return err
	}
	if err := WriteJSON(filepath.Join(output, "report.json"), report); err != nil {
		return err
	}
	if err := WriteJSON(filepath.Join(output, "metrics.json"), report.Metrics); err != nil {
		return err
	}
	if err := WriteSummary(filepath.Join(output, "ci-summary.md"), report); err != nil {
		return err
	}
	receipt := ExecutionReceipt{Schema: ReceiptSchema, AuthorityChain: []string{"GOOO_SOURCE", "SEMANTIC_IR", "TYPED_PROBE_GOOO", "TYPED_PROBE_IR", "TYPED_PROBE_GO", "CALLER_OWNED_FIXTURE", "APPEND_ONLY_CLAIM_TRANSITION"}, CallerOwnedOutput: true, RepositoryWrites: 0, LocalExecutions: map[string]int{"test": 0, "build": 0, "vet": 0, "conformance": 0, "integration": 0}}
	for _, caseReport := range report.Cases {
		caseDir := filepath.Join(output, "generated", caseReport.CaseID)
		if err := os.MkdirAll(caseDir, 0o755); err != nil {
			return err
		}
		goooPath := filepath.Join(caseDir, "probe.gooo")
		irPath := filepath.Join(caseDir, "probe-ir.json")
		goPath := filepath.Join(caseDir, "probe.go")
		if err := WriteText(goooPath, GenerateProbeGooo(caseReport)); err != nil {
			return err
		}
		if err := WriteJSON(irPath, caseReport.ProbeIR); err != nil {
			return err
		}
		if err := WriteText(goPath, GenerateProbeGo(caseReport)); err != nil {
			return err
		}
		receipt.GeneratedArtifacts = append(receipt.GeneratedArtifacts, filepath.ToSlash(filepath.Join("generated", caseReport.CaseID, "probe.gooo")), filepath.ToSlash(filepath.Join("generated", caseReport.CaseID, "probe-ir.json")), filepath.ToSlash(filepath.Join("generated", caseReport.CaseID, "probe.go")))
	}
	return WriteJSON(filepath.Join(output, "execution-receipt.json"), receipt)
}

func WriteJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return WriteText(path, string(raw)+"\n")
}

func WriteText(path, value string) error {
	return os.WriteFile(path, []byte(value), 0o644)
}

func inventory(root string) (Inventory, error) {
	var result Inventory
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if strings.HasPrefix(relative, ".git"+string(filepath.Separator)) || relative == ".git" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			result.DescendantDirs++
			return nil
		}
		if !entry.Type().IsRegular() || filepath.Base(path) == "README.md" && filepath.Dir(path) == root {
			return nil
		}
		result.RegularFiles++
		extension := filepath.Ext(path)
		if extension == ".go" {
			result.GoFiles++
			result.GoPhysicalLines += physicalLines(path)
		}
		if extension == ".gooo" {
			result.GoooFiles++
			result.GoooPhysicalLines += physicalLines(path)
		}
		return nil
	})
	return result, err
}

func physicalLines(path string) int {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines
}

func generatedMetric(output string) GeneratedMetric {
	var result GeneratedMetric
	_ = filepath.WalkDir(filepath.Join(output, "generated"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		info, statErr := entry.Info()
		if statErr == nil {
			result.Files++
			result.Bytes += int(info.Size())
		}
		return nil
	})
	return result
}

func generatedMetricForCases(cases []CaseReport) GeneratedMetric {
	var result GeneratedMetric
	for _, caseReport := range cases {
		result.Files++
		result.Bytes += len(GenerateProbeGo(caseReport))
	}
	return result
}

func processRSSKiB() int {
	if runtime.GOOS != "linux" {
		return 0
	}
	file, err := os.Open("/proc/self/status")
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 3 && fields[0] == "VmHWM:" {
			value, err := strconv.Atoi(fields[1])
			if err == nil {
				return value
			}
		}
	}
	return 0
}
