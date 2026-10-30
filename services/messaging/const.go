package messaging

const (
	defaultReplayBatchSize = 50
	maxReplayBatchSize     = 1024
)

// RangeType specifies whether stream iteration is bounded by sequence numbers or timestamps.
type RangeType string

const (
	RangeBySequence RangeType = "sequence"
	RangeByTime     RangeType = "time"
)

