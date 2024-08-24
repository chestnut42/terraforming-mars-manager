package logx

import (
	"context"
	"log/slog"
)

type contextKey struct{}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

func Logger(ctx context.Context) *slog.Logger {
	l, ok := ctx.Value(contextKey{}).(*slog.Logger)
	if !ok {
		return slog.Default()
	}
	return l
}

func AddArgs(ctx context.Context, args ...any) context.Context {
	l := Logger(ctx)
	l = l.With(args...)
	return WithLogger(ctx, l)
}
