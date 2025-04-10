package events

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"

	log "github.com/sirupsen/logrus"
	metrics2 "github.com/yeencloud/lib-events/metrics"
	"github.com/yeencloud/lib-shared/validation"

	"github.com/yeencloud/lib-events/contract"
	"github.com/yeencloud/lib-events/domain"
	metrics "github.com/yeencloud/lib-metrics"
	logShared "github.com/yeencloud/lib-shared/log"
	sharedMetrics "github.com/yeencloud/lib-shared/metrics"
)

type BasicHandler struct {
	channel   string
	handlers  *map[string]domain.EventHandlerFunc
	validator *validation.Validator
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

func (b *BasicHandler) handleResponse(ctx context.Context, err error, metric metrics2.MessageReceivedMetric) {
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Event processing failed")
		metric.Status = "Error"
		metric.Message = err.Error()
	} else {
		log.WithContext(ctx).Info("Event processing succeeded")
		metric.Status = "Success"
	}
	_ = metrics.WritePoint(ctx, domain.ReceivedEventsMetricPointName, metric)
}

func (b *BasicHandler) MsgReceived(event contract.Message) {
	if b.handlers == nil {
		b.handlers = &map[string]domain.EventHandlerFunc{}
	}

	handler, exists := (*b.handlers)[event.Header.Event]
	if !exists {
		return
	}

	ctx := context.Background()
	ctx = b.CreateLoggerForEvent(ctx, event)
	metric, ctx := b.CreateMetricsForRequest(ctx, event)

	logShared.GetLoggerFromContext(ctx).Info("Received event: ", event.Header.Event)

	defer func() {
		if r := recover(); r != nil {
			metric.Status = "Panic"
			metric.Message = fmt.Sprintf("%v", r)
			// TODO: Change error text (panic'ed feels weird)
			log.WithContext(ctx).WithField("panic", metric.Message).WithField("trace", string(debug.Stack())).Error("Event processing panic'ed")
			_ = metrics.WritePoint(ctx, domain.ReceivedEventsMetricPointName, metric)
		}
	}()

	err := b.validator.StructCtx(ctx, event.Header)
	if err != nil {
		b.handleResponse(ctx, err, metric)
	} else {
		err = handler(ctx, event.Body) // Call the handler on the service
		b.handleResponse(ctx, err, metric)
	}
}

func (b *BasicHandler) Handle(event string, handler domain.EventHandlerFunc) {
	if b.handlers == nil {
		b.handlers = &map[string]domain.EventHandlerFunc{}
	}

	log.WithField("event", fmt.Sprintf("%s:%s", b.channel, event)).Info("Registering event handler")
	(*b.handlers)[event] = handler
}
