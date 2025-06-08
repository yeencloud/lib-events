package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yeencloud/lib-shared/validation"
)

// TODO: Use real errors
func DecodeEvent[T any](v *validation.Validator, ctx context.Context, eventJson string) (*T, error) {
	var decodedEvent T
	if err := json.Unmarshal([]byte(eventJson), &decodedEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to %T: %w", decodedEvent, err)
	}

	err := v.StructCtx(ctx, decodedEvent)
	if err != nil {
		return nil, err
	}

	return &decodedEvent, nil
}
