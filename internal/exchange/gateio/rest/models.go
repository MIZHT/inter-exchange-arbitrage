package gateio_rest

type GateIOSpotMarket struct {
	ID          string `json:"id"`
	Base        string `json:"base"`
	Quote       string `json:"quote"`
	TradeStatus string `json:"trade_status"`
}

type GateIOFutureMarket struct {
	Name             string `json:"name"`
	Status           string `json:"status"`
	FundingRate      string `json:"funding_rate"`
	FundingNextApply int64  `json:"funding_next_apply"`
}

type GateIONetwork struct {
	Name             string `json:"name"`
	Address          string `json:"addr"`
	WithdrawDisabled bool   `json:"withdraw_disabled"`
	DepositDisabled  bool   `json:"deposit_disabled"`
	WithdrawFee      string `json:"withdraw_fee"`
}

type GateIOCurrency struct {
	Currency      string          `json:"currency"`
	Delisted      bool            `json:"delisted"`
	TradeDisabled bool            `json:"trade_disabled"`
	Chains        []GateIONetwork `json:"chains"`
}
