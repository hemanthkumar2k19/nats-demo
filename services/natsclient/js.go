package natsclient

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func NewJSClient(nc *nats.Conn) (jetstream.JetStream, error) {

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to create jetstream client: %w", err)
	}

	return js, nil
}
