package events

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/yeencloud/lib-shared/validation"
)

var eventNameValidatorRegex = regexp.MustCompile("^[A-Z_]+$")

var errEventNameMustNotBeEmpty = errors.New("event name cannot be empty")
var errEventNameMustNotEndWithUnderscoreEvent = errors.New("event name must not end with '_EVENT'")
var errEventNameMustValidateRegex = errors.New("event name must match the regex " + eventNameValidatorRegex.String())

func eventNameValidator(ctx context.Context, fl validator.FieldLevel) error {
	value := fl.Field().String()

	if len(value) == 0 {
		return errEventNameMustNotBeEmpty
	}

	// Events must be uppercase and can contain underscores
	if !eventNameValidatorRegex.MatchString(value) {
		return errEventNameMustValidateRegex
	}

	// Enforcing this so event names are more natural (e.g. USER_CREATED instead of USER_CREATED_EVENT)
	if strings.HasSuffix(value, "_EVENT") {
		return errEventNameMustNotEndWithUnderscoreEvent
	}
	return nil
}

func Validations() map[string]validation.ValidationFunc {
	return map[string]validation.ValidationFunc{
		"event_name": eventNameValidator,
	}
}
