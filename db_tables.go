package fbot

import (
	"encoding/json"

	"gorm.io/gorm"
)

// DB structs
type TableAmms struct {
	gorm.Model
	Pair         string
	BaseReserve  string
	QuoteReserve string
	BlockHeight  int64
	Bias         string
}

type TablePrices struct {
	gorm.Model
	Pair        string
	IndexPrice  string
	MarkPrice   string
	BlockHeight int64
}

type TablePosition struct {
	gorm.Model
	Pair          string
	UnrealizedPnl string
	Size          string
	Trader        string
	BlockHeight   int64
}

type TableBalances struct {
	gorm.Model
	Trader      string
	Denom       string
	Amount      string
	BlockHeight int64
}

func (prices TablePrices) String() string {
	bz, _ := json.Marshal(prices)
	return string(bz)
}

func (position TablePosition) String() string {
	bz, _ := json.Marshal(position)
	return string(bz)
}

func (amms TableAmms) String() string {
	bz, _ := json.Marshal(amms)
	return string(bz)
}

func (balances TableBalances) String() string {
	bz, _ := json.Marshal(balances)
	return string(bz)
}
