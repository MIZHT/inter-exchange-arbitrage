package http_client

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// client := httpclient.NewClient(
//     httpclient.WithTimeout(10*time.Second),
//     httpclient.WithProxy(cfg.Proxy),
//     httpclient.WithHeader("User-Agent", "MyParser/1.0"),
//     httpclient.WithHeader("Content-Type", "application/json"),
//     httpclient.WithRetry(3, time.Second),
//     httpclient.WithLogger(logger),
// )
//resp, err := client.Get(ctx, "https://api.bybit.com/v5/market/instruments-info?category=spot")

type Client struct {
	httpClient *http.Client
	logger     *slog.Logger
	headers    map[string]string
	maxRetries int
	retryDelay time.Duration
}

type Option func(*Client)

func NewHTTPClient(opts ...Option) *Client {
	transport := &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	c := &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
		maxRetries: 3,
		retryDelay: time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func WithLogger(l *slog.Logger) Option {
	return func(c *Client) {
		c.logger = l
	}
}

func WithProxy(proxyURL string) Option {
	return func(c *Client) {

		if proxyURL == "" {
			return
		}
		proxyAddr, err := url.Parse(proxyURL)
		if err != nil {
			c.logger.Warn("Invalid proxy URL", slog.String("proxy", proxyURL), slog.Any("error", err))
		} else {
			transport := c.httpClient.Transport.(*http.Transport)
			transport.Proxy = http.ProxyURL(proxyAddr)
			c.logger.Debug("Using proxy for HTTP client", slog.String("proxy", proxyURL))
		}
	}
}

func WithHeader(key, value string) Option {
	return func(c *Client) {
		if c.headers == nil {
			c.headers = make(map[string]string)
		}
		c.headers[key] = value
	}
}

// func WithRetry(maxRetries int, baseDelay time.Duration) Option
// func WithHeader(key, value stirng) Option

func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	for key, val := range c.headers {
		if req.Header.Get(key) == "" {
			req.Header.Set(key, val)
		}
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		delay := c.retryDelay * time.Duration(1<<uint(attempt+2))
		delay += time.Duration(time.Now().UnixNano() % int64(delay/2))
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}

		resp, err = c.httpClient.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		if c.logger != nil {
			c.logger.Warn("HTTP request failed",
				slog.Int("attempt", attempt+1),
				slog.String("url", req.URL.String()),
				slog.Any("error", err),
				slog.Int("status", statusCode(resp)))
		}

		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		//в случае attemp = 0
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

func statusCode(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}
