package fbot

import (
	"context"
	"fmt"
	"log"
	"os"

	"cosmossdk.io/math"
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	oracleTypes "github.com/NibiruChain/nibiru/x/oracle/types"
	perpTypes "github.com/NibiruChain/nibiru/x/perp/v2/types"
	"github.com/Unique-Divine/gonibi"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/joho/godotenv"
)

var _ = app.BankModule.Name

type BotState struct {
	Positions map[string]PositionFields
	Amms      map[string]AmmFields
	Prices    map[string]Prices
	Funds     map[string]math.Int
}

type PositionFields struct {
	Positon       perpTypes.Position
	UnrealizedPnl sdk.Dec
}

type AmmFields struct {
	Markets perpTypes.AMM
	Bias    sdk.Dec
}

type Bot struct {
	State     BotState
	Gosdk     *gonibi.NibiruClient
	RpcClient *rpchttp.HTTP
}

type Prices struct {
	IndexPrice sdk.Dec
	MarkPrice  sdk.Dec
}

type CurrPosStats struct {
	CurrMarkPrice   sdk.Dec
	CurrIndexPrice  sdk.Dec
	CurrSize        sdk.Dec
	PriceMultiplier sdk.Dec
	MarketDelta     sdk.Dec
	UnrealizedPnl   sdk.Dec
	IsAgainstMarket bool
}
type TradeAction int

const (
	OpenOrder TradeAction = iota
	CloseOrder
	CloseAndOpenOrder
	DontTrade
)

func main() {
	var LOGGING_FILE = "test-log.txt"
	SetupLoggingFile(LOGGING_FILE)
	fmt.Print(LOGGING_FILE)

}

func Run() {

	godotenv.Load()
	var GRPC_ENDPOINT = os.Getenv("GRPC_ENDPONT")
	var CHAIN_ID = os.Getenv("CHAIN_ID")

	// Use default network info if .env is empty
	if GRPC_ENDPOINT == "" {
		GRPC_ENDPOINT = gonibi.DefaultNetworkInfo.GrpcEndpoint
	}
	if CHAIN_ID == "" {
		CHAIN_ID = gonibi.DefaultNetworkInfo.ChainID
	}

	var bot = NewBot().PopulateGosdk(GRPC_ENDPOINT, CHAIN_ID)
	context := context.Background()
	db := CreateAndConnectDB()
	db.ClearDB()

	// Querying info for Prices/Amms structs
	pricesErr := bot.FetchNewPrices(context)
	blockHeight, blockHeightErr := bot.GetBlockHeight(context)

	if blockHeightErr != nil {
		log.Fatalf("Cannot GetBlockHeight(): %v", blockHeight)
	}

	if pricesErr != nil {
		log.Fatalf("Cannot FetchNewPrices(): %v", pricesErr)
	} else {
		db.PopulateAmmsTable(bot.State.Amms, blockHeight)
		db.PopulatePricesTable(bot.State.Prices, blockHeight)
	}

	// Querying trader address to find positions by
	sdkAddress, addressErr := bot.QueryAddress(os.TempDir())

	if addressErr != nil {
		log.Fatalf("Cannot QueryAddress(): %v", sdkAddress)
	}

	// Querying positions and storing in bot.State
	positionsErr := bot.FetchPositions(sdkAddress.String(), context)

	if positionsErr != nil {
		log.Fatalf("Cannot FetchPositions(): %v", positionsErr)
	} else {
		db.PopulatePositionTable(bot.State.Positions, blockHeight)
	}

	balancesErr := bot.FetchBalances(context)

	if balancesErr != nil {
		log.Fatalf("Cannot FetchBalances: %v", balancesErr)
	} else {
		db.PopulateBalancesTable(bot.State.Funds, blockHeight)
	}

	dbTables, errs := db.QueryAllTablesToJson()

	for _, err := range errs {
		if err != nil {
			log.Fatalf("Cannot QueryTable: %v", err)
		}
	}

	fmt.Print(dbTables)

	quoteToMove := QuoteNeededToMovePrice(*bot)

	for pair, quoteAmount := range quoteToMove {
		bot.PerformTradeAction(pair, quoteAmount)
	}

}

func (bot *Bot) PerformTradeAction(pair string, quoteAmount sdk.Dec) error {
	_, posExists := bot.State.Positions[pair]

	currPosition := CurrPosStats{
		CurrMarkPrice:   sdk.NewDec(0),
		CurrIndexPrice:  sdk.NewDec(0),
		CurrSize:        sdk.NewDec(0),
		PriceMultiplier: sdk.NewDec(0),
		MarketDelta:     sdk.NewDec(0),
		UnrealizedPnl:   sdk.NewDec(0),
		IsAgainstMarket: false,
	}

	if posExists {
		currPosition = bot.PopulateCurrPosStats(pair)
	}

	action := EvaluateTradeAction(quoteAmount, bot.State.Amms[pair].Markets, posExists, currPosition)
	switch action {
	case OpenOrder:
		return bot.OpenPosition(quoteAmount.RoundInt(), sdk.NewDec(1))
	case CloseOrder:
		return bot.ClosePosition(pair)
	case CloseAndOpenOrder:
		return bot.CloseAndOpenPosition(quoteAmount.RoundInt(), pair)
	case DontTrade:
		return nil
	default:
		return fmt.Errorf("Invalid action type: %v", action)
	}
}

