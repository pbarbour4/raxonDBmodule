package contracts

import "time"

type EventEnvelope struct {
	EventID            string                 `json:"event_id"`
	ChannelName        string                 `json:"channel_name"`
	BlockHeight        int64                  `json:"block_height"`
	TxID               string                 `json:"tx_id"`
	TxIndex            int                    `json:"tx_index"`
	EventType          string                 `json:"event_type"`
	EventSubtype       string                 `json:"event_subtype"`
	Payload            map[string]any         `json:"payload"`
	ChaincodeEventName string                 `json:"chaincode_event_name"`
	OccurredAt         time.Time              `json:"occurred_at"`
	IngestedAt         time.Time              `json:"ingested_at"`
	SchemaVersion      int                    `json:"schema_version"`
	TraceID            string                 `json:"trace_id"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}
