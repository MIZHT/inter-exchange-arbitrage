package bitmart_rest

type BitmartSpotMarket struct {
	Symbol        string `json:"symbol"`
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	TradeStatus   string `json:"trade_status"`
}

type BitmartSpotMarketResponse struct {
	Code    any                   `json:"code"`
	Trace   string                `json:"trace"`
	Message string                `json:"message"`
	Data    BitmartSpotMarketData `json:"data"`
}

type BitmartSpotMarketData struct {
	Symbols []BitmartSpotMarket `json:"symbols"`
}

type BitmartFutureMarket struct {
	Symbol      string `json:"symbol"`
	FundingRate string `json:"funding_rate"`
	FundingTime int64  `json:"funding_time"`
	Status      string `json:"status"`
}

type BitmartFutureMarketResponse struct {
	Code    any                     `json:"code"`
	Trace   string                  `json:"trace"`
	Message string                  `json:"message"`
	Data    BitmartFutureMarketData `json:"data"`
}

type BitmartFutureMarketData struct {
	Symbols []BitmartFutureMarket `json:"symbols"`
}

type BitmartNetwork struct {
	Network         string  `json:"network"`
	WithdrawEnabled bool    `json:"withdraw_enabled"`
	DepositEnabled  bool    `json:"deposit_enabled"`
	WithdrawFee     string  `json:"withdraw_fee"`
	ContractAddress *string `json:"contract_address"`
}

type BitmartCurrency struct {
	Currency string `json:"currency"`
	BitmartNetwork
}

type BitmartCurrencyResponse struct {
	Code    any                 `json:"code"`
	Trace   string              `json:"trace"`
	Message string              `json:"message"`
	Data    BitmartCurrencyData `json:"data"`
}

type BitmartCurrencyData struct {
	Currencies []BitmartCurrency `json:"currencies"`
}
