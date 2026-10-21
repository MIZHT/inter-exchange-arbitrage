package bybit_rest

type BybitSpotMarket struct {
	Symbol string `json:"symbol"`
	Status string `json:"status"`
}

type BybitSpotMarketResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []BybitSpotMarket `json:"list"`
	} `json:"result"`
}

type BybitFutureMarket struct {
	Symbol          string `json:"symbol"`
	FundingRate     string `json:"fundingRate"`
	NextFundingTime string `json:"nextFundingTime"`
}

type BybitFutureMarketResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []BybitFutureMarket `json:"list"`
	} `json:"result"`
}

type BybitNetwork struct {
	Chain            string `json:"chain"`
	WithdrawFee      string `json:"withdrawFee"`
	WithdrawDisabled bool   `json:"withdrawDisabled"`
	DepositDisabled  bool   `json:"depositDisabled"`
	ContractAddress  string `json:"contractAddress"`
}

type BybitCurrency struct {
	Coin   string         `json:"coin"`
	Chains []BybitNetwork `json:"chains"`
}

type BybitCurrencyResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Rows []BybitCurrency `json:"rows"`
	} `json:"result"`
}
