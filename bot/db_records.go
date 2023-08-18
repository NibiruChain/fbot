package fbot

import "encoding/json"

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
