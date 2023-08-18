package fbot

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"google.golang.org/grpc"
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
func (bals *PortfolioBalances) TotalValue() sdk.Coins {
	coins := sdk.NewCoins(bals.WalletCoins...)
	for _, balance := range bals.TradedBalances {
		coins.Add(balance)
	}
	return coins
}

func InitializePortfolio() *Portfolio {
	bals := &Portfolio{
		Balances: PortfolioBalances{
			TradedBalances: make(map[string]sdk.Coin),
			WalletCoins:    []sdk.Coin{},
		},
		BlockNumber: 0,
	}
	return bals
}

func (bals *PortfolioBalances) QueryWalletCoins(ctx context.Context,
	trader sdk.AccAddress, grpcCon *grpc.ClientConn) (*bankTypes.QueryAllBalancesResponse, error) {
	balQueryClient := bankTypes.NewQueryClient(grpcCon)

	return balQueryClient.AllBalances(
		ctx, &bankTypes.QueryAllBalancesRequest{
			Address: trader.String(),
		},
	)
}

func (bals *PortfolioBalances) PopWalletCoins(balanceResp *bankTypes.QueryAllBalancesResponse) {

	for _, coin := range balanceResp.Balances {
		bals.WalletCoins = bals.WalletCoins.Add(coin)
	}
}

func (bals *PortfolioBalances) AddTradedBalances(market string, amount sdk.Coin) {
	if bals.TradedBalances[market].Denom == amount.Denom {
		bals.TradedBalances[market] = bals.TradedBalances[market].Add(amount)
	} else {
		bals.TradedBalances[market] = amount
	}
}

func (bals *PortfolioBalances) RemoveTradedBalances(market string, amount sdk.Coin) {
	if bals.TradedBalances[market].Denom == amount.Denom {
		bals.TradedBalances[market] = bals.TradedBalances[market].Sub(amount)
	}
	bals.WalletCoins = append(bals.WalletCoins, amount)
}

// func (bals PortfolioBalances) CalcRatioBalances() {
// 	for pair, tradedCoin := range bals.TradedBalances {
// 		fmt.Println("Traded Coin: ", tradedCoin)
// 		for _, walCoin := range bals.WalletCoins {
// 			fmt.Println("Wallet Coin: ", walCoin)
// 			if walCoin.Denom == tradedCoin.Denom {

// 				bals.Ratios[pair] = sdk.Dec(tradedCoin.Amount).Quo(
// 					sdk.NewDecFromInt(walCoin.Amount),
// 				)
// 			}
// 		}
// 	}
// }
