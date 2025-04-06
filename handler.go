package events

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	metrics2 "github.com/yeencloud/lib-events/metrics"

	"github.com/yeencloud/lib-events/contract"
	"github.com/yeencloud/lib-events/domain"
	metrics "github.com/yeencloud/lib-metrics"
	logShared "github.com/yeencloud/lib-shared/log"
	sharedMetrics "github.com/yeencloud/lib-shared/metrics"
)

type BasicHandler struct {
	channel  string
	handlers *map[string]domain.EventHandlerFunc
}

func (b *BasicHandler) CreateLoggerForEvent(ctx context.Context, event contract.Message) context.Context {
	logEntry := log.NewEntry(log.StandardLogger())

	logEntry = domain.LogEventsReceivedChannelField.WithValue(b.channel).AsField(logEntry)
	logEntry = domain.LogEventsReceivedEventTypeField.WithValue(event.Header.Event).AsField(logEntry)
	logEntry = domain.LogEventsReceivedCorrelationIdField.WithValue(event.Header.CorrelationID).AsField(logEntry)

	return logShared.WithLogger(ctx, logEntry)
}

func (b *BasicHandler) CreateMetricsForRequest(ctx context.Context, event contract.Message) (metrics2.MessageReceivedMetric, context.Context) {
	ctx = metrics.SetTag(ctx, sharedMetrics.CorrelationIdKey.MetricKey(), event.Header.CorrelationID)

	var payload string
	data, err := json.Marshal(event.Body)
	if err != nil {
		payload = "Error: cannot marshal body"
	} else {
		payload = string(data)
	}

	metric := metrics2.MessageReceivedMetric{
		Channel: b.channel,
		Event:   event.Header.Event,
		Payload: payload,
		Status:  "",
	}

	return metric, ctx
}

func (b *BasicHandler) MsgReceived(event contract.Message) {
	if b.handlers == nil {
		b.handlers = &map[string]domain.EventHandlerFunc{}
	}
	handler, exists := (*b.handlers)[event.Header.Event]
	if exists {
		ctx := context.Background()
		ctx = b.CreateLoggerForEvent(ctx, event)
		metric, ctx := b.CreateMetricsForRequest(ctx, event)

		logShared.GetLoggerFromContext(ctx).Info("Received event: ", b.channel)
		err := handler(ctx, event.Body)

		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Event processing failed")
			metric.Status = "Error: " + err.Error()
		} else {
			log.WithContext(ctx).Info("Event processing succeeded")
			metric.Status = "Success"
		}

		_ = metrics.WritePoint(ctx, domain.ReceivedEventsMetricPointName, metric)
	}
}

func (b *BasicHandler) Handle(event string, handler domain.EventHandlerFunc) {
	if b.handlers == nil {
		b.handlers = &map[string]domain.EventHandlerFunc{}
	}

	log.WithField("event", fmt.Sprintf("%s:%s", b.channel, event)).Info("Registering event handler")
	(*b.handlers)[event] = handler
}
