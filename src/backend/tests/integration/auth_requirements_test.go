package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// RF01: real HTTP handler, login use case, bcrypt and JWT; users are in memory.
func TestRF01_LoginContract(t *testing.T) {
	const secret = "rf01-test-signing-key"
	for _, tc := range []struct {
		name, payload string
		status        int
		code          string
	}{
		{"success", `{"email":"operador@motiva.com","password":"senha123"}`, 200, ""},
		{"wrong_password", `{"email":"operador@motiva.com","password":"wrong"}`, 401, "ERR_INVALID_CREDENTIALS"},
		{"unknown_user", `{"email":"unknown@example.test","password":"senha123"}`, 401, "ERR_INVALID_CREDENTIALS"},
		{"inactive_user", `{"email":"operador@motiva.com","password":"senha123"}`, 403, "ERR_USER_INACTIVE"},
		{"missing_email", `{"password":"senha123"}`, 400, "ERR_VALIDATION"},
		{"missing_password", `{"email":"operador@motiva.com"}`, 400, "ERR_VALIDATION"},
		{"empty_fields", `{"email":"","password":""}`, 400, "ERR_VALIDATION"},
		{"malformed_json", `{`, 400, "ERR_INVALID_PAYLOAD"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := setupAuthIntegrationServer(secret, tc.name != "inactive_user")
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, response.Code, response.Body.String())
			}
			var body struct {
				Token string                     `json:"token"`
				Code  string                     `json:"code"`
				User  map[string]json.RawMessage `json:"user"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != tc.code {
				t.Fatalf("expected code %q, got %q", tc.code, body.Code)
			}
			if tc.status != http.StatusOK {
				if body.Token != "" || body.User != nil {
					t.Fatal("failed login exposed a session or user data")
				}
				return
			}
			for key := range body.User {
				switch key {
				case "id", "name", "email", "role":
				default:
					t.Errorf("unexpected field in user response: %s", key)
				}
			}
			token, err := jwt.Parse(body.Token, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
			if err != nil || !token.Valid {
				t.Fatalf("login must return a valid signed token with expiration: %v", err)
			}
			claims := token.Claims.(jwt.MapClaims)
			if claims["sub"] != "usr_int_01" || claims["email"] != "operador@motiva.com" || claims["role"] != "operator" {
				t.Fatalf("unexpected identity: %+v", claims)
			}
			me := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
			me.Header.Set("Authorization", "Bearer "+body.Token)
			profile := httptest.NewRecorder()
			router.ServeHTTP(profile, me)
			var identity struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(profile.Body.Bytes(), &identity); err != nil {
				t.Fatal(err)
			}
			if profile.Code != http.StatusOK || identity.ID != "usr_int_01" {
				t.Fatalf("authenticated identity mismatch: %d %s", profile.Code, profile.Body.String())
			}
		})
	}
}
