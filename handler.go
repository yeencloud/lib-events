package events

import (
	"context"
	"fmt"
	"runtime/debug"

	log "github.com/sirupsen/logrus"
	"github.com/yeencloud/lib-shared/validation"

	"github.com/yeencloud/lib-events/contract"
	"github.com/yeencloud/lib-events/domain"
	logShared "github.com/yeencloud/lib-shared/log"
)

type BasicHandler struct {
	channel   string
	handlers  map[string]domain.EventHandlerFunc
	validator *validation.Validator
}

func (b *BasicHandler) handleResponse(ctx context.Context, err error) {
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Event processing failed")
	} else {
		log.WithContext(ctx).Info("Event processing succeeded")
	}
}

func (b *BasicHandler) handlePanic(ctx context.Context, recovered any) {
	msg := fmt.Sprintf("Panic %v", recovered)
	log.WithContext(ctx).WithField("panic", msg).WithField("trace", string(debug.Stack())).Error("Event processing did panic")
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

	logShared.GetLoggerFromContext(ctx).Info("Received event: ", event.Header.Event)

	err := b.validator.StructCtx(ctx, event.Header)
	if err != nil {
		ack()
		return
	}

	defer func() {
		if r := recover(); r != nil {
			b.handlePanic(ctx, r)
		}
	}()
	err = serviceHandler(ctx, event.Body)
	b.handleResponse(ctx, err)
	ack() // shouldn't ack if panic happens
}

func (b *BasicHandler) Register(event string, handler domain.EventHandlerFunc) {
	if b.handlers == nil {
		return
	}

	log.WithField("event", fmt.Sprintf("%s:%s", b.channel, event)).Info("Registering event handler")
	b.handlers[event] = handler
}
