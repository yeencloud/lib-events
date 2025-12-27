package events

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"github.com/yeencloud/lib-events/contract"
	"github.com/yeencloud/lib-events/domain"
	lib_shared "github.com/yeencloud/lib-shared/domain"
)

type Publisher struct {
	client *redis.Client
}

type MessagePublishedMetric struct {
	Channel string `metric:"channel"`
	Event   string `metric:"event"`
	Payload string `metric:"payload"`
}

func (p Publisher) Publish(ctx context.Context, message domain.PublishableMessage) error {
	event := message.EventType()
	channel := message.Channel()

	err := p.client.XGroupCreateMkStream(ctx, channel, "*", "0").Err()
	if err != nil {
		return err
	}

	id, err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: lib_shared.AppName,
		Values: map[string]interface{}{
			"header": contract.Header{
				Date:  time.Now().String(),
				Event: event,
				//TODO: CorrelationID: point.Tags[sharedMetrics.CorrelationIdKey.MetricKey()],
			},
			"message": message,
		},
	}).Result()

	log.WithContext(ctx).Info("Published message with id: ", id)
	if err != nil {
		return err
	}

	return nil
}

func NewPublisher(client *redis.Client) *Publisher {
	return &Publisher{
		client: client,
	}
}
