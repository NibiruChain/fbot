package fbot

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	oracleTypes "github.com/NibiruChain/nibiru/x/oracle/types"
	perpTypes "github.com/NibiruChain/nibiru/x/perp/v2/types"
	"github.com/Unique-Divine/gonibi"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/joho/godotenv"
)

var _ = app.BankModule.Name

type BotState struct {
	Positions         map[string]PositionFields
	Amms              map[string]AmmFields
	Prices            map[string]Prices
	PortfolioBalances Portfolio
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
	TmrpcAddr string
	DB        BotDB
	KeyName   string
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

func LoadBot() (*Bot, error) {
	godotenv.Load()
	var GRPC_ENDPOINT = os.Getenv("GRPC_ENDPONT")
	var CHAIN_ID = os.Getenv("CHAIN_ID")
	var TMRPC_ENDPOINT = os.Getenv("TMRPC_ENDPOINT")

	// Use default network info if .env is empty
	if GRPC_ENDPOINT == "" {
		GRPC_ENDPOINT = gonibi.DefaultNetworkInfo.GrpcEndpoint
	}
	if CHAIN_ID == "" {
		CHAIN_ID = gonibi.DefaultNetworkInfo.ChainID
	}
	if TMRPC_ENDPOINT == "" {
		TMRPC_ENDPOINT = gonibi.DefaultNetworkInfo.TmRpcEndpoint
	}

	return NewBot(BotArgs{
		ChainId:     CHAIN_ID,
		GrpcEndpt:   GRPC_ENDPOINT,
		RpcEndpt:    TMRPC_ENDPOINT,
		Mnemonic:    os.Getenv("VALIDATOR_MNEMONIC"),
		UseMnemonic: true,
		KeyName:     KEY_NAME,
	})

}

func Run(bot *Bot) error {

	context := context.Background()

	if err := bot.SyncState(); err != nil {
		return err
	}
	// Querying info for Prices/Amms structs
	err := bot.FetchNewPrices(context)
	if err != nil {
		return fmt.Errorf("Cannot FetchNewPrices(): %s", err)
	}

	blockHeight, err := bot.GetBlockHeight(context, bot.TmrpcAddr)
	if err != nil {
		return fmt.Errorf("Cannot GetHeight(): %s", err)
	} else {
		bot.DB.PopulateAmmsTable(bot.State.Amms, blockHeight)
		bot.DB.PopulatePricesTable(bot.State.Prices, blockHeight)
		bot.State.PortfolioBalances.BlockNumber = blockHeight
	}

	//Querying trader address to find positions by
	sdkAddress, err := bot.GetAddress()

	if err != nil {
		return fmt.Errorf("Cannot QueryAddress(): %s", err)
	}

	balancesResp, err := bot.State.PortfolioBalances.Balances.QueryWalletCoins(
		context, sdkAddress, bot.Gosdk.GrpcClient,
	)
	if err != nil {
		return fmt.Errorf("Cannot QueryWalletCoins(): %s", err)
	}

	bot.State.PortfolioBalances.Balances.PopWalletCoins(balancesResp)

	bot.DB.PopulateBalancesTable(bot.State.PortfolioBalances.Balances.WalletCoins,
		sdkAddress.String(), blockHeight)

	quoteToMove, err := bot.QuoteNeededToMovePrice()

	if err != nil {
		return fmt.Errorf("Cannot FindQuoteToMove: %s", err)
	}

	for pair, quote := range quoteToMove {
		quoteAmount := quote.RoundInt()
		_, action, err := bot.PerformTradeAction(pair, quoteAmount, sdkAddress, context)
		if err != nil {
			log.Fatalf("Cannot PerformTradeAction(): %v", err)
		}

		bot.UpdateTradeBalance(action, pair, quoteAmount)
	}

	balancesResp, err = bot.State.PortfolioBalances.Balances.QueryWalletCoins(
		context, sdkAddress, bot.Gosdk.GrpcClient,
	)
	if err != nil {
		return fmt.Errorf("Cannot QueryWalletCoins(): %s", err)
	}

	bot.State.PortfolioBalances.Balances.PopWalletCoins(balancesResp)

	return nil
}

func (bot *Bot) SyncState() (err error) {

	// TODO: Query & Update Balances, Query & Update Wallets

	return err
}

func (bot *Bot) UpdateTradeBalance(action TradeAction, pair string, quoteAmount sdk.Int) {

	switch action {
	case OpenOrder:
		bot.State.PortfolioBalances.Balances.AddTradedBalances(pair,
			sdk.NewCoin(asset.Pair(pair).QuoteDenom(), quoteAmount.Abs()))
	case CloseOrder:
		bot.State.PortfolioBalances.Balances.RemoveTradedBalances(pair,
			sdk.NewCoin(asset.Pair(pair).QuoteDenom(), quoteAmount.Abs()))
	}

}

func (bot *Bot) PerformTradeAction(pair string, quoteAmount sdk.Int,
	trader sdk.AccAddress, ctx context.Context) (*sdk.TxResponse, TradeAction, error) {
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
		txResp, err := bot.OpenPosition(trader, quoteAmount, sdk.NewDec(1), pair, ctx)
		return txResp, action, err
	case CloseOrder:
		txResp, err := bot.ClosePosition(trader, pair, ctx)
		return txResp, action, err
	case CloseAndOpenOrder:
		txResp, err := bot.CloseAndOpenPosition(trader, quoteAmount, pair, ctx)
		return txResp, action, err
	case DontTrade:
		return nil, action, nil
	default:
		return nil, action, fmt.Errorf("Invalid action type: %v", action)
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

func EvaluateTradeAction(QuoteToMove sdk.Int, amm perpTypes.AMM, posExists bool, position CurrPosStats) TradeAction {

	QuoteToMovePrice := sdk.NewDecFromInt(QuoteToMove)
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

	gosdk, err := gonibi.NewNibiruClient(chainId, grpcClientConnection, gonibi.DefaultNetworkInfo.TmRpcEndpoint)

	if err != nil {
		log.Fatal(err)
	}
	bot.Gosdk = &gosdk

	return bot
}

func (bot *Bot) PopulateGosdkFromNetinfo(netinfo gonibi.NetworkInfo) *Bot {
	return bot.PopulateGosdk(netinfo.GrpcEndpoint, netinfo.ChainID)
}

type BotArgs struct {
	ChainId     string
	GrpcEndpt   string
	RpcEndpt    string
	Mnemonic    string
	UseMnemonic bool
	KeyName     string
}

const KEY_NAME = "bot"

func NewBot(args BotArgs) (*Bot, error) {

	grpcConn, err := gonibi.GetGRPCConnection(args.GrpcEndpt, true, 5)

	if err != nil {
		return nil, err
	}

	gosdk, err := gonibi.NewNibiruClient(args.ChainId, grpcConn,
		args.RpcEndpt)
	if err != nil {
		return nil, err
	}

	var keyName string = KEY_NAME

	dontUseMnemonic := args.Mnemonic != "" || args.UseMnemonic == false

	if !dontUseMnemonic {
		_, privKey, err := gonibi.CreateSigner(args.Mnemonic, gosdk.Keyring,
			keyName)

		if err != nil {

			return nil, err
		}

		err = gonibi.AddSignerToKeyring(gosdk.Keyring, privKey, keyName)
		if err != nil {
			return nil, err

		}
	} else {
		if args.KeyName == "" {
			return nil, fmt.Errorf("No Key Name passed in")
		}
		keyName = args.KeyName
	}

	return &Bot{
		State: BotState{
			Positions:         make(map[string]PositionFields),
			Amms:              make(map[string]AmmFields),
			Prices:            make(map[string]Prices),
			PortfolioBalances: *InitializePortfolio(),
		},
		Gosdk:     &gosdk,
		TmrpcAddr: args.RpcEndpt,
		DB:        CreateAndConnectDB(),
		KeyName:   keyName,
	}, nil
}

func (bot *Bot) OpenPosition(trader sdk.AccAddress, quoteToMove sdk.Int,
	leverage sdk.Dec, pair string, ctx context.Context) (*sdk.TxResponse, error) {

	var side int32 = 0
	if quoteToMove.GT(sdk.NewInt(0)) {
		side = 1
	} else {
		side = 2
	}

	resp, err := bot.Gosdk.BroadcastMsgsGrpc(trader, &perpTypes.MsgMarketOrder{
		Sender:               trader.String(),
		Pair:                 asset.Pair(pair),
		Side:                 perpTypes.Direction(side),
		QuoteAssetAmount:     quoteToMove.Abs(),
		Leverage:             leverage,
		BaseAssetAmountLimit: sdk.NewInt(0),
	})

	if err != nil {
		return nil, err
	}
	bot.FetchAndPopPositionsDB(trader, ctx)

	return resp, err

}

func (bot *Bot) CloseAndOpenPosition(trader sdk.AccAddress,
	quoteToMove sdk.Int, pair string, ctx context.Context) (*sdk.TxResponse, error) {

	_, openErr := bot.ClosePosition(trader, pair, ctx)

	if openErr != nil {
		return nil, openErr
	}

	resp, closeErr := bot.OpenPosition(trader, quoteToMove, sdk.NewDec(1), pair, ctx)

	if closeErr != nil {
		return resp, closeErr
	}

	return resp, nil
}

func (bot *Bot) ClosePosition(trader sdk.AccAddress, pair string, ctx context.Context) (*sdk.TxResponse, error) {

	resp, err := bot.Gosdk.BroadcastMsgs(trader, &perpTypes.MsgClosePosition{
		Sender: trader.String(),
		Pair:   asset.Pair(pair),
	})

	bot.MakeZeroPosition(trader, pair, ctx)

	bot.FetchAndPopPositionsDB(trader, ctx)

	return resp, err
}

func (bot *Bot) FetchAndPopPositionsDB(trader sdk.AccAddress, ctx context.Context) error {

	// Querying positions and storing in bot.State and then in DB
	err := bot.FetchPositions(trader.String(), ctx)
	if err != nil {
		return err
	}

	height, err := bot.GetBlockHeight(ctx, bot.TmrpcAddr)
	if err != nil {
		return err
	} else {
		bot.DB.PopulatePositionTable(bot.State.Positions, height)
	}
	return nil
}

func (bot *Bot) GetKeyringRecord() (*keyring.Record, error) {

	keyRecords, err := bot.Gosdk.Keyring.List()
	var keyRingRecord *keyring.Record

	for _, keyRecord := range keyRecords {
		if keyRecord.Name == bot.KeyName {
			keyRingRecord = keyRecord
		}
	}
	if keyRingRecord == nil {
		return nil, fmt.Errorf("Key name %s not found in keyring", bot.KeyName)
	}
	return keyRingRecord, err
}

func (bot *Bot) GetAddress() (sdk.AccAddress, error) {
	record, err := bot.GetKeyringRecord()

	if err != nil {
		return nil, err
	}

	return record.GetAddress()
}

func (bot *Bot) FetchPositions(trader string, ctx context.Context) error {

	positions, err := bot.Gosdk.Querier.Perp.QueryPositions(ctx, &perpTypes.QueryPositionsRequest{
		Trader: trader,
	})

	if err != nil {
		return err
	}

	bot.PopulatePositions(positions)

	return nil
}

func (bot *Bot) FetchNewPrices(ctx context.Context) error {

	_, err := bot.Gosdk.Querier.Oracle.ExchangeRates(ctx, &oracleTypes.QueryExchangeRatesRequest{})
	if err != nil {
		return err
	}

	queryMarkets, err := bot.Gosdk.Querier.Perp.QueryMarkets(ctx, &perpTypes.QueryMarketsRequest{})
	if err != nil {
		return err
	}

	fakeRates := MockQueryRates()

	bot.PopulateAmms(queryMarkets)
	bot.PopulatePrices(&fakeRates, queryMarkets)

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

// To test making market orders if index & mark are too close
func MockQueryRates() oracleTypes.QueryExchangeRatesResponse {

	return oracleTypes.QueryExchangeRatesResponse{
		ExchangeRates: []oracleTypes.ExchangeRateTuple{
			{
				Pair:         asset.NewPair("ubtc", "unusd"),
				ExchangeRate: sdk.NewDec(35000),
			},
			{
				Pair:         asset.NewPair("ueth", "unusd"),
				ExchangeRate: sdk.NewDec(1500),
			},
			{
				Pair:         asset.NewPair("uatom", "unusd"),
				ExchangeRate: sdk.NewDec(100),
			},
			{
				Pair:         asset.NewPair("unibi", "unusd"),
				ExchangeRate: sdk.NewDec(500),
			},
			{
				Pair:         asset.NewPair("uosmo", "unusd"),
				ExchangeRate: sdk.NewDec(600),
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

func (bot *Bot) QuoteNeededToMovePrice() (map[string]sdk.Dec, error) {

	var quoteReserveMap = make(map[string]sdk.Dec)

	for key, value := range bot.State.Amms {
		quoteReserveMap[key] = value.Markets.QuoteReserve

	}

	var qp = make(map[string]sdk.Dec)

	for key := range bot.State.Amms {
		qpTemp, err := common.SqrtDec(bot.State.Prices[key].IndexPrice.Quo(bot.State.Prices[key].MarkPrice))
		if err != nil {
			return nil, err
		}
		qp[key] = qpTemp
	}

	var quoteToMove = make(map[string]sdk.Dec)

	for key, value := range quoteReserveMap {
		quoteToMove[key] = ((value.Quo(qp[key])).Sub(value)).Mul(sdk.NewDec(-1))
	}

	return quoteToMove, nil

}

func (bot *Bot) GetBlockHeight(ctx context.Context, tmrpcEndpoint string) (int64, error) {
	// rpc, rpcErr := rpchttp.New(gonibi.DefaultNetworkInfo.TmRpcEndpoint,
	// 	"/websocket")
	rpc, rpcErr := rpchttp.New(tmrpcEndpoint,
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

func (bot *Bot) MakeZeroPosition(trader sdk.AccAddress, pair string, ctx context.Context) {

	bot.Gosdk.BroadcastMsgs(trader, &perpTypes.MsgMarketOrder{
		Sender:               trader.String(),
		Pair:                 asset.Pair(pair),
		Side:                 2,
		QuoteAssetAmount:     sdk.ZeroInt(),
		Leverage:             sdk.ZeroDec(),
		BaseAssetAmountLimit: sdk.ZeroInt(),
	})

}
