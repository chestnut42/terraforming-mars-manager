package docs

import (
	"log/slog"
	"net/http"

	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
)

func NewStaticHandler(data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(data); err != nil {
			logx.Logger(r.Context()).Warn("error writing response", slog.Any("error", err))
		}
	}
}
