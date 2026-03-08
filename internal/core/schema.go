package core

// ContextFile represents a parsed AGENTS.yaml file.
type ContextFile struct {
	Context   []ContextEntry  `yaml:"context"`
	Decisions []DecisionEntry `yaml:"decisions"`

	// sourceDir is the directory containing this file, used for resolving globs.
	sourceDir string
}

// ContextEntry is an atomic piece of guidance for an AI agent.
type ContextEntry struct {
	Content string   `yaml:"content"`
	Match   []string `yaml:"match"`
	Exclude []string `yaml:"exclude"`
	On      FlexList `yaml:"on"`
	When    string   `yaml:"when"`
}

// DecisionEntry captures an architectural decision and its rationale.
type DecisionEntry struct {
	Decision     string        `yaml:"decision"`
	Rationale    string        `yaml:"rationale"`
	Alternatives []Alternative `yaml:"alternatives"`
	RevisitWhen  string        `yaml:"revisit_when"`
	Date         string        `yaml:"date"`
	Match        []string      `yaml:"match"`
}

// Alternative records an option that was considered and why it was rejected.
type Alternative struct {
	Option         string `yaml:"option"`
	ReasonRejected string `yaml:"reason_rejected"`
}

// FlexList handles YAML fields that can be either a single string or a list of strings.
type FlexList []string

// UnmarshalYAML lets FlexList accept both `on: edit` and `on: [edit, create]`.
func (f *FlexList) UnmarshalYAML(unmarshal func(any) error) error {
	var single string
	if err := unmarshal(&single); err == nil {
		*f = []string{single}
		return nil
	}

	var list []string
	if err := unmarshal(&list); err != nil {
		return err
	}

	*f = list

	return nil
}

// Action represents the type of file operation.
type Action string

// Supported action values.
const (
	ActionRead   Action = "read"
	ActionEdit   Action = "edit"
	ActionCreate Action = "create"
	ActionAll    Action = "all"
)

// Timing represents when context should be delivered relative to file content.
type Timing string

// Supported timing values.
const (
	TimingBefore Timing = "before"
	TimingAfter  Timing = "after"
)

// ValidAction reports whether s is a recognised action value.
func ValidAction(s string) bool {
	switch Action(s) {
	case ActionRead, ActionEdit, ActionCreate, ActionAll:
		return true
	}
	return false
}

// ValidTiming reports whether s is a recognised timing value.
func ValidTiming(s string) bool {
	switch Timing(s) {
	case TimingBefore, TimingAfter:
		return true
	}
	return false
}

// ResolveRequest contains the universal inputs for context resolution.
// This is the agent-agnostic interface between adapters and the core engine.
type ResolveRequest struct {
	FilePath string
	Action   Action
	Timing   Timing
}

// ResolveResult contains the matched context and decisions for a request.
type ResolveResult struct {
	ContextEntries  []MatchedContext
	DecisionEntries []DecisionEntry
}

// MatchedContext pairs a context entry with the directory it came from.
type MatchedContext struct {
	Content   string
	SourceDir string
}
