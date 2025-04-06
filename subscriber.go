package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"github.com/yeencloud/lib-events/contract"
	"github.com/yeencloud/lib-events/domain"
)

type Subscriber struct {
	client *redis.Client

	subscriptions map[string]domain.EventHandler
}

func (s *Subscriber) listSubscriptions() []string {
	channels := make([]string, 0, len(s.subscriptions))

	for channel, _ := range s.subscriptions {
		channels = append(channels, channel)
	}

	return channels
}

// TODO: Get pending messages from redis
// TODO: Multiple pods will treat messages in parallel, we should write an overlay that sends and receives acks so that doesn't happen
func (s *Subscriber) subscribe(ctx context.Context) (*redis.PubSub, error) {
	subscriptions := s.listSubscriptions()

	log.Info("Subscribing to channels: ", subscriptions)
	sub := s.client.Subscribe(ctx, subscriptions...)

	subscriptionResponse, err := sub.Receive(ctx)
	if err != nil {
		return nil, err
	}

	switch msg := subscriptionResponse.(type) {
	case *redis.Subscription:
		log.WithFields(log.Fields{
			domain.LogEventsSubscriptionChannelField.MetricKey(): msg.Channel,
			domain.LogEventsSubscriptionCountField.MetricKey():   msg.Count,
			domain.LogEventsSubscriptionKindField.MetricKey():    msg.Kind,
		}).Info("Subscribed to channel")
	default:
		return nil, fmt.Errorf("unexpected redis response: %T", msg) // TODO: error in domain
	}

	return sub, nil
}

func (s *Subscriber) Subscribe(channel string) *BasicHandler {
	receiver := BasicHandler{
		channel:  channel,
		handlers: &map[string]domain.EventHandlerFunc{},
	}
	s.subscriptions[channel] = &receiver
	log.WithField("channel", channel).Info("Subscribing to channel")
	return &receiver
}

func (s *Subscriber) Listen(ctx context.Context) error {
	sub, err := s.subscribe(ctx)
	if err != nil {
		return err
	}

	log.Info("Listening for messages")
	ch := sub.Channel()

	for msg := range ch {
		channel, exists := s.subscriptions[msg.Channel]

		if exists {
			message := contract.Message{}

			err := json.Unmarshal([]byte(msg.Payload), &message)

			if err != nil {
				return err
			}

			channel.MsgReceived(message)
		}
	}

	return nil
}
func NewSubscriber(rdb *redis.Client) *Subscriber {
	return &Subscriber{
		client: rdb,

		subscriptions: map[string]domain.EventHandler{},
	}
}
