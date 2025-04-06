package domain

import (
	"github.com/yeencloud/lib-shared/namespace"
)

const ReceivedEventsMetricPointName = "received_events"
const PublishedEventsMetricPointName = "published_events"

var (
	LogEventsScope = namespace.Namespace{Identifier: ReceivedEventsMetricPointName}

	LogEventsSubscriptionsScope = namespace.Namespace{Identifier: "subscriptions"}
	LogEventsReceivedScope      = namespace.Namespace{Identifier: "received"}
)

var (
	LogEventsSubscriptionChannelField = namespace.Namespace{Parent: &LogEventsSubscriptionsScope, Identifier: "channel"}
	LogEventsSubscriptionCountField   = namespace.Namespace{Parent: &LogEventsSubscriptionsScope, Identifier: "count"}
	LogEventsSubscriptionKindField    = namespace.Namespace{Parent: &LogEventsSubscriptionsScope, Identifier: "kind"}

	LogEventsReceivedChannelField       = namespace.Namespace{Parent: &LogEventsReceivedScope, Identifier: "channel"}
	LogEventsReceivedEventTypeField     = namespace.Namespace{Parent: &LogEventsReceivedScope, Identifier: "event_type"}
	LogEventsReceivedCorrelationIdField = namespace.Namespace{Parent: &LogEventsReceivedScope, Identifier: "correlation_id"}
	LogEventsReceivedPayloadField       = namespace.Namespace{Parent: &LogEventsReceivedScope, Identifier: "payload"}
	LogEventsReceivedStatusField        = namespace.Namespace{Parent: &LogEventsReceivedScope, Identifier: "status"}
)
