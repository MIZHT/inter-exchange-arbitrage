package xt_rest

type XTSpotMarket struct {
	Symbol         string `json:"symbol"`
	State          string `json:"state"`
	TradingEnabled bool   `json:"tradingEnabled"`
	OpenAPIEnabled bool   `json:"openapiEnabled"`
	BaseCurrency   string `json:"baseCurrency"`
	QuoteCurrency  string `json:"quoteCurrency"`
}

type XTSpotMarketResponse struct {
	ReturnCode int    `json:"rc"`
	Message    string `json:"mc"`
	Result     struct {
		Symbols []XTSpotMarket `json:"symbols"`
	} `json:"result"`
}

type XTFutureMarket struct {
	Symbol          string  `json:"symbol"`
	FundingRate     float64 `json:"fundingRate"`
	NextFundingTime int64   `json:"nextCollectionTime"`
}

type XTFutureMarketResponse struct {
	ReturnCode int            `json:"returnCode"`
	Message    string         `json:"msgInfo"`
	Result     XTFutureMarket `json:"result"`
}

type XTNetwork struct {
	Chain           string  `json:"chain"`
	ContractAddress string  `json:"contract"`
	DepositEnabled  bool    `json:"depositEnabled"`
	WithdrawEnabled bool    `json:"withdrawEnabled"`
	WithdrawFee     float64 `json:"withdrawFeeAmount"`
}

type XTNetworkCurrency struct {
	Currency      string      `json:"currency"`
	SupportChains []XTNetwork `json:"supportChains"`
}

type XTNetworkResponse struct {
	ReturnCode int                 `json:"rc"`
	Message    string              `json:"mc"`
	Result     []XTNetworkCurrency `json:"result"`
}

type XTCurrency struct {
	Currency       string `json:"currency"`
	IsListing      int    `json:"isListing"`
	DepositStatus  int    `json:"depositStatus"`
	WithdrawStatus int    `json:"withdrawStatus"`
}

type XTCurrencyResponse struct {
	ReturnCode int    `json:"rc"`
	Message    string `json:"mc"`
	Result     struct {
		Currencies []XTCurrency `json:"currencies"`
	} `json:"result"`
}
