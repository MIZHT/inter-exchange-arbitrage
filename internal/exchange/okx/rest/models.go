package okx_rest

type OKXSpotMarket struct {
	InstID   string `json:"instId"`
	BaseCcy  string `json:"baseCcy"`
	QuoteCcy string `json:"quoteCcy"`
	State    string `json:"state"`
}

type OKXSpotMarketResponse struct {
	Code    string          `json:"code"`
	Message string          `json:"msg"`
	Data    []OKXSpotMarket `json:"data"`
}

type OKXFutureMarket struct {
	InstID          string `json:"instId"`
	SettleCcy       string `json:"settleCcy"`
	State           string `json:"state"`
	FundingRate     string `json:"fundingRate"`
	NextFundingTime string `json:"nextFundingTime"`
}

type OKXFutureMarketResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"msg"`
	Data    []OKXFutureMarket `json:"data"`
}

type OKXNetwork struct {
	Chain           string `json:"chain"`
	ContractAddress string `json:"ctAddr"`
	DepositEnabled  bool   `json:"canDep"`
	WithdrawEnabled bool   `json:"canWd"`
	WithdrawFee     string `json:"fee"`
}

type OKXCurrency struct {
	Currency string `json:"ccy"`
	OKXNetwork
}

type OKXCurrencyResponse struct {
	Code string        `json:"code"`
	Data []OKXCurrency `json:"data"`
}
