package events

import (
	"encoding/json"
	"fmt"
)

// TODO: Use real errors
func DecodeEvent[T any](event any) (*T, error) {
	eventMap, ok := event.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected event type: %T", event)
	}

	eventJSON, err := json.Marshal(eventMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event map: %w", err)
	}

	var decodedEvent T
	if err := json.Unmarshal(eventJSON, &decodedEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to %T: %w", decodedEvent, err)
	}

	// TODO: Add Validation for T
	return &decodedEvent, nil
}
