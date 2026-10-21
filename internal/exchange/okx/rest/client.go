package okx_rest

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
)

type Client struct {
	httpClient *http_client.Client
	logger     *core_logger.Logger
	apiKey     string
	secretKey  string
	passphrase string
}

func NewClient(httpClient *http_client.Client, logger *core_logger.Logger, cfg core_config.Exchange) *Client {
	return &Client{
		httpClient: httpClient,
		logger:     logger,
		apiKey:     cfg.APIKey,
		secretKey:  cfg.Secret,
		passphrase: cfg.Passphrase,
	}
}

func (c *Client) signedGet(ctx context.Context, endpoint string) (*http.Response, error) {
	if c.apiKey == "" || c.secretKey == "" || c.passphrase == "" {
		return nil, fmt.Errorf("OKX API key, secret and passphrase are required")
	}
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse signed GET endpoint: %w", err)
	}
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	requestPath := parsedURL.RequestURI()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create signed GET request: %w", err)
	}
	req.Header.Set("OK-ACCESS-KEY", c.apiKey)
	req.Header.Set("OK-ACCESS-SIGN", c.sign(timestamp+http.MethodGet+requestPath))
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", c.passphrase)
	req.Header.Set("Accept", "application/json")
	return c.httpClient.Do(ctx, req)
}

func (c *Client) sign(message string) string {
	h := hmac.New(sha256.New, []byte(c.secretKey))
	_, _ = h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
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
