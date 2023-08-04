package fbot

import (
	"encoding/json"

	"cosmossdk.io/math"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type BotDB struct {
	DB *gorm.DB
}

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
	Denom       string
	Amount      string
	BlockHeight int64
}

func CreateAndConnectDB() BotDB {
	botDB := new(BotDB)

	botDB.ConnectToDB()
	return *botDB
}

func (botdb *BotDB) ConnectToDB() {
	db, err := gorm.Open(sqlite.Open("bot.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	botdb.DB = db

	botdb.DB.AutoMigrate(&TablePrices{})
	botdb.DB.AutoMigrate(&TableAmms{})
	botdb.DB.AutoMigrate(&TableAmms{})
	botdb.DB.AutoMigrate(&TableBalances{})
}

func (botdb *BotDB) ClearDB() {

	// botdb.DB.Delete(&TablePosition{})
	botdb.DB.Where("pair IS NOT NULL").Delete(&TablePrices{})
	botdb.DB.Where("pair IS NOT NULL").Delete(&TableAmms{})
	botdb.DB.Where("pair IS NOT NULL").Delete(&TableBalances{})
	botdb.DB.Where("pair IS NOT NULL").Delete(&TablePosition{})
}

// func (botdb *BotDB) JoinTables() {
// 	var joinedData []JoinedData

// 	botdb.DB.Table("table_prices_amms").
// 		Select("table_prices_amms.*, table_position.*, table_balances.*").
// 		Joins("JOIN table_position ON table_prices_amms.block_height = table_position.block_height").
// 		Joins("JOIN table_balances ON table_prices_amms.block_height = table_balances.block_height").
// 		Scan(&joinedData)

// }

// Populating Tables

func (botdb *BotDB) PopulatePricesTable(prices map[string]Prices, blockHeight int64) {
	for pair, priceField := range prices {
		botdb.DB.Create(&TablePrices{
			Pair: pair, IndexPrice: priceField.IndexPrice.String(),
			MarkPrice:   priceField.MarkPrice.String(),
			BlockHeight: blockHeight,
		})
	}
}

func (botdb *BotDB) PopulateAmmsTable(amm map[string]AmmFields, blockHeight int64) {
	for pair, ammField := range amm {
		botdb.DB.Create(&TableAmms{
			Pair:         pair,
			BaseReserve:  ammField.Markets.BaseReserve.String(),
			QuoteReserve: ammField.Markets.QuoteReserve.String(),
			BlockHeight:  blockHeight,
			Bias:         ammField.Bias.String(),
		})
	}
}

func (botdb *BotDB) PopulatePositionTable(positions map[string]PositionFields, blockHeight int64) {
	for pair, posField := range positions {
		botdb.DB.Create(&TablePosition{
			Pair:          pair,
			UnrealizedPnl: posField.UnrealizedPnl.String(),
			Size:          posField.Positon.Size_.String(),
			Trader:        posField.Positon.TraderAddress,
			BlockHeight:   blockHeight,
		})
	}
}

func (botdb *BotDB) PopulateBalancesTable(balances map[string]math.Int, blockHeight int64) {
	for denom, balance := range balances {
		botdb.DB.Create(&TableBalances{
			Denom:       denom,
			Amount:      balance.String(),
			BlockHeight: blockHeight,
		})
	}
}

// Querying Prices

func (botdb *BotDB) QueryPricesByBlock(blockHeight int64) ([]TablePrices, error) {
	var prices []TablePrices
	db := botdb.DB.Find(&prices, "block_height = ?", blockHeight)
	return prices, db.Error
}

func (botdb *BotDB) QueryPricesTable() ([]TablePrices, error) {
	var allPrices []TablePrices
	db := botdb.DB.Find(&allPrices)

	return allPrices, db.Error
}

// Querying Positions

func (botdb *BotDB) QueryPositionByBlock(blockHeight int64) ([]TablePosition, error) {
	var positions []TablePosition
	db := botdb.DB.Find(&positions, "block_height = ?", blockHeight)
	return positions, db.Error
}

func (botdb *BotDB) QueryPositionTable() ([]TablePosition, error) {
	var allPositions []TablePosition
	db := botdb.DB.Find(&allPositions)
	return allPositions, db.Error
}

// Querying Amms

func (botdb *BotDB) QueryAmmByBlock(blockHeight int64) ([]TableAmms, error) {
	var amms []TableAmms
	db := botdb.DB.Find(&amms, "block_height = ?", blockHeight)

	return amms, db.Error
}

func (botdb *BotDB) QueryAmmTable() ([]TableAmms, error) {
	var allAmms []TableAmms
	db := botdb.DB.Find(&allAmms)

	return allAmms, db.Error
}

// Querying Balances

func (botdb *BotDB) QueryBalancesByBlock(blockHeight int64) ([]TableBalances, error) {
	var balances []TableBalances
	db := botdb.DB.Find(&balances, "block_height = ?", blockHeight)

	return balances, db.Error
}

func (botdb *BotDB) QueryBalancesTable() ([]TableBalances, error) {
	var allBalances []TableBalances
	db := botdb.DB.Find(&allBalances)

	return allBalances, db.Error
}

type DBRecords struct {
	PositionRecords []TablePosition `json:"positions"`
	AmmRecords      []TableAmms     `json:"amms"`
	BalanceRecords  []TableBalances `json:"balances"`
	PriceRecords    []TablePrices   `json:"prices"`
}

func NewDBRecordsFromString(recordsJson string) (DBRecords, error) {
	var dbRecords DBRecords
	err := json.Unmarshal([]byte(recordsJson), &dbRecords)

	records := DBRecords{
		PositionRecords: dbRecords.PositionRecords,
		AmmRecords:      dbRecords.AmmRecords,
		BalanceRecords:  dbRecords.BalanceRecords,
		PriceRecords:    dbRecords.PriceRecords,
	}
	return records, err
}

func DBRecordsToString(positions []TablePosition, amms []TableAmms, balances []TableBalances, prices []TablePrices) string {
	dbRecords := DBRecords{
		PositionRecords: positions,
		AmmRecords:      amms,
		BalanceRecords:  balances,
		PriceRecords:    prices,
	}
	return dbRecords.String()
}

func (botdb *BotDB) ImportDbRecords(records DBRecords) {
	for _, positions := range records.PositionRecords {
		botdb.DB.Create(&positions)
	}

	for _, prices := range records.PriceRecords {
		botdb.DB.Create(&prices)
	}
	for _, amms := range records.AmmRecords {
		botdb.DB.Create(&amms)
	}
	for _, balances := range records.BalanceRecords {
		botdb.DB.Create(&balances)
	}
}

// json golang struct tags

// Create a dbrecord to test funcs

func (records *DBRecords) String() string {

	bz, _ := json.Marshal(records)

	return string(bz)
}
