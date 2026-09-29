package api

// PublishRequest defines the JSON payload for triggering a NATS publish.
type PublishRequest struct {
	Subject string `json:"subject"`
	Data    string `json:"data"`
}

// PublishResponse defines the JSON response payload returning publish status.
type PublishResponse struct {
	Status  string `json:"status"`
	Subject string `json:"subject"`
}
