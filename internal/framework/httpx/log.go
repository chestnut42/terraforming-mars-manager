package httpx

import (
	"log/slog"
	"net/http"

	"github.com/felixge/httpsnoop"

	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
)

func WithLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := httpsnoop.CaptureMetrics(h, w, r)
		logx.Logger(r.Context()).Info("request served",
			slog.String("raddr", r.Header.Get("x-forwarded-for")),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("code", m.Code),
			slog.Duration("dt", m.Duration),
			slog.Int64("written", m.Written))
	})
}
