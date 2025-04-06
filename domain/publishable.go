package domain

type PublishableMessage interface {
	Channel() string
	EventType() string
}
