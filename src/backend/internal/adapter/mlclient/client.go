package mlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

var _ port.MLClient = (*Client)(nil)
var _ port.MLOpsClient = (*Client)(nil)

type PredictionRequest struct {
	StationCode string `json:"station_code"`
	Platform    string `json:"platform"`
	DoorNumber  int    `json:"door_number"`
}

type PredictionResponse struct {
	StationCode string   `json:"station_code"`
	Platform    string   `json:"platform"`
	DoorNumber  int      `json:"door_number"`
	Probability *float64 `json:"probability"`
}

type BatchPredictionResponse struct {
	ModelVersion string               `json:"model_version"`
	Predictions  []PredictionResponse `json:"predictions"`
}

type Client struct {
	baseURL        string
	httpClient     *http.Client
	circuitBreaker *CircuitBreaker
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		circuitBreaker: NewCircuitBreaker(3, 10*time.Second),
	}
}

type BatchPredictionRequest struct {
	Doors []PredictionRequest `json:"doors"`
}

func (c *Client) PredictBatch(ctx context.Context, doors []domain.Door) (*port.MLPredictionBatch, error) {
	reqPayload := make([]PredictionRequest, len(doors))
	for i, d := range doors {
		reqPayload[i] = PredictionRequest{
			StationCode: d.StationCode,
			Platform:    d.Platform,
			DoorNumber:  d.DoorNumber,
		}
	}

	payload := BatchPredictionRequest{Doors: reqPayload}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var mlResp BatchPredictionResponse

	err = c.circuitBreaker.Execute(func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+"/v1/predictions", bytes.NewBuffer(bodyBytes))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errors.New("vps retornou codigo nao-ok")
		}

		return json.NewDecoder(resp.Body).Decode(&mlResp)
	})

	if err != nil {
		return nil, domain.ErrPredictionServiceUnavailable
	}
	if mlResp.ModelVersion == "" || len(mlResp.Predictions) != len(doors) {
		return nil, domain.ErrPredictionServiceUnavailable
	}

	items := make([]domain.PredictionItem, len(mlResp.Predictions))
	for i, r := range mlResp.Predictions {
		if r.StationCode != doors[i].StationCode || r.Platform != doors[i].Platform || r.DoorNumber != doors[i].DoorNumber || r.Probability == nil || *r.Probability < 0 || *r.Probability > 1 {
			return nil, domain.ErrPredictionServiceUnavailable
		}
		probability := *r.Probability
		items[i] = domain.PredictionItem{
			StationCode: r.StationCode,
			Platform:    r.Platform,
			DoorNumber:  r.DoorNumber,
			Probability: &probability,
		}
	}
	return &port.MLPredictionBatch{ModelVersion: mlResp.ModelVersion, Predictions: items}, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, input, output any) error {
	var bodyBytes []byte
	var err error
	if input != nil {
		bodyBytes, err = json.Marshal(input)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.baseURL, "/")+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.ErrPredictionServiceUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return domain.ErrPredictionServiceUnavailable
	}
	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		return domain.ErrPredictionServiceUnavailable
	}
	return nil
}

func (c *Client) StartTraining(ctx context.Context, requestedBy string) (*domain.MLTrainingJob, error) {
	var job domain.MLTrainingJob
	if err := c.doJSON(ctx, http.MethodPost, "/v1/training-jobs", map[string]string{"requested_by": requestedBy}, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (c *Client) ResumeTraining(ctx context.Context, id, requestedBy string) (*domain.MLTrainingJob, error) {
	var job domain.MLTrainingJob
	path := "/v1/training-jobs/" + url.PathEscape(id) + "/resume"
	if err := c.doJSON(ctx, http.MethodPost, path, map[string]string{"requested_by": requestedBy}, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (c *Client) CancelTraining(ctx context.Context, id, requestedBy string) (*domain.MLTrainingJob, error) {
	var job domain.MLTrainingJob
	path := "/v1/training-jobs/" + url.PathEscape(id) + "/cancel"
	if err := c.doJSON(ctx, http.MethodPost, path, map[string]string{"requested_by": requestedBy}, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (c *Client) GetActiveModelMetadata(ctx context.Context) (map[string]any, error) {
	manifest := make(map[string]any)
	if err := c.doJSON(ctx, http.MethodGet, "/v1/models/active", nil, &manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (c *Client) GetTrainingJob(ctx context.Context, id string) (*domain.MLTrainingJob, error) {
	var job domain.MLTrainingJob
	if err := c.doJSON(ctx, http.MethodGet, "/v1/training-jobs/"+url.PathEscape(id), nil, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (c *Client) GetDrift(ctx context.Context, version string) (*domain.MLDriftReport, error) {
	var report domain.MLDriftReport
	if err := c.doJSON(ctx, http.MethodGet, "/v1/models/"+url.PathEscape(version)+"/drift", nil, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

func (c *Client) ActivateModel(ctx context.Context, version string) error {
	var manifest map[string]any
	return c.doJSON(ctx, http.MethodPost, "/v1/models/"+url.PathEscape(version)+"/activate", map[string]any{}, &manifest)
}

func (c *Client) CircuitState() State {
	return c.circuitBreaker.State()
}
