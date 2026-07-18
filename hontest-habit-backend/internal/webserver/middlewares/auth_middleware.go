package middlewares

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/insanelyharsh/hontest-habit/internal/app/auth"
	"github.com/insanelyharsh/hontest-habit/internal/common/errors"
	"github.com/insanelyharsh/hontest-habit/internal/constants"
)

// Authenticate validates the Authorization: Bearer <token> header against
// cfg and, on success, stores the token's claims on the request context.
// Writes the JSON error response itself rather than depending on the
// webserver package's helpers, to avoid an import cycle (webserver imports
// middlewares for the global TraceID wrap).
func Authenticate(cfg auth.JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || token == "" {
				writeAuthError(w, errors.Unauthorized("missing bearer token", nil))
				return
			}

			claims, err := auth.ValidateToken(r.Context(), cfg, token)
			if err != nil {
				writeAuthError(w, err)
				return
			}

			ctx := context.WithValue(r.Context(), constants.ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext returns the claims set by Authenticate, if any.
func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(constants.ClaimsContextKey).(*auth.Claims)
	return claims, ok
}

func writeAuthError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errors.StatusCode(err))
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid or missing token"})
}
