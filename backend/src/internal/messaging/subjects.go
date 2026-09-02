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

	// Request/Reply lifecycle events published by processor-service so job-service
	// can record them in the activity log and make the flow visible in the UI.
	SubjectJobRequestReceived = "jobs.request.received"
	SubjectJobReplySent       = "jobs.reply.sent"

	// Consumer management and deduplication subjects
	SubjectConsumerConfigSet = "consumer.config.set"
	SubjectJobDeduplicated   = "jobs.deduplicated"
)
