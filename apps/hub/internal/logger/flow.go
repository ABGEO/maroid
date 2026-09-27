package logger

import (
	"context"
	"log/slog"

	"github.com/abgeo/maroid/libs/rest/flow"
)

// flowIDHandler adds the flow identifier to every record that carries the
// context of a request.
type flowIDHandler struct {
	slog.Handler
}

// WithFlowID wraps a handler so that every record of a request carries the
// flow identifier of that request.
func WithFlowID(handler slog.Handler) slog.Handler {
	return flowIDHandler{Handler: handler}
}

// Handle adds the `flow_id` attribute and passes the record on. The value is
// bare, without the prefix that an instance member holds.
//
// Copies of a Record share state, so this clones before it adds. A handler that
// fans out gives the same record to each of its handlers.
func (h flowIDHandler) Handle(ctx context.Context, record slog.Record) error {
	if identifier := flow.IDFromContext(ctx); identifier != "" {
		record = record.Clone()
		record.AddAttrs(slog.String("flow_id", identifier))
	}

	//nolint:wrapcheck // The wrapped handler owns the error of the write.
	return h.Handler.Handle(ctx, record)
}

// WithAttrs keeps the wrapper in place when a component adds its attributes.
func (h flowIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return flowIDHandler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup keeps the wrapper in place when a component opens a group.
func (h flowIDHandler) WithGroup(name string) slog.Handler {
	return flowIDHandler{Handler: h.Handler.WithGroup(name)}
}
