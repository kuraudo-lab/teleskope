package advisor

import (
	"github.com/kuraudo-lab/teleskope/internal/inventory"
	"time"
)

type Evidence struct {
	Resource    inventory.ObjectRef `json:"resource"`
	Field       string              `json:"field"`
	Value       string              `json:"value"`
	CollectedAt time.Time           `json:"collectedAt"`
}

type Capability struct {
	Key            string                   `json:"key"`
	Scope          string                   `json:"scope"`
	Implementation string                   `json:"implementation,omitempty"`
	Assessment     string                   `json:"assessment"`
	Basis          string                   `json:"basis"`
	Summary        string                   `json:"summary"`
	Constraints    []string                 `json:"constraints,omitempty"`
	Evidence       []Evidence               `json:"evidence,omitempty"`
	Collection     []inventory.CoverageItem `json:"collection,omitempty"`
	Coverage       string                   `json:"coverage"`
	Freshness      string                   `json:"freshness"`
	RuleID         string                   `json:"ruleId"`
}

type Report struct {
	SchemaVersion string       `json:"schemaVersion"`
	RuleVersion   string       `json:"ruleVersion"`
	Summary       string       `json:"summary"`
	Capabilities  []Capability `json:"capabilities"`
}
