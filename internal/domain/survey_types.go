package domain

import "time"

type ConditionLevel string

const (
	ConditionGood      ConditionLevel = "good"
	ConditionAttention ConditionLevel = "attention"
	ConditionPoor      ConditionLevel = "poor"
	ConditionCritical  ConditionLevel = "critical"
)

func (c ConditionLevel) Valid() bool {
	return c == ConditionGood || c == ConditionAttention || c == ConditionPoor || c == ConditionCritical
}

type Observation struct {
	Condition ConditionLevel `json:"condition"`
	Notes     string         `json:"notes"`
}

type EnvironmentObservation struct {
	Notes              string `json:"notes"`
	SoilCompaction     bool   `json:"soil_compaction"`
	NearbyConstruction bool   `json:"nearby_construction"`
	HeavyTraffic       bool   `json:"heavy_traffic"`
	DrainageProblem    bool   `json:"drainage_problem"`
}

type PhotoRef struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Part    string `json:"part,omitempty"`
	TakenAt string `json:"taken_at,omitempty"`
	Caption string `json:"caption,omitempty"`
}

type ConditionSurvey struct {
	ID          string                 `json:"id"`
	CaseID      string                 `json:"case_id"`
	Crown       Observation            `json:"crown_condition"`
	Trunk       Observation            `json:"trunk_condition"`
	RootZone    Observation            `json:"root_zone_condition"`
	Environment EnvironmentObservation `json:"environment"`
	ObservedAt  time.Time              `json:"observed_at"`
	Observer    string                 `json:"observer_name"`
	PhotoRefs   []PhotoRef             `json:"photo_refs"`
	SubmittedAt time.Time              `json:"submitted_at"`
	Expired     bool                   `json:"expired"`
}
