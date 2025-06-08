package domain

import (
	"context"

	"github.com/yeencloud/lib-events/contract"
)

type EventHandlerFunc func(ctx context.Context, eventJson string) error

type EventHandler interface {
	Handle(event string, handler EventHandlerFunc)

	MsgReceived(ctx context.Context, event contract.Message)
}
