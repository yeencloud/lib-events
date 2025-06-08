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
	}

	return metric, ctx
}

func (b *BasicHandler) handleResponse(ctx context.Context, err error, metric metrics2.MessageReceivedMetric) {
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Event processing failed")
		metric.Message = "Error: %s" + err.Error()
	} else {
		log.WithContext(ctx).Info("Event processing succeeded")
	}
	_ = metrics.WritePoint(ctx, domain.ReceivedEventsMetricPointName, metric)
}

func (b *BasicHandler) MsgReceived(ctx context.Context, event contract.Message) {
	if b.handlers == nil {
		b.handlers = &map[string]domain.EventHandlerFunc{}
	}

	serviceHandler, exists := (*b.handlers)[event.Header.Event]
	if !exists {
		return
	}

	metric, ctx := b.CreateMetricsForRequest(ctx, event)

	logShared.GetLoggerFromContext(ctx).Info("Received event: ", event.Header.Event)

	defer func() {
		if r := recover(); r != nil {
			metric.Message = fmt.Sprintf("Panic %v", r)
			log.WithContext(ctx).WithField("panic", metric.Message).WithField("trace", string(debug.Stack())).Error("Event processing did panic")
			_ = metrics.WritePoint(ctx, domain.ReceivedEventsMetricPointName, metric)
		}
	}()

	err := b.validator.StructCtx(ctx, event.Header)
	if err != nil {
		b.handleResponse(ctx, err, metric)
	} else {
		err = serviceHandler(ctx, event.Body)
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
