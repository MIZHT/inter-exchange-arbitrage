package domain

type Currency struct {
	Name     string
	Networks []NetworkInfo
}

type NetworkInfo struct {
	Name            string
	ContractAddress string
	DepositEnabled  bool
	WithdrawEnabled bool
	WithdrawFee     float64
}
