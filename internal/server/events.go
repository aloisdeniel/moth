package server

import (
	"context"

	"github.com/aloisdeniel/moth/internal/events"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// eventSink adapts the store to the async event writer's BatchInserter:
// it stamps IDs and JSON-encodes metadata at write time, then lands the
// whole batch in one transaction.
type eventSink struct {
	store store.EventStore
}

func (s eventSink) InsertEvents(ctx context.Context, batch []events.Event) error {
	rows := make([]store.Event, len(batch))
	for i, e := range batch {
		rows[i] = store.Event{
			ID:         authrpc.NewID(),
			ProjectID:  e.ProjectID,
			UserID:     e.UserID,
			Type:       e.Type,
			Provider:   e.Provider,
			Platform:   e.Platform,
			SDKVersion: e.SDKVersion,
			Metadata:   e.MetadataJSON(),
			CreatedAt:  e.CreatedAt,
		}
	}
	return s.store.InsertEvents(ctx, rows)
}
