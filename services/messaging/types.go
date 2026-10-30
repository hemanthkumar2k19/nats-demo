package messaging

import (
	"time"

	"services/model"
)

// RangeParams defines the boundaries for stream message replay or reprocessing.
type RangeParams struct {
	Type          RangeType
	StartSequence uint64
	EndSequence   uint64
	StartTime     time.Time
	EndTime       *time.Time
}

// ConsumerEffect defines whether the consumer execution is audit-only or applies business side-effects.
type ConsumerEffect string

const (
	EffectAudit    ConsumerEffect = "none"
	EffectBusiness ConsumerEffect = "business-effect"
)

// ReplaySummary holds execution boundaries and statistics for replayed or reprocessed messages.
type ReplaySummary struct {
	FirstSequence  uint64
	LastSequence   uint64
	FirstTimestamp *model.FormattedTime
	LastTimestamp  *model.FormattedTime
	Count          int
}

