package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// RF02: exercise the application's actual route registration, including future
// API routes. Login is the only public /api/v1 endpoint in this contract.
func TestRF02_AllProtectedRoutesRejectUnauthorizedAccess(t *testing.T) {
	const secret = "rf02-test-signing-key"
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "rf02-user", "email": "rf02@example.test", "role": "enginner",
		"exp": time.Now().Add(-time.Hour).Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	wrongSignature, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "rf02-user", "role": "enginner", "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("another-test-key"))
	if err != nil {
		t.Fatal(err)
	}
	for _, environment := range []string{"production", "development"} {
		t.Run(environment, func(t *testing.T) {
			// Nil use cases deliberately make reaching business handlers a failure.
			router := buildRouter(&config.Config{Environment: environment}, slog.New(slog.NewTextHandler(io.Discard, nil)), appHandlers{
				auth: httpAdapter.NewAuthHandler(nil), upload: httpAdapter.NewUploadHandler(nil, nil),
				maint: httpAdapter.NewMaintenanceHandler(nil), alarm: httpAdapter.NewAlarmHandler(nil),
				ins: httpAdapter.NewInspectionHandler(nil), dash: httpAdapter.NewDashboardHandler(nil),
				health: httpAdapter.NewHealthHandler(nil), docs: httpAdapter.NewDocsHandler("../../docs/openapi.json"),
				jwtSecret: secret,
			})
			checked := 0
			for _, route := range router.Routes() {
				if !strings.HasPrefix(route.Path, "/api/v1/") || route.Path == "/api/v1/auth/login" {
					continue
				}
				checked++
				path := strings.NewReplacer(":id", "rf02-id", ":code", "FAL", ":version", "v1").Replace(route.Path)
				for _, tc := range []struct{ name, header, code string }{
					{"missing", "", "ERR_MISSING_TOKEN"},
					{"malformed", "Bearer invalid.jwt", "ERR_INVALID_TOKEN"},
					{"expired", "Bearer " + expired, "ERR_INVALID_TOKEN"},
					{"wrong_signature", "Bearer " + wrongSignature, "ERR_INVALID_TOKEN"},
				} {
					t.Run(route.Method+" "+route.Path+"/"+tc.name, func(t *testing.T) {
						req := httptest.NewRequest(route.Method, path, nil)
						if tc.header != "" {
							req.Header.Set("Authorization", tc.header)
						}
						response := httptest.NewRecorder()
						router.ServeHTTP(response, req)
						var problem httpAdapter.ProblemDetails
						if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
							t.Fatalf("expected problem JSON: %v; body=%s", err, response.Body.String())
						}
						if response.Code != http.StatusUnauthorized || problem.Status != 401 || problem.Code != tc.code {
							t.Fatalf("expected 401/%s, got %d: %s", tc.code, response.Code, response.Body.String())
						}
						var body map[string]json.RawMessage
						if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
							t.Fatal(err)
						}
						for key := range body {
							switch key {
							case "type", "title", "status", "detail", "instance", "code", "request_id":
							default:
								t.Errorf("unexpected data in rejection response: %s", key)
							}
						}
					})
				}
			}
			if checked == 0 {
				t.Fatal("no protected routes were checked")
			}
			t.Logf("checked %d protected routes with 4 unauthorized inputs", checked)
		})
	}
}
