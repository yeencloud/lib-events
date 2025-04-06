package metrics

type MessageReceivedMetric struct {
	Channel string `metric:"channel"`
	Event   string `metric:"event"`
	Payload string `metric:"payload"`
	Status  string `metric:"status"`
}
