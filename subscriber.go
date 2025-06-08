package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"github.com/yeencloud/lib-events/contract"
	"github.com/yeencloud/lib-events/domain"
	metrics "github.com/yeencloud/lib-metrics"
	logShared "github.com/yeencloud/lib-shared/log"
	"github.com/yeencloud/lib-shared/validation"
)

type Subscriber struct {
	hostName    string
	serviceName string

	client        *redis.Client
	subscriptions map[string]domain.EventHandler
	Validator     *validation.Validator
}

// Subscribe registers a handler for a given stream (channel).
func (s *Subscriber) Subscribe(stream string) *BasicHandler {
	h := BasicHandler{
		channel:   stream,
		handlers:  &map[string]domain.EventHandlerFunc{},
		validator: s.Validator,
	}
	s.subscriptions[stream] = &h
	log.WithField("stream", stream).Info("Registered subscription for stream")
	return &h
}

func (s *Subscriber) subscribe(ctx context.Context) error {
	for stream := range s.subscriptions {
		err := s.client.XGroupCreateMkStream(ctx, stream, s.serviceName, "0").Err()
		if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("XGroupCreateMkStream failed for %s: %w", stream, err)
		}
	}
	return nil
}

func (s *Subscriber) DecodeHeader(msg redis.XMessage) (*contract.Header, error) {
	rawHeader, ok := msg.Values["header"].(string)
	if !ok {
		return nil, errors.New("expecting string header but got unsupported format")
	}

	var header contract.Header
	if err := json.Unmarshal([]byte(rawHeader), &header); err != nil {
		return nil, errors.New("unable to unmarshal raw header")
	}

	if err := s.Validator.Struct(header); err != nil {
		return nil, errors.New("header failed to validate")
	}

	return &header, nil
}

func (s *Subscriber) Listen(ctx context.Context) error {
	err := s.subscribe(ctx)
	if err != nil {
		return err
	}

	for {
		streams := make([]string, 0, len(s.subscriptions)*2)
		for stream := range s.subscriptions {
			streams = append(streams, stream, ">")
		}

		args := &redis.XReadGroupArgs{
			Group:    s.serviceName,
			Consumer: s.hostName,
			Streams:  streams,
		}

		xstreams, err := s.client.XReadGroup(ctx, args).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			return fmt.Errorf("XReadGroup error: %w", err)
		}

		for _, xstream := range xstreams {
			logEntry := log.NewEntry(log.StandardLogger())
			h, ok := s.subscriptions[xstream.Stream]
			if !ok || h == nil {
				continue
			}
			for _, msg := range xstream.Messages {
				logEntry = domain.LogEventsReceivedChannelField.WithValue(xstream.Stream).AsField(logEntry)
				logEntry = domain.LogEventsReceivedMessageIdField.WithValue(msg.ID).AsField(logEntry)
				ctx = logShared.WithLogger(ctx, logEntry)

				log.WithContext(ctx).Tracef("Received event: %v", msg)

				ctx = metrics.SetTag(ctx, "message_id", msg.ID)
				header, err := s.DecodeHeader(msg)
				if err != nil {
					log.WithContext(ctx).WithError(err).Warning("Received event but failed to decode header")
					continue
				}
				logEntry = domain.LogEventsReceivedCorrelationIdField.WithValue(header.CorrelationID).AsField(logEntry)
				logEntry = domain.LogEventsReceivedEventTypeField.WithValue(header.Event).AsField(logEntry)
				ctx = logShared.WithLogger(ctx, logEntry)

				bodyRaw, _ := msg.Values["message"].(string)
				ctrtmsg := contract.Message{
					Header: *header,
					Body:   bodyRaw,
				}

				if err := s.client.XAck(ctx, xstream.Stream, s.serviceName, msg.ID).Err(); err != nil {
					log.Errorf("failed to XAck %s on stream %s: %v", msg.ID, xstream.Stream, err)
				}

				h.MsgReceived(ctx, ctrtmsg)
			}
		}
	}
}

func NewSubscriber(validator *validation.Validator, serviceName, hostName string, rdb *redis.Client) *Subscriber {
	return &Subscriber{
		hostName:      hostName,
		serviceName:   serviceName,
		client:        rdb,
		subscriptions: make(map[string]domain.EventHandler),
		Validator:     validator,
	}
}
