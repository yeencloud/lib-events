package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/yeencloud/lib-events/contract"
	"github.com/yeencloud/lib-events/domain"
	metrics "github.com/yeencloud/lib-metrics"
	MetricsDomain "github.com/yeencloud/lib-metrics/domain"
	sharedMetrics "github.com/yeencloud/lib-shared/metrics"
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

	tags := ctx.Value(sharedMetrics.MetricsPointKey).(MetricsDomain.Point).Tags

	messageToSend := contract.Message{
		Header: contract.Header{
			Date:          time.Now().String(),
			Event:         event,
			CorrelationID: tags[sharedMetrics.CorrelationIdKey.MetricKey()],
		},
		Body: message,
	}

	j, _ := json.Marshal(messageToSend)

	err := p.client.Publish(ctx, channel, j).Err()
	if err != nil {
		return err
	}

	j, _ = json.Marshal(messageToSend.Body)
	return metrics.WritePoint(ctx, domain.PublishedEventsMetricPointName, MessagePublishedMetric{
		Channel: channel,
		Event:   event,
		Payload: string(j),
	})
}

func NewPublisher(client *redis.Client) *Publisher {
	return &Publisher{
		client: client,
	}
}
