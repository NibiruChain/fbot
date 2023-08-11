package fbot

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Portfolio struct {
	Balances PortfolioBalances

	// BlockNumber: Block number that the portfolio was last updated.
	BlockNumber int64
}

// TotalValue: Coin value of both the wallet and trading positions.
func (portfolio Portfolio) TotalValue() sdk.Coins {
	return portfolio.Balances.TotalValue()
}

type PortfolioBalances struct {
	// TradedBalances: Balances traded in each perp market
	TradedBalances map[string]sdk.Coin

	// WalletCoins: Liquid assets available in the wallet
	WalletCoins sdk.Coins
}

// TotalValue: Coin value of both the wallet and trading positions.
func (bals PortfolioBalances) TotalValue() sdk.Coins {
	coins := sdk.NewCoins(bals.WalletCoins...)
	for _, balance := range bals.TradedBalances {
		coins.Add(balance)
	}
	return coins
}
