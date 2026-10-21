package bitget_rest

type BitgetSpotMarket struct {
	Symbol    string `json:"symbol"`
	BaseCoin  string `json:"baseCoin"`
	QuoteCoin string `json:"quoteCoin"`
	Status    string `json:"status"`
}

type BitgetSpotMarketResponse struct {
	Code string             `json:"code"`
	Msg  string             `json:"msg"`
	Data []BitgetSpotMarket `json:"data"`
}

type BitgetFutureMarket struct {
	Symbol      string `json:"symbol"`
	FundingRate string `json:"fundingRate"`
	NextUpdate  string `json:"nextUpdate"`
}

type BitgetFutureMarketResponse struct {
	Code string               `json:"code"`
	Msg  string               `json:"msg"`
	Data []BitgetFutureMarket `json:"data"`
}

type BitgetNetwork struct {
	Chain           string `json:"chain"`
	Withdrawable    string `json:"withdrawable"`
	Rechargeable    string `json:"rechargeable"`
	WithdrawFee     string `json:"withdrawFee"`
	ContractAddress string `json:"contractAddress"`
}

type BitgetCurrency struct {
	Coin   string          `json:"coin"`
	Chains []BitgetNetwork `json:"chains"`
}

type BitgetCurrencyResponse struct {
	Code string           `json:"code"`
	Msg  string           `json:"msg"`
	Data []BitgetCurrency `json:"data"`
}
