package telemetry

// Event names should use underscores because they will be ingested into
// Snowflake and then snakeCase becomes SNAKECASE which makes it a
// little hard to read.
const EventServerHeartbeat = "server_heartbeat"
const EventServerUserAction = "server_user_action"

const ServerHeartbeatTypeInitialized = "initialized"
const ServerHeartbeatTypePanic = "panic"

const ServerUserActionTypeCommandExecuted = "commandExecuted"
const ServerUserActionTypeFileAnalyzed = "fileAnalyzed"

type TelemetryPayload struct {
	Records []Record `json:"records"`
}

type Record struct {
	Event      string         `json:"event"`
	Source     string         `json:"source"`
	Timestamp  int64          `json:"event_timestamp"`
	Properties map[string]any `json:"properties"`
}
