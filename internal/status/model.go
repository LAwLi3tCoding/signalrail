package status

import "time"

type Confidence string

const (
	ConfidenceExact       Confidence = "exact"
	ConfidenceEstimated   Confidence = "estimated"
	ConfidenceUnavailable Confidence = "unavailable"
)

type Freshness string

const (
	FreshnessFresh    Freshness = "fresh"
	FreshnessCached   Freshness = "cached"
	FreshnessStale    Freshness = "stale"
	FreshnessDegraded Freshness = "degraded"
)

type Source struct {
	Provider string
	Detail   string
}

type Datum[T any] struct {
	Value      T
	Confidence Confidence
	Freshness  Freshness
	Source     Source
	ObservedAt time.Time
	ExpiresAt  time.Time
}

type SegmentName string

const (
	SegmentModel    SegmentName = "model"
	SegmentProject  SegmentName = "project"
	SegmentTask     SegmentName = "task"
	SegmentProgress SegmentName = "progress"
	SegmentContext  SegmentName = "context"
	SegmentCost     SegmentName = "cost"
)

type Snapshot struct {
	Model   Datum[Model]
	Project Datum[Project]
	Task    Datum[Task]
	Context Datum[Context]
	Cost    Datum[Cost]
	Privacy Privacy
}

type Model struct {
	Name   string
	Effort string
}

type Project struct {
	Name   string
	Branch string
	Dirty  bool
}

type Privacy struct {
	RedactUser        bool
	RedactPaths       bool
	SensitiveBranches []string
}

type Task struct {
	Title         string
	Phase         string
	Step          int
	TotalSteps    int
	State         string
	BlockerNote   string
	UpdatedAt     time.Time
	SourceRuntime string
}

type Context struct {
	UsedPercent float64
	LeftLabel   string
	WindowLabel string
}

type Cost struct {
	Amount   float64
	Currency string
}
