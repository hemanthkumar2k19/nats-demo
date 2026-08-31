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
)
