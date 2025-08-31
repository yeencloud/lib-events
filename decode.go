package events

import (
	"context"
	"encoding/json"

	"github.com/yeencloud/lib-events/domain"
	"github.com/yeencloud/lib-shared/validation"
)

func DecodeEvent[T any](v *validation.Validator, ctx context.Context, eventJson string) (*T, error) {
	var decodedEvent T
	if err := json.Unmarshal([]byte(eventJson), &decodedEvent); err != nil {
		return nil, domain.EventUnmarshallingError[T]{
			Event: decodedEvent,
		}
	}

	err := v.StructCtx(ctx, decodedEvent)
	if err != nil {
		return nil, err
	}

	return &decodedEvent, nil
}
