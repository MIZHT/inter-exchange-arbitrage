package binance_rest

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
)

const (
	currenciesEndpoint     = "https://api.binance.com/sapi/v1/capital/config/getall"
	spotMarketsEndpoint    = "https://api.binance.com/api/v3/exchangeInfo"
	futuresMarketsEndpoint = "https://fapi.binance.com/fapi/v1/premiumIndex"
)

type Client struct {
	httpClient *http_client.Client
	logger     *core_logger.Logger
	apiKey     string
	secretKey  string
}

func NewClient(
	httpClient *http_client.Client,
	logger *core_logger.Logger,
	cfg core_config.Exchange,
) *Client {
	return &Client{
		httpClient: httpClient,
		logger:     logger,
		apiKey:     cfg.APIKey,
		secretKey:  cfg.Secret,
	}
}

func (c *Client) signedGet(ctx context.Context, endpoint string, params url.Values) (*http.Response, error) {
	if c.apiKey == "" || c.secretKey == "" {
		return nil, fmt.Errorf("api key and secret are required for signed Binance request")
	}

	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")

	queryToSign := params.Encode()
	params.Set("signature", c.sign(queryToSign))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create signed GET request: %w", err)
	}
	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	return c.httpClient.Do(ctx, req)
}

func (c *Client) sign(query string) string {
	h := hmac.New(sha256.New, []byte(c.secretKey))
	h.Write([]byte(query))
	return hex.EncodeToString(h.Sum(nil))
}

func decodeJSON(resp *http.Response, dst any) error {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}
