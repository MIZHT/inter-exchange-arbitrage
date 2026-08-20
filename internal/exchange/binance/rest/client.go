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
	"sync"
	"time"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
	"go.uber.org/zap"
)

const (
	currenciesEndpoint     = "https://api.binance.com/sapi/v1/capital/config/getall"
	spotMarketsEndpoint    = "https://api.binance.com/api/v3/exchangeInfo"
	futuresPremiumEndpoint = "https://fapi.binance.com/fapi/v1/premiumIndex"
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

func (c *Client) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	resp, err := c.signedGet(ctx, currenciesEndpoint, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("request asset configs: %w", err)
	}
	defer resp.Body.Close()

	var assets BinanceAssetConfigsResponse
	if err := decodeJSON(resp, &assets); err != nil {
		return nil, fmt.Errorf("decode asset configs: %w", err)
	}

	currencies, err := mapCurrencies(assets)
	if err != nil {
		return nil, fmt.Errorf("map asset configs: %w", err)
	}

	c.logger.Info("Binance FetchCurrencies", zap.Int("count", len(currencies)))
	return currencies, nil
}

func (c *Client) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	var (
		wg       sync.WaitGroup
		info     BinanceExchangeInfo
		premiums []BinanceFuturesPremiumIndex
		spotErr  error
		futErr   error
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		info, spotErr = c.fetchSpotMarket(ctx)
	}()
	go func() {
		defer wg.Done()
		premiums, futErr = c.fetchFuturesMarket(ctx)
	}()
	wg.Wait()

	if spotErr != nil {
		return nil, fmt.Errorf("fetch spot exchange info: %w", spotErr)
	}
	if futErr != nil {
		return nil, fmt.Errorf("fetch futures premium index: %w", futErr)
	}

	markets, err := mapMarkets(info, premiums)
	if err != nil {
		return nil, fmt.Errorf("map markets: %w", err)
	}

	c.logger.Info("Binance FetchMarkets", zap.Int("count", len(markets)))
	return markets, nil
}

func (c *Client) fetchSpotMarket(ctx context.Context) (BinanceExchangeInfo, error) {
	resp, err := c.httpClient.Get(ctx, spotMarketsEndpoint)
	if err != nil {
		return BinanceExchangeInfo{}, err
	}
	defer resp.Body.Close()

	var info BinanceExchangeInfo
	if err := decodeJSON(resp, &info); err != nil {
		return BinanceExchangeInfo{}, fmt.Errorf("decode spot exchange info: %w", err)
	}
	return info, nil
}

func (c *Client) fetchFuturesMarket(ctx context.Context) ([]BinanceFuturesPremiumIndex, error) {
	resp, err := c.httpClient.Get(ctx, futuresPremiumEndpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var premiums []BinanceFuturesPremiumIndex
	if err := decodeJSON(resp, &premiums); err != nil {
		return nil, fmt.Errorf("decode futures premium index: %w", err)
	}
	return premiums, nil
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

func mapCurrencies(assets BinanceAssetConfigsResponse) ([]domain.Currency, error) {
	currencies := make([]domain.Currency, 0, len(assets))
	for _, asset := range assets {
		cur := domain.Currency{Name: strings.ToUpper(asset.Coin)}

		for _, net := range asset.NetworkList {
			if net.ContractAddress == "" {
				continue
			}
			if !net.DepositEnable || !net.WithdrawEnable {
				continue
			}

			fee, err := strconv.ParseFloat(net.WithdrawFee, 64)
			if err != nil {
				return nil, fmt.Errorf("parse withdraw fee for %s %s: %w", asset.Coin, net.Network, err)
			}

			cur.Networks = append(cur.Networks, domain.NetworkInfo{
				Name:            net.Network,
				ContractAddress: strings.ToLower(net.ContractAddress),
				DepositEnabled:  net.DepositEnable,
				WithdrawEnabled: net.WithdrawEnable,
				WithdrawFee:     fee,
			})
		}

		if len(cur.Networks) == 0 {
			continue
		}
		currencies = append(currencies, cur)
	}
	return currencies, nil
}

func mapMarkets(
	info BinanceExchangeInfo,
	premiums []BinanceFuturesPremiumIndex,
) ([]domain.Market, error) {
	marketsByCoin := make(map[string]*domain.Market)

	for _, s := range info.Symbols {
		if s.QuoteAsset != "USDT" {
			continue
		}
		if s.Status != "TRADING" || !s.IsSpotTradingAllowed {
			continue
		}

		coin := strings.ToUpper(s.BaseAsset)
		getOrCreateMarket(marketsByCoin, coin).Spotable = true
	}

	for _, p := range premiums {
		if p.NextFundingTime <= 0 {
			continue
		}
		if !strings.HasSuffix(p.Symbol, "USDT") {
			continue
		}

		rate, err := strconv.ParseFloat(p.LastFundingRate, 64)
		if err != nil {
			return nil, fmt.Errorf("parse funding rate for %s: %w", p.Symbol, err)
		}

		coin := strings.TrimSuffix(p.Symbol, "USDT")
		market := getOrCreateMarket(marketsByCoin, coin)
		market.Futurable = true
		market.FuturesFundingRate = rate
		market.FuturesNextFundingTime = p.NextFundingTime
	}

	markets := make([]domain.Market, 0, len(marketsByCoin))
	for _, market := range marketsByCoin {
		if !market.Spotable && !market.Futurable {
			continue
		}
		markets = append(markets, *market)
	}
	return markets, nil
}

func getOrCreateMarket(markets map[string]*domain.Market, coin string) *domain.Market {
	market := markets[coin]
	if market == nil {
		market = &domain.Market{Coin: coin}
		markets[coin] = market
	}
	return market
}
