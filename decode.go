package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yeencloud/lib-shared/validation"
)

// TODO: Use real errors
func DecodeEvent[T any](v *validation.Validator, ctx context.Context, event any) (*T, error) {
	eventMap, ok := event.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected event type: %T", event)
	}

	eventJSON, err := json.Marshal(eventMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event map: %w", err)
	}

	var decodedEvent T
	if err = json.Unmarshal(eventJSON, &decodedEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to %T: %w", decodedEvent, err)
	}

	err = v.StructCtx(ctx, decodedEvent)
	if err != nil {
		return nil, err
	}

	return &decodedEvent, nil
}
