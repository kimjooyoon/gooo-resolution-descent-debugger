package debugger

import "encoding/json"

const (
	SourceSchema               = "gooo/resolution_descent_debugger/v1"
	FixtureSchema              = "gooo/resolution-descent-debugger/fixture/v1"
	IRSchema                   = "gooo/resolution-descent-debugger/ir/v1"
	ProbeIRSchema              = "gooo/resolution-descent-debugger/probe-ir/v1"
	ReportSchema               = "gooo/resolution-descent-debugger/report/v1"
	ConformanceSchema          = "gooo/resolution-descent-debugger/conformance/v1"
	ReceiptSchema              = "gooo/resolution-descent-debugger/execution-receipt/v1"
	MetricsSchema              = "gooo/resolution-descent-debugger/metrics/v1"
	DecisionClosed             = "CLOSED"
	DecisionUnknown            = "UNKNOWN"
	DecisionRefuted            = "REFUTED"
	LifecycleOpen              = "OPEN"
	LifecycleDischarged        = "DISCHARGED"
	LifecycleRefuted           = "REFUTED"
	ObservationMatch           = "MATCH"
	ObservationDependencyMatch = "DEPENDENCY_MATCH"
	ObservationStale           = "STALE_SOURCE"
	ObservationAmbiguous       = "AMBIGUOUS"
	ObservationUnbounded       = "UNBOUNDED"
	ObservationForbidden       = "FORBIDDEN_EFFECT"
	ObservationReplay          = "REPLAY_MATCH"
	RunnerDigest               = "sha256:go1.27-gooo-typed-probe-runner-v1"
	ToolchainDigest            = "sha256:go1.27"
	UnknownDirect              = "DIRECT_MISSING"
	UnknownDependency          = "DEPENDENCY_BLOCKED"
	UnknownAmbiguous           = "AMBIGUOUS_EVIDENCE"
	UnknownUnbounded           = "UNBOUNDED_DESCENT"
)

var (
	ResolutionLevels      = []string{"STAGE", "STEP", "REASON", "BLOCKED_BY"}
	RequiredStatuses      = []string{DecisionClosed, DecisionUnknown, DecisionRefuted}
	RequiredPrecedence    = []string{DecisionRefuted, DecisionUnknown, DecisionClosed}
	RequiredUnknownFields = []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"}
)

type Source struct {
	Schema           string
	Authority        string
	Origin           Origin
	ResolutionLevels []string
	Capabilities     []ProbeCapability
	Effects          []ProbeEffect
	DescentRules     []DescentRule
	Transitions      []ClaimTransitionRule
	Statuses         []string
	Precedence       []string
	UnknownFields    []string
	Denominator      DenominatorDecl
	Scenarios        []Scenario
}

type Origin struct {
	Source   string
	Contract string
	Fixture  string
}

type ProbeCapability struct {
	ID          string
	Input       string
	Observation string
}

type ProbeEffect struct {
	ID               string
	Kind             string
	RepositoryWrites int
}

type DescentRule struct {
	Ordinal    int
	From       string
	To         string
	Capability string
	Effect     string
}

type ClaimTransitionRule struct {
	Ordinal  int
	From     string
	To       string
	Evidence string
}

type DenominatorDecl struct {
	ID    string
	Cases int
	Unit  string
}

type Scenario struct {
	Ordinal  int    `json:"ordinal"`
	ID       string `json:"id"`
	Expected string `json:"expected"`
	Fixture  string `json:"fixture"`
}

type Denominator struct {
	Schema                      string                `json:"schema"`
	DenominatorID               string                `json:"denominator_id"`
	ScenarioCount               int                   `json:"scenario_count"`
	RootReadmeInventoryExcluded bool                  `json:"root_readme_inventory_excluded"`
	UnknownFields               []string              `json:"unknown_fields"`
	Precedence                  []string              `json:"precedence"`
	Authority                   DenominatorAuthority  `json:"authority"`
	Scenarios                   []DenominatorScenario `json:"scenarios"`
}

type DenominatorAuthority struct {
	Source                string `json:"source"`
	InputRepositoryWrites int    `json:"input_repository_writes"`
	AutomaticCommit       int    `json:"automatic_commit"`
	AutomaticPush         int    `json:"automatic_push"`
	AutomaticMerge        int    `json:"automatic_merge"`
	AutomaticRelease      int    `json:"automatic_release"`
}

type DenominatorScenario struct {
	Ordinal  int    `json:"ordinal"`
	ID       string `json:"id"`
	Expected string `json:"expected"`
}

type Fixture struct {
	Schema          string `json:"schema"`
	CaseID          string `json:"case_id"`
	Subject         string `json:"subject"`
	StartResolution string `json:"start_resolution"`
	Claim           Claim  `json:"claim"`
	Probe           Probe  `json:"probe"`
}

