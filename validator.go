package events

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/yeencloud/lib-shared/validation"
)

func eventNameValidator(ctx context.Context, fl validator.FieldLevel) error {
	value := fl.Field().String()

	if len(value) == 0 {
		return errors.New("event name cannot be empty")
	}

	// Events must be uppercase and can contain underscores
	re := regexp.MustCompile(`^[A-Z_]+$`) // TODO: MustCompile shold be used only once for performance
	if !re.MatchString(value) {
		return errors.New("event name must match the regex " + re.String())
	}

	// Enforcing this so event names are more natural (e.g. USER_CREATED instead of USER_CREATED_EVENT)
	if strings.HasSuffix(value, "_EVENT") {
		return errors.New("event name must not end with '_EVENT'")
	}
	return nil
}

func Validations() map[string]validation.ValidationFunc {
	return map[string]validation.ValidationFunc{
		"event_name": eventNameValidator,
	}
}
