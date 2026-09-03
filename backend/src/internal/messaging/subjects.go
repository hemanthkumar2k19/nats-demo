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
	SubjectConsumerReset     = "consumer.reset"
	SubjectJobDeduplicated   = "jobs.deduplicated"

	// JetStream replay subject
	SubjectJobReplayed = "jobs.replayed"

	// JetStream Dead Letter Queue subjects
	SubjectJobDLQ            = "jobs.dlq"
	SubjectJobDLQPublished   = "jobs.dlq.published"
	SubjectJobReprocessed    = "jobs.reprocessed"
	SubjectJobDLQReprocessed = "jobs.dlq.reprocessed"

	// Core NATS Queue Group constants
	SubjectJobQueue            = "jobs.queue"
	SubjectJobQueueReceived    = "jobs.queue.received"
	SubjectJobQueueCompleted   = "jobs.queue.completed"
	QueueGroupJobWorkers       = "job-workers"
	SubjectQueueGroupConfigSet = "queuegroup.config.set"
	SubjectQueueGroupStatus    = "queuegroup.status"
	SubjectQueueGroupReset     = "queuegroup.reset"

	// Delayed and Retry Delivery subjects
	SubjectJobScheduled  = "jobs.scheduled"
	SubjectJobNakDelayed = "jobs.nak.delayed"
	SubjectJobAckTimeout = "jobs.ack.timeout"
)
