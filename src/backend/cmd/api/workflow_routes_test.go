package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/adapter/security"
	"github.com/g04/backend/internal/config"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
)

type routePredictionRepo struct {
	port.PredictionRepository
	saved int
}

func (*routePredictionRepo) ListDoorsForInference(context.Context) ([]domain.Door, error) {
	return []domain.Door{{StationCode: "FAL", Platform: "P1", DoorNumber: 4, Status: "active"}}, nil
}
func (r *routePredictionRepo) SaveRun(context.Context, *domain.PredictionRun, []domain.PredictionItem) error {
	r.saved++
	return nil
}

type routeMLClient struct{ calls int }

func (m *routeMLClient) PredictBatch(context.Context, []domain.Door) (*port.MLPredictionBatch, error) {
	m.calls++
	p := 0.2
	return &port.MLPredictionBatch{ModelVersion: "route-test-v1", Predictions: []domain.PredictionItem{{StationCode: "FAL", Platform: "P1", DoorNumber: 4, Probability: &p}}}, nil
}

type routeUOWStore struct {
	port.UnitOfWorkStore
	repo *routePredictionRepo
}

func (s *routeUOWStore) Predictions() port.PredictionRepository { return s.repo }

type routeUOW struct{ store port.UnitOfWorkStore }

func (u *routeUOW) Do(_ context.Context, fn func(port.UnitOfWorkStore) error) error {
	return fn(u.store)
}

func TestRFIntegration_ActualPredictionRouteRoles(t *testing.T) {
	for _, environment := range []string{"production", "development"} {
		for _, role := range []string{"operator", "enginner"} {
			t.Run(environment+"/"+role, func(t *testing.T) {
				const secret = "workflow-route-test-key"
				repo := &routePredictionRepo{}
				ml := &routeMLClient{}
				uc := usecase.NewDashboardUseCase(nil, repo, ml, nil, &routeUOW{store: &routeUOWStore{repo: repo}})
				router := buildRouter(&config.Config{Environment: environment}, slog.New(slog.NewTextHandler(io.Discard, nil)), appHandlers{
					auth: httpAdapter.NewAuthHandler(nil), upload: httpAdapter.NewUploadHandler(nil, nil), maint: httpAdapter.NewMaintenanceHandler(nil), alarm: httpAdapter.NewAlarmHandler(nil), ins: httpAdapter.NewInspectionHandler(nil), dash: httpAdapter.NewDashboardHandler(uc), health: httpAdapter.NewHealthHandler(nil), docs: httpAdapter.NewDocsHandler("../../docs/openapi.json"), jwtSecret: secret,
				})
				token, err := security.NewJWTTokenGenerator(secret).Generate(map[string]any{"sub": "workflow-user", "role": role}, time.Hour)
				if err != nil {
					t.Fatal(err)
				}
				req := httptest.NewRequest(http.MethodPost, "/api/v1/predictions/trigger", strings.NewReader(`{}`))
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()
				router.ServeHTTP(response, req)
				if role == "operator" {
					var problem httpAdapter.ProblemDetails
					if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
						t.Fatal(err)
					}
					if response.Code != 403 || problem.Code != "ERR_FORBIDDEN" || ml.calls != 0 || repo.saved != 0 {
						t.Fatalf("operator reached prediction: %d %s", response.Code, response.Body.String())
					}
				} else {
					var run httpAdapter.PredictionRunResponse
					if err := json.Unmarshal(response.Body.Bytes(), &run); err != nil {
						t.Fatal(err)
					}
					if response.Code != 201 || ml.calls != 1 || repo.saved != 1 || run.TotalDoorsEvaluated != 1 || run.TotalDoorsAtRisk != 0 || run.HorizonDays != 7 {
						t.Fatalf("engineer could not execute: %d %s", response.Code, response.Body.String())
					}
				}
			})
		}
	}
}
