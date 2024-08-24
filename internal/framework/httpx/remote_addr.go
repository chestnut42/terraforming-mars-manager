package httpx

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
)

type remoteAddrKey struct{}

func ContextWithRemoteAddr(ctx context.Context, remoteAddr string) context.Context {
	return context.WithValue(ctx, remoteAddrKey{}, remoteAddr)
}

func RemoteAddrFromContext(ctx context.Context) (string, bool) {
	remoteAddr, ok := ctx.Value(remoteAddrKey{}).(string)
	return remoteAddr, ok
}

func WithRemoteAddress(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteAddr := r.Header.Get("x-forwarded-for")
		if remoteAddr != "" {
			ctx := r.Context()
			ctx = ContextWithRemoteAddr(ctx, remoteAddr)
			ctx = logx.AddArgs(ctx, slog.String("raddr", remoteAddr))
			r = r.WithContext(ctx)
		}

		h.ServeHTTP(w, r)
	})
}
