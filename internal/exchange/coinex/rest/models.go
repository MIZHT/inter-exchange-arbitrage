package coinex_rest

type CoinExSpotMarket struct {
	BaseCcy               string `json:"base_ccy"`
	IsAPITradingAvailable bool   `json:"is_api_trading_available"`
	Market                string `json:"market"`
	QuoteCcy              string `json:"quote_ccy"`
	Status                string `json:"status"`
}

type CoinExSpotMarketResponse struct {
	Code    int                `json:"code"`
	Message string             `json:"message"`
	Data    []CoinExSpotMarket `json:"data"`
}

type CoinExFutureMarket struct {
	FundingTime     int64  `json:"funding_time"`
	FundingRateNext string `json:"funding_rate_next"`
}

type CoinExFutureMarketResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Ticker map[string]CoinExFutureMarket `json:"ticker"`
	} `json:"data"`
}

type CoinExNetwork struct {
	Chain            string `json:"chain"`
	ExplorerAssetURL string `json:"explorer_asset_url"`
	DepositEnabled   bool   `json:"deposit_enabled"`
	WithdrawEnabled  bool   `json:"withdraw_enabled"`
	WithdrawFee      string `json:"withdraw_fee"`
}

type CoinExAsset struct {
	Ccy string `json:"ccy"`
}

type CoinExCurrency struct {
	Asset  CoinExAsset     `json:"asset"`
	Chains []CoinExNetwork `json:"chains"`
}

type CoinExCurrencyResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    []CoinExCurrency `json:"data"`
}
