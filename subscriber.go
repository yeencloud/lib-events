package events

import (
	"context"
	"encoding/json"
	"errors"
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
		handlers:  map[string]domain.EventHandlerFunc{},
		validator: s.Validator,
	}
	s.subscriptions[stream] = &h
	log.WithField("stream", stream).Info("Registered subscription for stream")
	return &h
}

func (s *Subscriber) subscribe(ctx context.Context) error {
	for stream := range s.subscriptions {
		err := s.client.XGroupCreateMkStream(ctx, stream, s.serviceName, "0").Err()
		// if redis returns a busygroup it means the stream already exists
		if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
			return errors.Join(domain.FailedToCreateStreamError{Stream: stream}, err)
		}
	}
	return nil
}

func (s *Subscriber) DecodeHeader(msg redis.XMessage) (*contract.Header, error) {
	rawHeader, ok := msg.Values["header"].(string)
	if !ok {
		return nil, domain.ErrEventHeaderShouldBeAString
	}

	var header contract.Header
	if err := json.Unmarshal([]byte(rawHeader), &header); err != nil {
		return nil, errors.Join(domain.ErrUnableToUnmarshalEventHeader, err)
	}

	if err := s.Validator.Struct(header); err != nil {
		return nil, errors.Join(domain.ErrFailedToValidateEventHeader, err)
	}

	return &header, nil
}

func (s *Subscriber) Listen(ctx context.Context) error {
	err := s.subscribe(ctx)
	if err != nil {
		return err
	}

	if len(s.subscriptions) == 0 {
		return nil
	}

	s.processMessageStuckInQueue(ctx)

	streams := make([]string, 0, len(s.subscriptions)*2)
	for stream := range s.subscriptions {
		streams = append(streams, stream, ">")
	}
	for {
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
			return errors.Join(domain.ErrUnableToReadFromGroup, err)
		}

		for _, xstream := range xstreams {
			// Parsing through received messages
			for _, msg := range xstream.Messages {
				s.processReceivedMessage(xstream.Stream, msg) //nolint: contextcheck
			}
		}
	}
}

func (s *Subscriber) processMessageStuckInQueue(ctx context.Context) {
	// Checking if any message is pending for a long time and reprocess them in priority before taking on the queue
	for stream := range s.subscriptions {
		pending := s.client.XPending(ctx, stream, s.serviceName)
		count := pending.Val().Count
		log.WithContext(ctx).Warnf("%d events are found pending", count)
		args := &redis.XPendingExtArgs{
			Stream: stream,
			Group:  s.serviceName,
			Start:  "-", // lowest ID
			End:    "+", // highest ID
			Count:  count,
		}
		res, err := s.client.XPendingExt(ctx, args).Result()
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Unable to list pending messages")
		}

		for _, pendingMessage := range res {
			claimResponse := s.client.XClaim(ctx, &redis.XClaimArgs{
				Stream:   stream,
				Group:    s.serviceName,
				Consumer: s.hostName,
				MinIdle:  0,
				Messages: []string{pendingMessage.ID},
			})

			if err = claimResponse.Err(); err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Unable to claim id: %s", pendingMessage.ID)
				continue
			}

			claimedMessage := claimResponse.Val()
			if len(claimedMessage) != 1 {
				log.WithContext(ctx).WithError(err).Errorf("received the wrong number of messages, got %d, expected 2", len(claimedMessage))
			}

			msg := claimResponse.Val()[0]
			s.processReceivedMessage(stream, msg) //nolint: contextcheck
		}
	}
}

func (s *Subscriber) processReceivedMessage(stream string, msg redis.XMessage) {
	// Building context and logger
	msgCtx := context.Background()

	h, ok := s.subscriptions[stream]
	if !ok || h == nil {
		s.Ack(msgCtx, stream, msg) // ACK the message cause this service doesn't handle this event
		return
	}

	logEntry := log.NewEntry(log.StandardLogger())

	logEntry = domain.LogEventsReceivedChannelField.WithValue(stream).AsField(logEntry)
	logEntry = domain.LogEventsReceivedMessageIdField.WithValue(msg.ID).AsField(logEntry)
	msgCtx = logShared.WithLogger(msgCtx, logEntry)
	msgCtx = metrics.SetTag(msgCtx, domain.MessageIdKey.MetricKey(), msg.ID)
	log.WithContext(msgCtx).Debugf("Received event: %v", msg)

	// Header
	header, err := s.DecodeHeader(msg)
	if err != nil {
		log.WithContext(msgCtx).WithError(err).Warning("Failed to decode the event header")
		s.Ack(msgCtx, stream, msg) // ACK the message cause we can't process the invalid header
		return
	}

	logEntry = domain.LogEventsReceivedCorrelationIdField.WithValue(header.CorrelationID).AsField(logEntry)
	logEntry = domain.LogEventsReceivedEventTypeField.WithValue(header.Event).AsField(logEntry)
	msgCtx = logShared.WithLogger(msgCtx, logEntry)

	// Body
	bodyRaw, _ := msg.Values["message"].(string)
	ctrtmsg := contract.Message{
		Header: *header,
		Body:   bodyRaw,
	}

	// Dispatching message
	ackWrapper := func() {
		s.Ack(msgCtx, stream, msg)
	}

	h.MsgReceived(msgCtx, ctrtmsg, ackWrapper)
	return
}

func (s *Subscriber) Ack(msgCtx context.Context, stream string, msg redis.XMessage) {
	if err := s.client.XAck(msgCtx, stream, s.serviceName, msg.ID).Err(); err != nil {
		log.Errorf("failed to XAck %s on stream %s: %v", msg.ID, stream, err)
	}
	log.WithContext(msgCtx).Info("Event Acknowledged")
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
