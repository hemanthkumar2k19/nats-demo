package messaging

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func PublishJS(ctx context.Context, js jetstream.JetStream, subject string, data []byte) (*jetstream.PubAck, error) {
	return js.Publish(ctx, subject, data)
}

func PublishMsgJS(ctx context.Context, js jetstream.JetStream, msg *nats.Msg) (*jetstream.PubAck, error) {
	ack, err := js.PublishMsg(ctx, msg)
	return ack, err
}

func PublishAsyncJS(js jetstream.JetStream, subject string, data []byte) (jetstream.PubAckFuture, error) {
	return js.PublishAsync(subject, data)
}

func PublishAsyncCompleteJS(js jetstream.JetStream) <-chan struct{} {
	return js.PublishAsyncComplete()
}

func CreateOrUpdateConsumer(ctx context.Context, js jetstream.JetStream, stream string, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	return js.CreateOrUpdateConsumer(ctx, stream, cfg)
}

func OrderedConsumer(ctx context.Context, js jetstream.JetStream, stream string, cfg jetstream.OrderedConsumerConfig) (jetstream.Consumer, error) {
	return js.OrderedConsumer(ctx, stream, cfg)
}

func ConsumerInfo(ctx context.Context, cons jetstream.Consumer) (*jetstream.ConsumerInfo, error) {
	return cons.Info(ctx)
}

func PauseConsumer(ctx context.Context, js jetstream.JetStream, stream string, consumer string, until time.Time) (*jetstream.ConsumerPauseResponse, error) {
	return js.PauseConsumer(ctx, stream, consumer, until)
}

func ResumeConsumer(ctx context.Context, js jetstream.JetStream, stream string, consumer string) (*jetstream.ConsumerPauseResponse, error) {
	return js.ResumeConsumer(ctx, stream, consumer)
}

func DeleteConsumer(ctx context.Context, js jetstream.JetStream, stream string, consumer string) error {
	return js.DeleteConsumer(ctx, stream, consumer)
}

func Stream(ctx context.Context, js jetstream.JetStream, name string) (jetstream.Stream, error) {
	return js.Stream(ctx, name)
}

func GetMsg(ctx context.Context, stream jetstream.Stream, seq uint64) (*jetstream.RawStreamMsg, error) {
	return stream.GetMsg(ctx, seq)
}

func GetLastMsgForSubject(ctx context.Context, stream jetstream.Stream, subject string) (*jetstream.RawStreamMsg, error) {
	return stream.GetLastMsgForSubject(ctx, subject)
}

func Consume(cons jetstream.Consumer, handler jetstream.MessageHandler) (jetstream.ConsumeContext, error) {
	return cons.Consume(handler)
}

func Fetch(cons jetstream.Consumer, batch int, opts ...jetstream.FetchOpt) (jetstream.MessageBatch, error) {
	return cons.Fetch(batch, opts...)
}

func FetchNoWait(cons jetstream.Consumer, batch int) (jetstream.MessageBatch, error) {
	return cons.FetchNoWait(batch)
}

func NextMsgJS(cons jetstream.Consumer) (jetstream.Msg, error) {
	return cons.Next()
}

func AckMsg(msg jetstream.Msg) error {
	return msg.Ack()
}

func NakMsg(msg jetstream.Msg) error {
	return msg.Nak()
}

func NakMsgWithDelay(msg jetstream.Msg, delay time.Duration) error {
	return msg.NakWithDelay(delay)
}

func InProgressMsg(msg jetstream.Msg) error {
	return msg.InProgress()
}

func TermMsg(msg jetstream.Msg) error {
	return msg.Term()
}
