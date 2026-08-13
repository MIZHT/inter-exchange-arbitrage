package domain

type FilteredMarkets struct {
	FilteredMarkets map[string]FilteredMarket
}

type FilteredMarket struct {
	SpotStatus       MarketStatus
	FutureStatus     MarketStatus
	DepositNetworks  map[string]NetworkInfo
	WithdrawNetworks map[string]NetworkInfo
}

type MarketStatus struct {
	Available       bool
	FundingRate     string
	NextFundingTime int64
}

type Market struct {
	Coin      string
	Spotable  bool
	Futurable bool

	FuturesFundingRate     string
	FuturesNextFundingTime int64
}
