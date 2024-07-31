package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/chestnut42/terraforming-mars-manager/internal/auth"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
)

const bearerPrefix = "Bearer "

type BearerAuth interface {
	Authenticate(ctx context.Context, token string) (*auth.User, error)
}

func WithAuthorization(h http.Handler, bearer BearerAuth) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			h.ServeHTTP(w, r)
			return
		}

		if !strings.HasPrefix(authHeader, bearerPrefix) {
			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write([]byte("Authorization header does not start with bearer prefix")); err != nil {
				logx.Logger(ctx).Debug("failed to write response", slog.Any("error", err))
			}
			return
		}

		token := strings.TrimPrefix(authHeader, bearerPrefix)
		user, err := bearer.Authenticate(ctx, token)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			if _, err := w.Write([]byte("Failed to parse token: " + err.Error())); err != nil {
				logx.Logger(ctx).Debug("failed to write response", slog.Any("error", err))
			}
			return
		}

		ctx = auth.ContextWithUser(ctx, user)
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}
