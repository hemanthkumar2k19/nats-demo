package messaging

import (
	"time"

	"github.com/nats-io/nats.go"
)

func Publish(nc *nats.Conn, subject string, data []byte) error {
	return nc.Publish(subject, data)
}

func PublishRequest(nc *nats.Conn, subject string, reply string, data []byte) error {
	return nc.PublishRequest(subject, reply, data)
}

func PublishMsg(nc *nats.Conn, msg *nats.Msg) error {
	return nc.PublishMsg(msg)
}

func Request(nc *nats.Conn, subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	return nc.Request(subject, data, timeout)
}

func RequestMsg(nc *nats.Conn, msg *nats.Msg, timeout time.Duration) (*nats.Msg, error) {
	return nc.RequestMsg(msg, timeout)
}

func FlushTimeout(nc *nats.Conn, timeout time.Duration) error {
	return nc.FlushTimeout(timeout)
}

func Subscribe(nc *nats.Conn, subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	return nc.Subscribe(subject, handler)
}

func SubscribeSync(nc *nats.Conn, subject string) (*nats.Subscription, error) {
	return nc.SubscribeSync(subject)
}

func NextMsg(sub *nats.Subscription, timeout time.Duration) (*nats.Msg, error) {
	return sub.NextMsg(timeout)
}

func ChanSubscribe(nc *nats.Conn, subject string, ch chan *nats.Msg) (*nats.Subscription, error) {
	return nc.ChanSubscribe(subject, ch)
}

func QueueSubscribe(nc *nats.Conn, subject string, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	return nc.QueueSubscribe(subject, queue, handler)
}

func AutoUnsubscribe(sub *nats.Subscription, max int) error {
	return sub.AutoUnsubscribe(max)
}

func SetPendingLimits(sub *nats.Subscription, msgLimit int, bytesLimit int) error {
	return sub.SetPendingLimits(msgLimit, bytesLimit)
}

func DrainSubscription(sub *nats.Subscription) error {
	return sub.Drain()
}

func Unsubscribe(sub *nats.Subscription) error {
	return sub.Unsubscribe()
}

func Respond(msg *nats.Msg, data []byte) error {
	return msg.Respond(data)
}

func NewRespInbox(nc *nats.Conn) string {
	return nc.NewRespInbox()
}
