package domain

import "github.com/yeencloud/lib-shared/namespace"

type MessageReceivedMetric struct {
	Channel string `metric:"channel"`
	Event   string `metric:"event"`
	Payload string `metric:"payload"`
	Message string `metric:"message"`
}

var MessageIdKey = namespace.Namespace{Identifier: "message_id", IsMetricTag: true}