type Claim struct {
	ID            string   `json:"id"`
	Resolution    string   `json:"resolution"`
	State         string   `json:"state"`
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type Probe struct {
	ID               string   `json:"id"`
	Operation        string   `json:"operation"`
	Capability       string   `json:"capability"`
	Effect           string   `json:"effect"`
	TargetResolution string   `json:"target_resolution"`
	Observation      string   `json:"observation"`
	ObservedIDs      []string `json:"observed_ids"`
	ExpectedIDs      []string `json:"expected_ids"`
	NarrowedFrontier []string `json:"narrowed_frontier,omitempty"`
	Evidence         string   `json:"evidence"`
}

type SemanticIR struct {
	Schema           string                `json:"schema"`
	SourcePath       string                `json:"source_path"`
	SourceDigest     string                `json:"source_digest"`
	ContractPath     string                `json:"contract_path"`
	ContractDigest   string                `json:"contract_digest"`
	ResolutionLevels []string              `json:"resolution_levels"`
	Capabilities     []ProbeCapability     `json:"capabilities"`
	Effects          []ProbeEffect         `json:"effects"`
	DescentRules     []DescentRule         `json:"descent_rules"`
	Transitions      []ClaimTransitionRule `json:"claim_transitions"`
	Scenarios        []Scenario            `json:"scenarios"`
}

type ProbeIR struct {
	Schema           string   `json:"schema"`
	CaseID           string   `json:"case_id"`
	ClaimID          string   `json:"claim_id"`
	Operation        string   `json:"operation"`
	Capability       string   `json:"capability"`
	Effect           string   `json:"effect"`
	FromResolution   string   `json:"from_resolution"`
	ToResolution     string   `json:"to_resolution"`
	Observation      string   `json:"observation"`
	ExpectedIDs      []string `json:"expected_ids"`
	ObservedIDs      []string `json:"observed_ids"`
	NarrowedFrontier []string `json:"narrowed_frontier,omitempty"`
	SourceDigest     string   `json:"source_digest"`
	ContractDigest   string   `json:"contract_digest"`
	FixtureDigest    string   `json:"fixture_digest"`
	RunnerDigest     string   `json:"runner_digest"`
	ToolchainDigest  string   `json:"toolchain_digest"`
}

type ProbeEvidence struct {
	ID               string   `json:"id"`
	Observation      string   `json:"observation"`
	Accepted         bool     `json:"accepted"`
	Contradiction    bool     `json:"contradiction"`
	Capability       string   `json:"capability"`
	Effect           string   `json:"effect"`
	ObservedIDs      []string `json:"observed_ids"`
	ExpectedIDs      []string `json:"expected_ids"`
	NarrowedFrontier []string `json:"narrowed_frontier,omitempty"`
	Reason           string   `json:"reason"`
	EvidenceDigest   string   `json:"evidence_digest"`
	InputDigest      string   `json:"input_digest"`
	RunnerDigest     string   `json:"runner_digest"`
	ToolchainDigest  string   `json:"toolchain_digest"`
}

type ClaimTransition struct {
	Sequence      int      `json:"sequence"`
	ClaimID       string   `json:"claim_id"`
	From          string   `json:"from"`
	To            string   `json:"to"`
	EvidenceID    string   `json:"evidence_id"`
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
	AppendOnly    bool     `json:"append_only"`
}

type ExactPair struct {
	ScenarioID      string `json:"scenario_id"`
	SourceDigest    string `json:"source_digest"`
	ContractDigest  string `json:"contract_digest"`
	ToolchainDigest string `json:"toolchain_digest"`
	RunnerDigest    string `json:"runner_digest"`
	ExactIdentity   bool   `json:"exact_identity"`
	Before          int    `json:"before"`
	After           int    `json:"after"`
}

type Improvement struct {
	State     string         `json:"state"`
	ExactPair bool           `json:"exact_pair"`
	Pair      ExactPair      `json:"pair"`
	Reason    string         `json:"reason"`
	Unknown   *UnknownRecord `json:"unknown,omitempty"`
}

type UnknownRecord struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type CaseReport struct {
	Ordinal                          int               `json:"ordinal"`
	CaseID                           string            `json:"case_id"`
	Expected                         string            `json:"expected"`
	Subject                          string            `json:"subject"`
	StartResolution                  string            `json:"start_resolution"`
	Decision                         string            `json:"decision"`
	ClosureBasis                     string            `json:"closure_basis"`
	OriginalClaim                    Claim             `json:"original_claim"`
	Claims                           []Claim           `json:"claims"`
	ProbeIR                          ProbeIR           `json:"probe_ir"`
	ProbeEvidence                    ProbeEvidence     `json:"probe_evidence"`
	ClaimTransitions                 []ClaimTransition `json:"claim_transitions"`
	BeforeUnknownFrontierCardinality int               `json:"before_unknown_frontier_cardinality"`
	AfterUnknownFrontierCardinality  int               `json:"after_unknown_frontier_cardinality"`
	BeforeBlockedBy                  []string          `json:"before_blocked_by"`
	AfterBlockedBy                   []string          `json:"after_blocked_by"`
	NarrowedFrontier                 []string          `json:"narrowed_frontier"`
	ReplayEqual                      bool              `json:"replay_equal"`
	ReplayDigest                     string            `json:"replay_digest,omitempty"`
	Improvement                      Improvement       `json:"improvement"`
}

type Summary struct {
	CasesTotal            int `json:"cases_total"`
	Closed                int `json:"closed"`
	Unknown               int `json:"unknown"`
	Refuted               int `json:"refuted"`
	UnknownFrontierBefore int `json:"unknown_frontier_before"`
	UnknownFrontierAfter  int `json:"unknown_frontier_after"`
	ProbeExecutions       int `json:"probe_executions"`
	ReplayComparisons     int `json:"replay_comparisons"`
	ReplayMismatches      int `json:"replay_mismatches"`
	TestsTotal            int `json:"tests_total"`
	TestsSelected         int `json:"tests_selected"`
	TestsExecuted         int `json:"tests_executed"`
	TestsReused           int `json:"tests_reused"`
	TestsFailed           int `json:"tests_failed"`
	TestsUnknown          int `json:"tests_unknown"`
}

type Authority struct {
	VerificationAuthority      string `json:"verification_authority"`
	RepositoryWrites           int    `json:"repository_writes"`
	InputRepositoryWrites      int    `json:"input_repository_writes"`
	LocalTestExecutions        int    `json:"local_test_executions"`
	LocalBuildExecutions       int    `json:"local_build_executions"`
	LocalVetExecutions         int    `json:"local_vet_executions"`
	LocalConformanceExecutions int    `json:"local_conformance_executions"`
	LocalIntegrationExecutions int    `json:"local_integration_executions"`
	AutomaticCommit            int    `json:"automatic_commit"`
	AutomaticPush              int    `json:"automatic_push"`
	AutomaticMerge             int    `json:"automatic_merge"`
	AutomaticRelease           int    `json:"automatic_release"`
}

type Inventory struct {
	GoFiles           int `json:"go_files"`
	GoooFiles         int `json:"gooo_files"`
	GoPhysicalLines   int `json:"go_physical_lines"`
	GoooPhysicalLines int `json:"gooo_physical_lines"`
	DescendantDirs    int `json:"descendant_dirs"`
	RegularFiles      int `json:"regular_files"`
}

type StageMetric struct {
	WallMS     int `json:"wall_ms"`
	PeakRSSKiB int `json:"peak_rss_kib"`
}

type Metrics struct {
	Schema           string          `json:"schema"`
	Inventory        Inventory       `json:"inventory"`
	Generated        GeneratedMetric `json:"generated"`
	Stages           StageMetrics    `json:"stages"`
	Tests            TestMetrics     `json:"tests"`
	RepositoryWrites int             `json:"repository_writes"`
	Authority        Authority       `json:"authority"`
}

type GeneratedMetric struct {
	Files int `json:"files"`
	Bytes int `json:"bytes"`
}

type StageMetrics struct {
	Compile     StageMetric `json:"compile"`
	Build       StageMetric `json:"build"`
	Test        StageMetric `json:"test"`
	Conformance StageMetric `json:"conformance"`
	Integration StageMetric `json:"integration"`
}

type TestMetrics struct {
	Total    int `json:"total"`
	Selected int `json:"selected"`
	Executed int `json:"executed"`
	Reused   int `json:"reused"`
	Failed   int `json:"failed"`
	Unknown  int `json:"unknown"`
}

type Report struct {
	Schema           string       `json:"schema"`
	Decision         string       `json:"decision"`
	SourcePath       string       `json:"source_path"`
	SourceDigest     string       `json:"source_digest"`
	ContractPath     string       `json:"contract_path"`
	ContractDigest   string       `json:"contract_digest"`
	ToolchainDigest  string       `json:"toolchain_digest"`
	RunnerDigest     string       `json:"runner_digest"`
	ResolutionLevels []string     `json:"resolution_levels"`
	Precedence       []string     `json:"precedence"`
	UnknownFields    []string     `json:"unknown_fields"`
	Cases            []CaseReport `json:"cases"`
	Summary          Summary      `json:"summary"`
	Authority        Authority    `json:"authority"`
	Improvement      Improvement  `json:"improvement"`
	Inventory        Inventory    `json:"inventory"`
	Metrics          Metrics      `json:"metrics"`
	RepositoryWrites int          `json:"repository_writes"`
}

type ExecutionReceipt struct {
	Schema             string         `json:"schema"`
	AuthorityChain     []string       `json:"authority_chain"`
	GeneratedArtifacts []string       `json:"generated_artifacts"`
	CallerOwnedOutput  bool           `json:"caller_owned_output"`
	RepositoryWrites   int            `json:"repository_writes"`
	LocalExecutions    map[string]int `json:"local_executions"`
}

type ConformanceReport struct {
	Schema           string       `json:"schema"`
	Decision         string       `json:"decision"`
	FixedDenominator int          `json:"fixed_denominator"`
	Summary          Summary      `json:"summary"`
	Cases            []CaseReport `json:"cases"`
	Authority        Authority    `json:"authority"`
	RepositoryWrites int          `json:"repository_writes"`
}

func (r Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
