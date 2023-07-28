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
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ = app.BankModule.Name

type BotState struct {
	Positions map[string]perpTypes.Position
	Amms      map[string]perpTypes.AMM
	Prices    map[string]Prices
	Funds     map[string]sdk.Int
}

type Bot struct {
	State BotState
	Gosdk *gonibi.NibiruClient
	// db field
}

type Prices struct {
	IndexPrice sdk.Dec
	MarkPrice  sdk.Dec
}

// func StoreTradeResult() {

// }

func main() {
	var LOGGING_FILE = "test-log.txt"
	SetupLoggingFile(LOGGING_FILE)
	fmt.Print(LOGGING_FILE)

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
			Positions: make(map[string]perpTypes.Position),
			Amms:      make(map[string]perpTypes.AMM),
			Prices:    make(map[string]Prices)},
		Gosdk: &gonibi.NibiruClient{},
	}
}

func (bot *Bot) PopulateAmms(queryMarketsResp *perpTypes.QueryMarketsResponse) {
	for index, value := range queryMarketsResp.AmmMarkets {
		pair := value.Amm.Pair
		bot.State.Amms[pair.String()] = queryMarketsResp.AmmMarkets[index].Amm
	}

}

// Make start their own network (look at grpcclientsuite setupsuite())
// func RunNetwork() {
// 	gonibi.
// }

func (bot *Bot) MakePosition() {

	bot.Gosdk.Tx.BroadcastMsgs()
}

func (bot *Bot) QueryAddress(nodeDirName string) (sdk.AccAddress, error) {
	mnemonic := os.Getenv("VALIDATOR_MNEMONIC")
	//kb, err := s.gosdk.Keyring.List()

	signAlgo, _ := bot.Gosdk.Keyring.SupportedAlgorithms()
	addr, _, err := sdktestutil.GenerateSaveCoinKey(
		bot.Gosdk.Keyring, nodeDirName, mnemonic, true, signAlgo[0],
	)
	return addr, err
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
		bot.State.Positions[pair.String()] = positionResponse.Position
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

func ShouldTrade(QuoteToMovePrice sdk.Dec, amm perpTypes.AMM) bool {
	if QuoteToMovePrice.Abs().LT(amm.QuoteReserve.QuoInt64(20)) {
		return false
	} else {
		return true
	}

}

func QuoteNeededToMovePrice(bot Bot) map[string]sdk.Dec {

	var quoteReserveMap = make(map[string]sdk.Dec)

	for key, value := range bot.State.Amms {
		quoteReserveMap[key] = value.QuoteReserve
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
