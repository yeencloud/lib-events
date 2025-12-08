package events

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"

	log "github.com/sirupsen/logrus"
	"github.com/yeencloud/lib-shared/validation"

	"github.com/yeencloud/lib-events/contract"
	"github.com/yeencloud/lib-events/domain"
	metrics "github.com/yeencloud/lib-metrics"
	logShared "github.com/yeencloud/lib-shared/log"
	sharedMetrics "github.com/yeencloud/lib-shared/metrics"
)

type BasicHandler struct {
	channel   string
	handlers  map[string]domain.EventHandlerFunc
	validator *validation.Validator
}

func (b *BasicHandler) CreateMetricsForRequest(ctx context.Context, event contract.Message) (domain.MessageReceivedMetric, context.Context) {
	ctx = metrics.SetTag(ctx, sharedMetrics.CorrelationIdKey.MetricKey(), event.Header.CorrelationID)

	var payload string
	data, err := json.Marshal(event.Body)
	if err != nil {
		payload = "Error: cannot marshal body: %e" + err.Error()
	} else {
		payload = string(data)
	}

	metric := domain.MessageReceivedMetric{
		Channel: b.channel,
		Event:   event.Header.Event,
		Payload: payload,
	}

	return metric, ctx
}

func (b *BasicHandler) handleResponse(ctx context.Context, err error, metric domain.MessageReceivedMetric) {
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Event processing failed")
		metric.Message = "Error: " + err.Error()
	} else {
		log.WithContext(ctx).Info("Event processing succeeded")
	}
	_ = metrics.WritePoint(ctx, domain.ReceivedEventsMetricPointName, metric)
}

func (b *BasicHandler) handlePanic(ctx context.Context, metric domain.MessageReceivedMetric, recovered any) {
	metric.Message = fmt.Sprintf("Panic %v", recovered)
	log.WithContext(ctx).WithField("panic", metric.Message).WithField("trace", string(debug.Stack())).Error("Event processing did panic")
	_ = metrics.WritePoint(ctx, domain.ReceivedEventsMetricPointName, metric)
}

func (b *BasicHandler) MsgReceived(ctx context.Context, event contract.Message, ack func()) {
	if b.handlers == nil {
		// This event isn't handled by this service, silently return and ack
		ack()
		return
	}

	serviceHandler, handlerRegistered := b.handlers[event.Header.Event]
	if !handlerRegistered {
		ack()
		return
	}

	metric, ctx := b.CreateMetricsForRequest(ctx, event)

	logShared.GetLoggerFromContext(ctx).Info("Received event: ", event.Header.Event)

	err := b.validator.StructCtx(ctx, event.Header)
	if err != nil {
		ack()
		return
	}

	defer func() {
		if r := recover(); r != nil {
			b.handlePanic(ctx, metric, r)
		}
	}()
	err = serviceHandler(ctx, event.Body)
	b.handleResponse(ctx, err, metric)
	ack() // shouldn't ack if panic happens
}

func (b *BasicHandler) Register(event string, handler domain.EventHandlerFunc) {
	if b.handlers == nil {
		return
	}

	log.WithField("event", fmt.Sprintf("%s:%s", b.channel, event)).Info("Registering event handler")
	b.handlers[event] = handler
}
