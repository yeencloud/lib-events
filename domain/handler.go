package domain

import (
	"context"

	"github.com/yeencloud/lib-events/contract"
)

type EventHandlerFunc func(ctx context.Context, body any) error

type EventHandler interface {
	Handle(event string, handler EventHandlerFunc)

	MsgReceived(event contract.Message)
}
