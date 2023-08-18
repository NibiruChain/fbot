package fbot

import (
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type BotDB struct {
	DB   *gorm.DB
	Name string
}

func CreateAndConnectDB(dbName string) BotDB {
	botDB := new(BotDB)

	botDB.ConnectToDB(dbName)
	return *botDB
}

func (botdb *BotDB) ConnectToDB(dbName string) {
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	botdb.DB = db
	botdb.Name = dbName

	botdb.DB.AutoMigrate(&TablePrices{})
	botdb.DB.AutoMigrate(&TableAmms{})
	botdb.DB.AutoMigrate(&TablePosition{})
	botdb.DB.AutoMigrate(&TableBalances{})
}

func (botdb *BotDB) ClearDB() {
	botdb.DB.Where("pair IS NOT NULL").Delete(&TablePrices{})
	botdb.DB.Where("pair IS NOT NULL").Delete(&TableAmms{})
	botdb.DB.Where("pair IS NOT NULL").Delete(&TableBalances{})
	botdb.DB.Where("pair IS NOT NULL").Delete(&TablePosition{})
}

func (botdb *BotDB) DeleteDB() {
	os.Remove(botdb.Name)
}

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

func (botdb *BotDB) PopulateBalancesTable(wallet sdk.Coins, trader string, blockHeight int64) {
	for _, coin := range wallet {
		botdb.DB.Create(&TableBalances{
			Trader:      trader,
			Denom:       coin.Denom,
			Amount:      coin.Amount.String(),
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

// Querying All
func (botdb *BotDB) QueryAllTablesToJson() (string, []error) {
	var errors []error
	amms, ammErr := botdb.QueryAmmTable()
	prices, pricesErr := botdb.QueryPricesTable()
	balances, balErr := botdb.QueryBalancesTable()
	positions, posErr := botdb.QueryPositionTable()

	errors = append(errors, ammErr, pricesErr, balErr, posErr)

	return DBRecordsToString(positions, amms, balances, prices), errors
}

func (botdb *BotDB) QueryAllTablesByBlockToJson(blockHeight int64) (string, []error) {
	var errors []error
	amms, ammErr := botdb.QueryAmmByBlock(blockHeight)
	prices, pricesErr := botdb.QueryPricesByBlock(blockHeight)
	balances, balErr := botdb.QueryBalancesByBlock(blockHeight)
	positions, posErr := botdb.QueryPositionByBlock(blockHeight)

	errors = append(errors, ammErr, pricesErr, balErr, posErr)

	return DBRecordsToString(positions, amms, balances, prices), errors
}
