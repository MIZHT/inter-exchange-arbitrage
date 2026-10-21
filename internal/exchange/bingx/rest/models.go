package bingx_rest

import "encoding/json"

type BingXSpotMarket struct {
	Symbol string      `json:"symbol"`
	Status json.Number `json:"status"`
}

type BingXSpotMarketResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Symbols []BingXSpotMarket `json:"symbols"`
	} `json:"data"`
}

type BingXFutureMarket struct {
	Symbol          string `json:"symbol"`
	LastFundingRate string `json:"lastFundingRate"`
	NextFundingTime int64  `json:"nextFundingTime"`
}

type BingXFutureMarketResponse struct {
	Code int                 `json:"code"`
	Msg  string              `json:"msg"`
	Data []BingXFutureMarket `json:"data"`
}

type BingXNetwork struct {
	Network         string `json:"network"`
	WithdrawEnable  bool   `json:"withdrawEnable"`
	DepositEnable   bool   `json:"depositEnable"`
	ContractAddress string `json:"contractAddress"`
	WithdrawFee     string `json:"withdrawFee"`
}

type BingXCurrency struct {
	Coin        string         `json:"coin"`
	NetworkList []BingXNetwork `json:"networkList"`
}

type BingXCurrencyResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data []BingXCurrency `json:"data"`
}
