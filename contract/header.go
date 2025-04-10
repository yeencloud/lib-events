package contract

type Header struct {
	Date          string  `validate:"required,date_time"`
	Event         string  `validate:"required,event_name"`
	CorrelationID string  `validate:"required,uuid"`
	UserID        *string `validate:"omitempty,uuid"`
}
