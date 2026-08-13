package domain

type Currency struct {
	Name            string
	DepositEnabled  bool
	WithdrawEnabled bool
	Networks        []NetworkInfo
}

type NetworkInfo struct {
	Name            string
	ContractAddress string
	DepositEnabled  bool
	WithdrawEnabled bool
	WithdrawFee     string
}