func (bot *Bot) PopulateCurrPosStats(pair string) CurrPosStats {
	MarkPrice := bot.State.Prices[pair].MarkPrice
	IndexPrice := bot.State.Prices[pair].IndexPrice
	Size := bot.State.Positions[pair].Positon.Size_
	PriceMult := bot.State.Amms[pair].Markets.PriceMultiplier
	MarketDelta := MarkPrice.Sub(IndexPrice).Mul(PriceMult).Abs()
	IsPosAgainstMarket := IsPosAgainstMarket(Size,
		MarkPrice, IndexPrice)
	UnrealizedPnl := bot.State.Positions[pair].UnrealizedPnl

	position := CurrPosStats{
		CurrMarkPrice:   MarkPrice,
		CurrIndexPrice:  IndexPrice,
		CurrSize:        Size,
		PriceMultiplier: PriceMult,
		MarketDelta:     MarketDelta,
		IsAgainstMarket: IsPosAgainstMarket,
		UnrealizedPnl:   UnrealizedPnl,
	}

	return position
}

func EvaluateTradeAction(QuoteToMovePrice sdk.Dec, amm perpTypes.AMM, posExists bool, position CurrPosStats) TradeAction {

	if ShouldNotTrade(QuoteToMovePrice, amm.QuoteReserve) &&
		posExists && position.IsAgainstMarket &&
		position.MarketDelta.GT(position.CurrIndexPrice.Quo(sdk.NewDec(10))) {
		return CloseOrder
	} else if !posExists && !ShouldNotTrade(QuoteToMovePrice, amm.QuoteReserve) {
		return OpenOrder
	} else if !position.IsAgainstMarket &&
		position.UnrealizedPnl.GT(position.CurrSize.Abs().Quo(sdk.NewDec(10))) {
		return CloseAndOpenOrder
	} else {
		return DontTrade
	}

}

func (bot *Bot) PopulateGosdk(grpcUrl string, chainId string) *Bot {
	grpcClientConnection, err := gonibi.GetGRPCConnection(
		grpcUrl, true, 5)
	if err != nil {
		log.Fatal(err)
	}

	gosdk, err := gonibi.NewNibiruClient(chainId, grpcClientConnection)

	if err != nil {
		log.Fatal(err)
	}
	bot.Gosdk = &gosdk

	return bot
}

func (bot *Bot) PopulateGosdkFromNetinfo(netinfo gonibi.NetworkInfo) *Bot {
	return bot.PopulateGosdk(netinfo.GrpcEndpoint, netinfo.ChainID)
}

func NewBot() *Bot {
	return &Bot{
		State: BotState{
			Positions: make(map[string]PositionFields),
			Amms:      make(map[string]AmmFields),
			Prices:    make(map[string]Prices)},
		Gosdk: &gonibi.NibiruClient{},
	}
}

func (bot *Bot) OpenPosition(quoteToMove math.Int, leverage sdk.Dec) error {

	// var txResp sdk.TxResponse
	// txResp.Logs
	// var willBeLong bool
	// if quoteToMove > 0 {
	// 	willBeLong = true
	// } else {
	// 	willBeLongLong = false
	// }

	// bot.Gosdk.Tx.BroadcastMsgs()

	return nil
}

func (bot *Bot) CloseAndOpenPosition(quoteToMove sdk.Int, pair string) error {
	return nil
}

func (bot *Bot) ClosePosition(pair string) error {

	return nil
}

func (bot *Bot) QueryAddress(nodeDirName string) (sdk.AccAddress, error) {
	mnemonic := os.Getenv("VALIDATOR_MNEMONIC")

	signAlgo, _ := bot.Gosdk.Keyring.SupportedAlgorithms()
	addr, _, err := sdktestutil.GenerateSaveCoinKey(
		bot.Gosdk.Keyring, nodeDirName, mnemonic, true, signAlgo[0],
	)

	return addr, err
}

// token balance query
func (bot *Bot) FetchBalances(ctx context.Context) error {
	moduleAccounts, err := bot.Gosdk.Query.Perp.ModuleAccounts(ctx, &perpTypes.QueryModuleAccountsRequest{})

	if err != nil {
		return err
	}

	bot.PopulateBalances(moduleAccounts)

	return nil
}

func (bot *Bot) PopulateBalances(moduleAccounts *perpTypes.QueryModuleAccountsResponse) {
	for _, perpAccount := range moduleAccounts.GetAccounts() {
		for _, coin := range perpAccount.Balance {
			bot.State.Funds[coin.Denom] = coin.Amount
		}
	}
}

func (bot *Bot) FetchPositions(trader string, ctx context.Context) error {

	positions, err := bot.Gosdk.Query.Perp.QueryPositions(ctx, &perpTypes.QueryPositionsRequest{
		Trader: trader,
	})

	if err != nil {
		return err
	}

	bot.PopulatePositions(positions)

	return nil
}

