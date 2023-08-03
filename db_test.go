package fbot_test

import (
	"fbot"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

type DBSuite struct {
	suite.Suite
	DB *fbot.BotDB
}

func TestDB(t *testing.T) {
	suite.Run(t, new(DBSuite))
}
func (db *DBSuite) TestDBSuite() {
	db.SetupDB()
	db.T().Run("RunTestPopulatePricesTable", db.RunTestPopulatePricesTable)
	db.T().Run("RunTestQueryPricesByBlock", db.RunTestQueryPricesByBlock)
	db.T().Run("RunTestQueryAllPrices", db.RunTestQueryAllPrices)
	db.T().Run("RunTestNewDbExportFromString", db.RunTestNewDbExportFromString)
	db.TearDownSuite()
}

func (db *DBSuite) SetupDB() {
	botDB := new(fbot.BotDB)
	db.DB = botDB
	botDB.ConnectToDB()
}

func (db *DBSuite) TearDownSuite() {
	db.DB.ClearDB()
}

func (db *DBSuite) RunTestPopulatePricesTable(t *testing.T) {
	Prices := map[string]fbot.Prices{
		"ubtc:unusd": {
			IndexPrice: sdk.NewDec(100),
			MarkPrice:  sdk.NewDec(125),
		},
		"ueth:unusd": {
			IndexPrice: sdk.NewDec(100),
			MarkPrice:  sdk.NewDec(125),
		},
	}
	db.DB.PopulatePricesTable(Prices, 1)
}

func (db *DBSuite) RunTestQueryPricesByBlock(t *testing.T) {
	prices, err := db.DB.QueryPricesTableByBlock(1)
	db.NoError(err)
	for _, price := range prices {
		db.NotNil(t, price.Pair)
		db.NotNil(t, price.IndexPrice)
		db.NotNil(t, price.MarkPrice)

	}
}

func (db *DBSuite) RunTestQueryAllPrices(t *testing.T) {
	qBlock, err := db.DB.QueryPricesTable()
	db.NoError(err)
	for _, price := range qBlock {
		db.NotNil(t, price.Pair)
		db.NotNil(t, price.IndexPrice)
		db.NotNil(t, price.MarkPrice)
	}
}

func (db *DBSuite) RunTestNewDbExportFromString(t *testing.T) {
	db.DB.NewDbExportFromString(`{
		"Pair":       "ubtc:unusd",
		"UnrealizedPnl": "42",
		"Size":          "100",
		"Trader":        "nibi1234",
		"BlockHeight":   1,
	}`, `{
		"Pair":       "ueth:unusd",
		"BaseReserve": "500",
		"QuoteReserve":          "500",
		"BlockHeight":        "1",
		"Bias":   -60,
	}`, `{
		"Pair":	"ubtc:unusd",
		"Amount": "700",
		"BlockHeight":   1,
	}`, `{
		"Pair":       "ueth:unusd",
		"IndexPrice": "3200",
		"MarkPrice":          "2600",
		"BlockHeight":   1,
	}`)
}
