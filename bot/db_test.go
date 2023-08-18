package fbot_test

import (
	fbot "fbot/bot"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

type DBSuite struct {
	suite.Suite
	DB            *fbot.BotDB
	records       fbot.DBRecords
	recordsString string
}

func TestDB(t *testing.T) {
	suite.Run(t, new(DBSuite))
}
func (db *DBSuite) TestDBSuite() {
	db.SetupDB()
	db.T().Run("RunTestPopulatePricesTable", db.RunTestPopulatePricesTable)
	db.T().Run("RunTestQueryPricesByBlock", db.RunTestQueryPricesByBlock)
	db.T().Run("RunTestQueryAllPrices", db.RunTestQueryAllPrices)
	db.T().Run("RunTestNewDBRecordsFromString", db.RunTestNewDBRecordsFromString)
	db.T().Run("RunTestRecordsString", db.RunTestRecordsString)

}

func (db *DBSuite) SetupDB() {
	botDB := new(fbot.BotDB)
	db.DB = botDB
	botDB.ConnectToDB("db_test.db")
}

func (db *DBSuite) TearDownAllSuite() {
	db.DB.ClearDB()
	db.DB.DeleteDB()
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
	prices, err := db.DB.QueryPricesByBlock(1)
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
	//fmt.Printf("Prices block: %v \n \n", qBlock)
	for _, price := range qBlock {
		db.NotNil(t, price.Pair)
		db.NotNil(t, price.IndexPrice)
		db.NotNil(t, price.MarkPrice)
		//fmt.Printf("PRICES STRING: %s", price)
	}
}

func (db *DBSuite) RunTestNewDBRecordsFromString(t *testing.T) {

	db.recordsString = `
	{
		"amms": [
			{
				"Pair": "ubtc:unusd",
				"BaseReserve": "100",
				"QuoteReserve": "200",
				"BlockHeight": 1234,
				"Bias": "1000"
			}
		],
		"prices": [
			{
				"Pair": "ueth:unusd",
				"IndexPrice": "10000",
				"MarkPrice": "10200",
				"BlockHeight": 1234
			}
		],
		"positions": [
			{
				"Pair": "ubtc:unusd",
				"UnrealizedPnl": "100",
				"Size": "1",
				"Trader": "nibi1234",
				"BlockHeight": 1234
			}
		],
		"balances": [
			{
				"Pair": "ueth",
				"Amount": "10",
				"BlockHeight": 1234
			}
		]
	}`

	records, err := fbot.NewDBRecordsFromString(db.recordsString)
	db.NoError(err)
	db.NotNil(t, records)
	db.records = records
}

func (db *DBSuite) RunTestRecordsString(t *testing.T) {
	recordJson := db.records.String()
	db.NotNil(t, recordJson)
}
