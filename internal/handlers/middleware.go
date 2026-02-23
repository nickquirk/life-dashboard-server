package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/utils"
	"google.golang.org/api/idtoken"
)

func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the cookie
		cookie, err := r.Cookie("life-dashboard") // Ensure name matches SetCookie in google.go
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate token and get User ID
		userID, err := utils.GetUserIdFromToken(cookie.Value)
		if err != nil {
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		// Store UserID in context so the handler can use it
		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		// Call the next handler with the new context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthenticateOIDC validates the OIDC identity token that Cloud Scheduler
// sends when configured with a service account. This makes IAM the primary
// authn layer on Cloud Run; the shared secret in the handler is a secondary check.
//
// Requires CLOUD_RUN_URL to be set (used as the expected audience).
// In local dev (CLOUD_RUN_URL unset) the middleware is a no-op so you can
// still test with just the shared secret.
func AuthenticateOIDC(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		audience := os.Getenv("CLOUD_RUN_URL")
		if audience == "" {
			// Local development — skip OIDC, fall through to shared-secret check.
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			slog.Warn("OIDC: missing or malformed Authorization header")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")

		payload, err := idtoken.Validate(r.Context(), token, audience)
		if err != nil {
			slog.Warn("OIDC: token validation failed", "error", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Optional: lock down to a specific service account.
		if expected := os.Getenv("SCHEDULER_SERVICE_ACCOUNT"); expected != "" {
			if email, _ := payload.Claims["email"].(string); email != expected {
				slog.Warn("OIDC: unexpected caller", "email", email, "expected", expected)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// To give extra time to slow requests, eg syncUsers
func WithExtendedTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rc := http.NewResponseController(w)
			rc.SetWriteDeadline(time.Now().Add(timeout))
			next.ServeHTTP(w, r)
		})
	}
}
