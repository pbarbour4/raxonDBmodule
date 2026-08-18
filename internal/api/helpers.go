package api

import (
	"net/http"
	"strconv"
	"time"

	"raxonplatform/internal/contracts"
)

func pagination(r *http.Request) (limit, offset int) {
	limit = 10
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 1 {
			offset = (p - 1) * limit
		}
	}
	return
}

func buildEventEnvelope(eventType string, payload map[string]any, requestedBy string) contracts.EventEnvelope {
	return contracts.EventEnvelope{
		ChannelName:        "tokenization-channel",
		EventType:          eventType,
		Payload:            payload,
		ChaincodeEventName: eventType,
		OccurredAt:         time.Now(),
		IngestedAt:         time.Now(),
		SchemaVersion:      1,
		Metadata:           map[string]any{"requested_by": requestedBy},
	}
}
