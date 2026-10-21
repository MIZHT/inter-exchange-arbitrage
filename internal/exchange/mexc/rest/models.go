package mexc_rest

import "encoding/json"

type MEXCSpotMarket struct {
	Symbol               string      `json:"symbol"`
	Status               json.Number `json:"status"`
	BaseAsset            string      `json:"baseAsset"`
	QuoteAsset           string      `json:"quoteAsset"`
	IsSpotTradingAllowed bool        `json:"isSpotTradingAllowed"`
}

type MEXCSpotMarketResponse struct {
	Symbols []MEXCSpotMarket `json:"symbols"`
}

type MEXCFutureMarket struct {
	Symbol         string  `json:"symbol"`
	FundingRate    float64 `json:"fundingRate"`
	NextSettleTime int64   `json:"nextSettleTime"`
}

type MEXCFutureMarketResponse struct {
	Success bool               `json:"success"`
	Code    int                `json:"code"`
	Data    []MEXCFutureMarket `json:"data"`
}

type MEXCNetwork struct {
	Network         string `json:"network"`
	WithdrawEnable  bool   `json:"withdrawEnable"`
	DepositEnable   bool   `json:"depositEnable"`
	ContractAddress string `json:"contract"`
	WithdrawFee     string `json:"withdrawFee"`
}

type MEXCCurrency struct {
	Coin        string        `json:"coin"`
	NetworkList []MEXCNetwork `json:"networkList"`
}
