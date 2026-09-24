package messaging

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

func Publish(nc *nats.Conn, sub string, msg []byte) error {
	return nc.Publish(sub, msg)
}

func Subscribe(nc *nats.Conn, sub string) (*nats.Subscription, error) {
	return nc.Subscribe(sub, func(msg *nats.Msg) {
		fmt.Printf("Received message on subject '%s': %s\n", msg.Subject, string(msg.Data))
	})
}

func Request(nc *nats.Conn, subject string, msg []byte, timeout time.Duration) (*nats.Msg, error) {
	return nc.Request(subject, msg, timeout)
}
