package contract

type Message[T any] struct {
	Header Header
	Body   T
}
