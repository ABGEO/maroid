package logger

import (
	"context"
	"log/slog"

	"github.com/abgeo/maroid/libs/rest"
)

// requestHandler adds the flow identifier to every record that carries the
// context of a request.
type requestHandler struct {
	slog.Handler
}

// WithRequest wraps a handler so that every record of a request carries the
// flow identifier of that request.
func WithRequest(handler slog.Handler) slog.Handler {
	return requestHandler{Handler: handler}
}

// Handle adds the `flow_id` attribute and passes the record on. LOG-009 gives
// the name. The value is bare, without the prefix that an instance member holds.
//
// Copies of a Record share state, so this clones before it adds. A handler that
// fans out gives the same record to each of its handlers.
func (h requestHandler) Handle(ctx context.Context, record slog.Record) error {
	if identifier := rest.FlowIDFromContext(ctx); identifier != "" {
		record = record.Clone()
		record.AddAttrs(slog.String("flow_id", identifier))
	}

	//nolint:wrapcheck // The wrapped handler owns the error of the write.
	return h.Handler.Handle(ctx, record)
}

// WithAttrs keeps the wrapper in place when a component adds its attributes.
func (h requestHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return requestHandler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup keeps the wrapper in place when a component opens a group.
func (h requestHandler) WithGroup(name string) slog.Handler {
	return requestHandler{Handler: h.Handler.WithGroup(name)}
}
