package kucoin_rest

type KuCoinSpotMarket struct {
	Symbol        string `json:"symbol"`
	BaseCurrency  string `json:"baseCurrency"`
	QuoteCurrency string `json:"quoteCurrency"`
	EnableTrading bool   `json:"enableTrading"`
}

type KuCoinSpotMarketResponse struct {
	Code string             `json:"code"`
	Data []KuCoinSpotMarket `json:"data"`
}

type KuCoinFutureMarket struct {
	Symbol                  string  `json:"symbol"`
	Status                  string  `json:"status"`
	BaseCurrency            string  `json:"baseCurrency"`
	QuoteCurrency           string  `json:"quoteCurrency"`
	SettleCurrency          string  `json:"settleCurrency"`
	FundingFeeRate          float64 `json:"fundingFeeRate"`
	NextFundingRateDateTime int64   `json:"nextFundingRateDateTime"`
}

type KuCoinFutureMarketResponse struct {
	Code string               `json:"code"`
	Data []KuCoinFutureMarket `json:"data"`
}

type KuCoinNetwork struct {
	ContractAddress   string `json:"contractAddress"`
	IsDepositEnabled  bool   `json:"isDepositEnabled"`
	IsWithdrawEnabled bool   `json:"isWithdrawEnabled"`
	WithdrawalMinFee  string `json:"withdrawalMinFee"`
}

type KuCoinCurrency struct {
	Currency string `json:"currency"`
	KuCoinNetwork
}

type KuCoinCurrencyResponse struct {
	Code string           `json:"code"`
	Data []KuCoinCurrency `json:"data"`
}
