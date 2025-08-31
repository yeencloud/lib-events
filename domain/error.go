package domain

import (
	"errors"
	"fmt"
)

type EventUnmarshallingError[T any] struct {
	Event T
}

func (e EventUnmarshallingError[T]) Error() string {
	return fmt.Sprintf("failed to unmarshal to %T", e.Event)
}

type FailedToCreateStreamError struct {
	Stream string
}

func (f FailedToCreateStreamError) Error() string {
	return "Failed to create stream:" + f.Stream
}

var ErrEventHeaderShouldBeAString = errors.New("the event header should be a string but got an unsupported type")
var ErrUnableToUnmarshalEventHeader = errors.New("unable to unmarshal the header in a json document")
var ErrFailedToValidateEventHeader = errors.New("failed to validate event header")
var ErrUnableToReadFromGroup = errors.New("unable to read from group")
