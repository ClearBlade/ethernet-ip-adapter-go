package main

type ethernetIpAdapterSettings struct {
	EndpointIp   string `json:"endpoint_ip"`
	EndpointPort uint   `json:"endpoint_tcp_port"`
}

type ethernetIpReadRequestMQTTMessage struct {
	Tags []string `json:"tags"`
}

type ethernetIpReadResponseMQTTMessage struct {
	ServerTimestamp string                                `json:"server_timestamp"`
	Data            map[string]ethernetIpReadResponseData `json:"data"`
	Success         bool                                  `json:"success"`
	StatusCode      uint32                                `json:"status_code"`
	ErrorMessage    string                                `json:"error_message"`
}

type ethernetIpReadResponseData struct {
	Value           interface{} `json:"value"`
	SourceTimestamp string      `json:"source_timestamp"`
}

type ethernetIpWriteRequestMQTTMessage struct {
	NodeID string      `json:"node_id"`
	Value  interface{} `json:"value"`
}

type ethernetIpWriteResponseMQTTMessage struct {
	NodeID       string `json:"node_id"`
	Timestamp    string `json:"timestamp"`
	Success      bool   `json:"success"`
	StatusCode   uint32 `json:"status_code"`
	ErrorMessage string `json:"error_message"`
}

type ethernetIpMethodRequestMQTTMessage struct {
	ObjectID       string        `json:"object_id"`
	MethodID       string        `json:"method_id"`
	InputArguments []interface{} `json:"arguments"`
}

type ethernetIpMethodResponseMQTTMessage struct {
	ObjectID       string        `json:"object_id"`
	MethodID       string        `json:"method_id"`
	Timestamp      string        `json:"timestamp"`
	Success        bool          `json:"success"`
	StatusCode     uint32        `json:"status_code"`
	ErrorMessage   string        `json:"error_message"`
	InputArguments []interface{} `json:"arguments"`
	OutputValues   []interface{} `json:"values"`
}

type SubscriptionOperationType string

// TODO - Add missing subscription operation types when they are implemented by github.com/gopcua
// * ModifySubscription
// * SetPublishingMode
// * Republish
// * TransferSubscriptions
const (
	SubscriptionCreate    SubscriptionOperationType = "create"
	SubscriptionRepublish SubscriptionOperationType = "republish"
	SubscriptionPublish   SubscriptionOperationType = "publish"
	SubscriptionDelete    SubscriptionOperationType = "delete"
)

// https://reference.opcfoundation.org/v104/Core/docs/Part4/5.13.2/
//
// publish_interval - The minimum amount of time (milliseconds) between updates
// lifetime - How long the connection to the OPC UA server is preserved in the absence of updates before it is killed and recreated.
// keepalive - The maximum number of times the publish timer expires without sending any notifications before sending a keepalive message.
// max_publish_notifications - The maximum number of notifications that the Client wishes to receive in a single Publish response.
// priority - Indicates the relative priority of the Subscription
//
type ethernetIpSubscriptionCreateParmsMQTTMessage struct {
	PublishInterval            *uint32                                     `json:"publish_interval,omitempty"`
	LifetimeCount              *uint32                                     `json:"lifetime,omitempty"`
	MaxKeepAliveCount          *uint32                                     `json:"keepalive,omitempty"`
	MaxNotificationsPerPublish *uint32                                     `json:"max_publish_notifications,omitempty"`
	Priority                   *uint8                                      `json:"priority,omitempty"`
	MonitoredItems             *[]ethernetIpMonitoredItemCreateMQTTMessage `json:"items_to_monitor,omitempty"`
}

type ethernetIpSubscriptionRepublishParmsMQTTMessage struct {
	SubscriptionID uint32 `json:"subscription_id"`
}

type ethernetIpSubscriptionDeleteParmsMQTTMessage struct {
	SubscriptionID uint32 `json:"subscription_id"`
}

//TODO - MonitoringParameters - Build out structure, figure out how to implement Filter
//TODO - Uncomment AttributeID when we are ready to handle more than ua.AttributeIDValue
type ethernetIpMonitoredItemCreateMQTTMessage struct {
	NodeID string `json:"node_id"`
	Values bool   `json:"values"`
	Events bool   `json:"events"`
	//AttributeID uint32 `json:"attribute_id"` - For now, we will only use attribute id 13 (AttributeIDValue)
	//
	// TODO - Implement later
	//
	//Monitoring params - We will be using defaults for now
	// SamplingInterval float64
	// Filter           *ExtensionObject
	// QueueSize        uint32
	// DiscardOldest    bool
}

//TODO - Uncomment AttributeID when we are ready to handle more than ua.AttributeIDValue
type ethernetIpMonitoredItemCreateResultMQTTMessage struct {
	NodeID string `json:"node_id"`
	//AttributeID             uint32  `json:"attribute_id"`
	ClientHandle            uint32  `json:"client_handle"`
	DiscardOldest           bool    `json:"disgard_oldest"`
	StatusCode              uint32  `json:"status_code"`
	RevisedSamplingInterval float64 `json:"revised_sampling_interval"`
	RevisedQueueSize        uint32  `json:"revised_queue_size"`
	FilterResult            interface{}
	MonitoringMode          uint32 `json:"monitoring_mode"`
	TimestampsToReturn      uint32 `json:"timestamps_to_return,omitempty"`
}

//TODO - Uncomment AttributeID when we are ready to handle more than ua.AttributeIDValue
type ethernetIpMonitoredItemNotificationMQTTMessage struct {
	NodeID string `json:"node_id"`
	//AttributeID             uint32  `json:"attribute_id"`
	ClientHandle uint32                 `json:"client_handle"`
	Value        interface{}            `json:"value,omitempty"`
	Event        ethernetIpEventMessage `json:"event,omitempty"`
}

//eventFieldNames        = []string{"EventId", "EventType", "Severity", "Time", "Message"}
type ethernetIpEventMessage struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	Severity  uint32 `json:"severity"`
	Time      string `json:"time"`
	Message   string `json:"message"`
}

type ethernetIpSubscriptionRequestMQTTMessage struct {
	RequestType   SubscriptionOperationType `json:"request_type"`
	RequestParams *interface{}              `json:"request_params,omitempty"`
}

type ethernetIpSubscriptionResponseMQTTMessage struct {
	RequestType    SubscriptionOperationType `json:"request_type"`
	SubscriptionID uint32                    `json:"subscription_id"`
	Timestamp      string                    `json:"timestamp"`
	Success        bool                      `json:"success"`
	StatusCode     uint32                    `json:"status_code"`
	ErrorMessage   string                    `json:"error_message"`
	Results        []interface{}             `json:"results"`
}
