package adapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/g04/backend/internal/adapter/mlclient"
	"github.com/g04/backend/internal/domain"
)

func TestMLClient_PredictBatch_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/predict" {
			http.NotFound(w, r)
			return
		}
		var reqBody mlclient.BatchPredictionRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resps := []mlclient.PredictionResponse{
			{
				StationCode: reqBody.Doors[0].StationCode,
				Platform:    reqBody.Doors[0].Platform,
				DoorNumber:  reqBody.Doors[0].DoorNumber,
				WillFail:    true,
				Probability: 0.72,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resps)
	}))
	defer ts.Close()

	client := mlclient.NewClient(ts.URL)
	doors := []domain.Door{
		{StationCode: "FAL", Platform: "P1", DoorNumber: 4},
	}

	items, err := client.PredictBatch(context.Background(), doors, nil)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(items) != 1 || !items[0].WillFail {
		t.Fatalf("unexpected items: %+v", items)
	}
	if items[0].Probability == nil || *items[0].Probability != 0.72 {
		t.Fatalf("expected probability to be preserved, got %+v", items[0])
	}
	if client.CircuitState() != mlclient.StateClosed {
		t.Fatalf("expected closed circuit state")
	}
}

func TestMLClient_PredictBatch_NonOKStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := mlclient.NewClient(ts.URL)
	doors := []domain.Door{
		{StationCode: "PIN", Platform: "P2", DoorNumber: 8},
	}

	_, err := client.PredictBatch(context.Background(), doors, nil)
	if err != domain.ErrPredictionServiceUnavailable {
		t.Fatalf("expected ErrPredictionServiceUnavailable, got %v", err)
	}
}

func TestMLClient_PredictBatch_BadJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer ts.Close()

	client := mlclient.NewClient(ts.URL)
	doors := []domain.Door{
		{StationCode: "PIN", Platform: "P2", DoorNumber: 8},
	}

	_, err := client.PredictBatch(context.Background(), doors, nil)
	if err != domain.ErrPredictionServiceUnavailable {
		t.Fatalf("expected ErrPredictionServiceUnavailable on bad json, got %v", err)
	}
}

func TestMLClient_PredictBatch_NetworkError(t *testing.T) {
	client := mlclient.NewClient("http://127.0.0.1:59999")
	doors := []domain.Door{
		{StationCode: "PIN", Platform: "P2", DoorNumber: 8},
	}

	_, err := client.PredictBatch(context.Background(), doors, nil)
	if err != domain.ErrPredictionServiceUnavailable {
		t.Fatalf("expected ErrPredictionServiceUnavailable on network error, got %v", err)
	}
}

func TestMLClient_PredictBatch_InvalidURL(t *testing.T) {
	client := mlclient.NewClient("://invalid-url\n")
	doors := []domain.Door{
		{StationCode: "PIN", Platform: "P2", DoorNumber: 8},
	}

	_, err := client.PredictBatch(context.Background(), doors, nil)
	if err != domain.ErrPredictionServiceUnavailable {
		t.Fatalf("expected ErrPredictionServiceUnavailable on invalid url, got %v", err)
	}
}
