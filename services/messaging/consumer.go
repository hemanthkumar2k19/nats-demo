package messaging

import (
	"context"

	"services/model"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// Get or Create
func CreateConsumer(ctx context.Context, js jetstream.JetStream, cr *model.CreateConsumerRequest) (jetstream.Consumer, error) {
	consumer, err := js.Consumer(ctx, cr.StreamName, cr.ConsumerName)

	if err == nil {

		info, _ := consumer.Info(ctx)
		log.Info().Msg("Consumer Already Exists: " + info.Name)
		return consumer, nil
	}

	if err == jetstream.ErrConsumerNotFound {

		log.Info().Msg("Creating new consumer: " + cr.ConsumerName)

		cc := jetstream.ConsumerConfig{
			Durable:        cr.ConsumerName,
			FilterSubjects: cr.FilterSubjects,
			AckPolicy:      ResolveAckPolicy(cr.AckPolicy),
			DeliverPolicy:  ResolveDeliverPolicy(cr.DeliverPolicy),
		}

		if cr.DeliverSubject != "" {
			// push consumer
			cc.DeliverSubject = cr.DeliverSubject
		}

		consumer, err = js.CreateConsumer(ctx, cr.StreamName, cc)
		if err != nil {
			return nil, err
		}

		info, _ := consumer.Info(ctx)
		log.Info().Msg("Consumer Created: " + info.Name)
		return consumer, nil
	}

	return nil, err
}

func DeleteConsumer(ctx context.Context, js jetstream.JetStream, cr *model.CreateConsumerRequest) error {
	err := js.DeleteConsumer(ctx, cr.StreamName, cr.ConsumerName)
	if err == nil {
		log.Info().Msg("Consumer deleted: " + cr.ConsumerName)
	}

	if err == jetstream.ErrConsumerNotFound {
		log.Info().Msg("Consumer not found: " + cr.ConsumerName)
		return nil
	}

	return err
}
