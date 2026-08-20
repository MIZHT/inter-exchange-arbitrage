package domain

type FilteredPairs map[string]FilteredPair

type FilteredPair struct {
	SpotStatus       MarketStatus
	FutureStatus     MarketStatus
	DepositNetworks  []NetworkInfo
	WithdrawNetworks []NetworkInfo
}

type MarketStatus struct {
	Available       bool
	FundingRate     float64
	NextFundingTime int64
}

type Market struct {
	Coin      string
	Spotable  bool
	Futurable bool

	FuturesFundingRate     float64
	FuturesNextFundingTime int64
}
