package coinex_rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
)

type Client struct {
	httpClient *http_client.Client
	logger     *core_logger.Logger
}

func NewClient(httpClient *http_client.Client, logger *core_logger.Logger, _ core_config.Exchange) *Client {
	return &Client{httpClient: httpClient, logger: logger}
}

func decodeJSON(resp *http.Response, dst any) error {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}
