package binance_rest

// Spot symbols
type BinanceSpotMarket struct {
	Symbol               string `json:"symbol"`
	Status               string `json:"status"`
	BaseAsset            string `json:"baseAsset"`
	QuoteAsset           string `json:"quoteAsset"`
	IsSpotTradingAllowed bool   `json:"isSpotTradingAllowed"`
}

// Spot exchangeInfo response
type BinanceExchangeInfo struct {
	Symbols []BinanceSpotMarket `json:"symbols"`
}

// Futures premium index (funding)
type BinanceFutureMarket struct {
	Symbol          string `json:"symbol"`
	LastFundingRate string `json:"lastFundingRate"`
	NextFundingTime int64  `json:"nextFundingTime"`
}

// Asset config response (currencies)
type BinanceCurrencies []BinanceCurrency

type BinanceCurrency struct {
	Coin        string           `json:"coin"`
	Name        string           `json:"name,omitempty"`
	NetworkList []BinanceNetwork `json:"networkList"`
}

type BinanceNetwork struct {
	Network         string `json:"network"`
	Name            string `json:"name"`
	WithdrawEnable  bool   `json:"withdrawEnable"`
	DepositEnable   bool   `json:"depositEnable"`
	WithdrawFee     string `json:"withdrawFee"`
	ContractAddress string `json:"contractAddress,omitempty"`
	IsDefault       bool   `json:"isDefault,omitempty"`
}
