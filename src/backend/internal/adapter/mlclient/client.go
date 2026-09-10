package mlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

var _ port.MLClient = (*Client)(nil)

type PredictionRequest struct {
	StationCode string `json:"station_code"`
	Platform    string `json:"platform"`
	DoorNumber  int    `json:"door_number"`
}

type PredictionResponse struct {
	StationCode string `json:"station_code"`
	Platform    string `json:"platform"`
	DoorNumber  int    `json:"door_number"`
	WillFail    bool   `json:"will_fail"`
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

func (c *Client) PredictBatch(ctx context.Context, doors []domain.Door) ([]domain.PredictionItem, error) {
	reqPayload := make([]PredictionRequest, len(doors))
	for i, d := range doors {
		reqPayload[i] = PredictionRequest{
			StationCode: d.StationCode,
			Platform:    d.Platform,
			DoorNumber:  d.DoorNumber,
		}
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	var mlResp []PredictionResponse

	err = c.circuitBreaker.Execute(func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/predict", bytes.NewBuffer(bodyBytes))
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

	items := make([]domain.PredictionItem, len(mlResp))
	for i, r := range mlResp {
		items[i] = domain.PredictionItem{
			StationCode: r.StationCode,
			Platform:    r.Platform,
			DoorNumber:  r.DoorNumber,
			WillFail:    r.WillFail,
		}
	}
	return items, nil
}

func (c *Client) CircuitState() State {
	return c.circuitBreaker.State()
}
