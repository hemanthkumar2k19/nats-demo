package messaging

const (
	SubjectJobValidate            = "jobs.validate"
	SubjectJobSubmitted           = "jobs.submitted"
	SubjectJobProcessing          = "jobs.processing"
	SubjectJobProcessingStarted   = "jobs.processing.started"
	SubjectJobProcessingCompleted = "jobs.processing.completed"
	SubjectJobProcessingFailed    = "jobs.processing.failed"
	SubjectJobCompleted           = "jobs.completed"
	SubjectJobFailed              = "jobs.failed"
	SubjectJobStored              = "jobs.stored"
	SubjectJobDelivered           = "jobs.delivered"
	SubjectJobAcked               = "jobs.acked"
	SubjectJobReceived            = "jobs.received"
	SubjectJobNoConsumer          = "jobs.noconsumer"
	SubjectProcessorStateSet      = "processor.state.set"
	SubjectProcessorState         = SubjectProcessorStateSet

	// Request/Reply lifecycle events published so observers can record them in the activity log
	SubjectJobRequestSent     = "jobs.request.sent"
	SubjectJobRequestReceived = "jobs.request.received"
	SubjectJobReplySent       = "jobs.reply.sent"
	SubjectJobReplyReceived   = "jobs.reply.received"
	SubjectJobRequestTimeout  = "jobs.request.timeout"

	// Consumer management and deduplication subjects
	SubjectConsumerConfigSet = "consumer.config.set"
	SubjectJobDeduplicated   = "jobs.deduplicated"

	// JetStream replay subject
	SubjectJobReplayed = "jobs.replayed"
)