func (bot *Bot) PopulatePositions(positions *perpTypes.QueryPositionsResponse) {
	for _, positionResponse := range positions.GetPositions() {
		pair := positionResponse.Position.Pair
		bot.State.Positions[pair.String()] = PositionFields{
			Positon:       positionResponse.Position,
			UnrealizedPnl: positionResponse.UnrealizedPnl,
		}
	}

}

func (bot *Bot) FetchNewPrices(ctx context.Context) error {

	oracle, err := bot.Gosdk.Query.Oracle.ExchangeRates(ctx, &oracleTypes.QueryExchangeRatesRequest{})
	if err != nil {
		return err
	}
	queryMarkets, err := bot.Gosdk.Query.Perp.QueryMarkets(ctx, &perpTypes.QueryMarketsRequest{})
	if err != nil {
		return err
	}
	bot.PopulateAmms(queryMarkets)
	bot.PopulatePrices(oracle, queryMarkets)

	return nil
}

func (bot *Bot) PopulateAmms(queryMarketsResp *perpTypes.QueryMarketsResponse) {
	for index, value := range queryMarketsResp.AmmMarkets {
		pair := value.Amm.Pair
		bot.State.Amms[pair.String()] = AmmFields{
			Markets: queryMarketsResp.AmmMarkets[index].Amm,
			Bias:    value.Amm.Bias(),
		}
	}

}

func (bot *Bot) PopulatePrices(oracle *oracleTypes.QueryExchangeRatesResponse,
	queryMarkets *perpTypes.QueryMarketsResponse) {

	queryRatesMap := oracle.ExchangeRates.ToMap()
	for _, value := range queryMarkets.AmmMarkets {
		pair := value.Amm.Pair
		indexPrice, exists := queryRatesMap[pair]
		if !exists {
			continue
		}
		prices := Prices{IndexPrice: indexPrice, MarkPrice: value.Amm.MarkPrice()}
		bot.State.Prices[pair.String()] = prices
	}

}

func MockQueryRates() oracleTypes.QueryExchangeRatesResponse {

	return oracleTypes.QueryExchangeRatesResponse{
		ExchangeRates: []oracleTypes.ExchangeRateTuple{
			{
				Pair:         asset.NewPair("ubtc", "unusd"),
				ExchangeRate: sdk.NewDec(30000),
			},
			{
				Pair:         asset.NewPair("ueth", "unusd"),
				ExchangeRate: sdk.NewDec(30000),
			},
		},
	}
}

func ShouldNotTrade(quoteToMovePrice sdk.Dec, quoteReserve sdk.Dec) bool {
	if quoteToMovePrice.Abs().LT(quoteReserve.QuoInt64(20)) {
		return true
	}
	return false
}

func QuoteNeededToMovePrice(bot Bot) map[string]sdk.Dec {

	var quoteReserveMap = make(map[string]sdk.Dec)

	for key, value := range bot.State.Amms {
		quoteReserveMap[key] = value.Markets.QuoteReserve
	}
	// use for loop
	var qp = make(map[string]sdk.Dec)

	for key := range bot.State.Amms {
		qpTemp, err := common.SqrtDec(bot.State.Prices[key].IndexPrice.Quo(bot.State.Prices[key].MarkPrice))
		if err != nil {
			log.Fatal(err)
		}
		qp[key] = qpTemp
	}

	var quoteToMove = make(map[string]sdk.Dec)

	for key, value := range quoteReserveMap {
		quoteToMove[key] = ((value.Quo(qp[key])).Sub(value)).Mul(sdk.NewDec(-1))
	}

	return quoteToMove

}

func (bot *Bot) GetBlockHeight(ctx context.Context) (int64, error) {
	rpc, rpcErr := rpchttp.New(gonibi.DefaultNetworkInfo.TmRpcEndpoint,
		"/websocket")

	if rpcErr != nil {
		return -1, rpcErr
	}

	abciInfo, abciErr := rpc.ABCIInfo(ctx)

	if abciErr != nil {
		return -1, abciErr
	}

	latestBlockHeight := abciInfo.Response.LastBlockHeight
	return latestBlockHeight, nil
}

// func LoggingFilename() string {

// 	// TODO  Specify the logging file name with a command line arg
// 	// if os.Arg...
// 	// TODO  Specify the logging file name with a config variable
// 	//return LOGGING_FILE
// }

// Initializes a logging file with the given file name.

func SetupLoggingFile(loggingFilename string) {

	// Make blank file
	file, err := os.Create(loggingFilename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	log.SetOutput(file)
	log.Printf("logger name: %v", loggingFilename)
}

// IsPosAgainstMarket returns true if the position is diverging the mark and
// index price. In other words, it returns true if the trader is paying funding
// on this position rather than receiving it.
func IsPosAgainstMarket(posSize sdk.Dec, mark sdk.Dec, index sdk.Dec) bool {
	marketLong := mark.GT(index)
	posLong := posSize.IsPositive()
	return marketLong != posLong
}
