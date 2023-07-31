package fbot_test

import (
	"bufio"
	"context"
	"fbot"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/testutil/cli"
	perpTypes "github.com/NibiruChain/nibiru/x/perp/v2/types"
	"github.com/Unique-Divine/gonibi"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

type BotSuite struct {
	suite.Suite
	bot     *fbot.Bot
	gosdk   *gonibi.NibiruClient
	ctx     context.Context
	address sdk.AccAddress
}

func TestBot(t *testing.T) {
	suite.Run(t, new(BotSuite))
}

// func TestRun(t *testing.T) {
// 	fbot.Run()
// }

// Example of iterative test cases
func TestIsPosAgainstMarket(t *testing.T) {

	for _, tc := range []struct {
		name      string
		posSize   sdk.Dec
		mark      sdk.Dec
		index     sdk.Dec
		isAgainst bool
	}{
		{
			name:    "pos long, mark < index",
			posSize: sdk.NewDec(10), mark: sdk.NewDec(10), index: sdk.NewDec(20), isAgainst: true},
		{
			name:    "pos long, mark > index",
			posSize: sdk.NewDec(10), mark: sdk.NewDec(20), index: sdk.NewDec(10), isAgainst: false},
		{
			name:    "pos short, mark < index",
			posSize: sdk.NewDec(-10), mark: sdk.NewDec(10), index: sdk.NewDec(20), isAgainst: false},
		{
			name:    "pos short, mark > index",
			posSize: sdk.NewDec(-10), mark: sdk.NewDec(20), index: sdk.NewDec(10), isAgainst: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.isAgainst, fbot.IsPosAgainstMarket(tc.posSize, tc.mark, tc.index))
		})

	}
}

type BlockChain struct {
	suite.Suite
	gosdk    *gonibi.NibiruClient
	grpcConn *grpc.ClientConn
	cfg      *cli.Config
	network  *cli.Network
	val      *cli.Validator
}

// func TestRunChain(t *testing.T) {
// 	gonibi.
// }

// func (chain *BlockChain) SetupChain() {
// 	app.SetPrefixes(app.AccountAddressPrefix)
// 	encConfig := app.MakeEncodingConfig()
// 	genState := genesis.NewTestGenesisState(encConfig)
// 	cliCfg := cli.BuildNetworkConfig(genState)
// 	chain.cfg = &cliCfg
// 	chain.cfg.NumValidators = 1

// 	network, err := cli.New(
// 		chain.T(),
// 		chain.T().TempDir(),
// 		*chain.cfg,
// 	)
// 	chain.NoError(err)
// 	chain.network = network
// 	chain.NoError(chain.network.WaitForNextBlock())

// 	chain.val = chain.network.Validators[0]
// 	AbsorbServerConfig(chain.cfg, chain.val.AppConfig)
// 	AbsorbTmConfig(chain.cfg, chain.val.Ctx.Config)
// 	chain.ConnectGrpc()

// }

func TestSetupLoggingFile(t *testing.T) {
	filename := "temp-test"
	if _, err := os.Stat(filename); err == nil {
		err := os.Remove(filename)
		require.NoError(t, err)
	}

	fbot.SetupLoggingFile(filename)
	file, err := os.Open(filename)
	defer file.Close()
	require.NoError(t, err)

	var hasExpectedContent bool
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, fmt.Sprintf("logger name: %v", filename)) {
			hasExpectedContent = true
		}
	}

	err = scanner.Err()
	require.NoError(t, err)
	require.True(t, hasExpectedContent)

	require.NoError(t, os.Remove(filename))
}

func (s *BotSuite) TestBotSuite() {
	s.SetupGoSdk()
	// s.T().Run("RunTestPopulatePrices", s.RunTestFetchPrices)
	// s.T().Run("RunTestPopulateAmms", s.RunTestPopulateAmms)
	// s.T().Run("RunTestQuoteNeededToMovePrice", s.RunTestQuoteNeededToMovePrice)
	// s.T().Run("RunTestQueryAddress", s.RunTestQueryAddress)
	s.T().Run("RunTestEvaluateTradeAction", s.RunTestEvaluateTradeAction)
	//s.T().Run("RunTestFetchPositions", s.RunTestFetchPositions)
}

func (s *BotSuite) SetupGoSdk() {
	netInfo := gonibi.DefaultNetworkInfo
	grpcClientConnection, err := gonibi.GetGRPCConnection(
		netInfo.GrpcEndpoint, true, 5)

	s.Require().NoError(err)

	gosdk, err := gonibi.NewNibiruClient(netInfo.ChainID, grpcClientConnection)
	s.NoError(err)
	s.ctx = context.Background()

	s.gosdk = &gosdk
	s.bot = fbot.NewBot().PopulateGosdkFromNetinfo(netInfo)
}
func PrivKeyAddressPairs(n int) (keys []cryptotypes.PrivKey, addrs []sdk.AccAddress) {
	r := rand.New(rand.NewSource(12345)) // make the generation deterministic
	keys = make([]cryptotypes.PrivKey, n)
	addrs = make([]sdk.AccAddress, n)
	for i := 0; i < n; i++ {
		secret := make([]byte, 32)
		_, err := r.Read(secret)
		if err != nil {
			panic("Could not read randomness")
		}
		keys[i] = secp256k1.GenPrivKeyFromSecret(secret)
		addrs[i] = sdk.AccAddress(keys[i].PubKey().Address())
	}
	return
}

// take nodeDirName string
func (s *BotSuite) RunTestQueryAddress(t *testing.T) {
	addr, err := s.bot.QueryAddress(s.T().TempDir())
	s.NoError(err)
	s.NotNil(addr)
	s.address = addr
}

func (s *BotSuite) RunTestPopulateAmms(t *testing.T) {

	resp, err := s.gosdk.Query.Perp.QueryMarkets(
		s.ctx, &perpTypes.QueryMarketsRequest{},
	)
	s.NoErrorf(err, "Perp Resp: %v", resp)

	var bot = fbot.NewBot()

	bot.PopulateAmms(resp)
	s.NotNil(bot.State.Amms["ubtc:unusd"])
	s.NotNil(bot.State.Amms["ueth:unusd"])

	//gosdk, err := gonibi.NewNibiruClient(netInfo.ChainID, grpcClientConnection)
	//gosdk.Tx.BroadcastMsgs()

}

func (s *BotSuite) RunTestFetchPrices(t *testing.T) {

	err := s.bot.FetchNewPrices(s.ctx)
	s.NoError(err)

}

func (s *BotSuite) RunTestFetchPositions(t *testing.T) {
	err := s.bot.FetchPositions(string(s.address), s.ctx)
	fmt.Print("Positions ", s.bot.State.Positions["ubtc:unusd"])
	s.NoError(err)

}

func (s *BotSuite) RunTestEvaluateTradeAction(t *testing.T) {
	for _, tc := range []struct {
		name        string
		quoteAmount sdk.Dec
		amm         perpTypes.AMM
		posExists   bool
		position    fbot.CurrPosStats
		tradeAction fbot.TradeAction
	}{
		{
			name:        "Open Order",
			quoteAmount: sdk.NewDec(3500), amm: perpTypes.AMM{
				Pair:            "ubtc:unusd",
				BaseReserve:     sdk.NewDec(10000),
				QuoteReserve:    sdk.NewDec(10000),
				SqrtDepth:       sdk.NewDec(10),
				PriceMultiplier: sdk.NewDec(10),
				TotalLong:       sdk.NewDec(10),
				TotalShort:      sdk.NewDec(10),
			}, posExists: false, position: fbot.CurrPosStats{
				CurrMarkPrice:   sdk.NewDec(5),
				CurrIndexPrice:  sdk.NewDec(10),
				CurrSize:        sdk.NewDec(10),
				PriceMultiplier: sdk.NewDec(10),
				MarketDelta:     sdk.NewDec(10),
				UnrealizedPnl:   sdk.NewDec(10),
				IsAgainstMarket: false,
			},
			tradeAction: fbot.OpenOrder,
		},
		{
			name:        "Close Order",
			quoteAmount: sdk.NewDec(350), amm: perpTypes.AMM{
				Pair:            "ueth:unusd",
				BaseReserve:     sdk.NewDec(10000),
				QuoteReserve:    sdk.NewDec(10000),
				SqrtDepth:       sdk.NewDec(10),
				PriceMultiplier: sdk.NewDec(10),
				TotalLong:       sdk.NewDec(10),
				TotalShort:      sdk.NewDec(10),
			}, posExists: true, position: fbot.CurrPosStats{
				CurrMarkPrice:   sdk.NewDec(5),
				CurrIndexPrice:  sdk.NewDec(2000),
				CurrSize:        sdk.NewDec(10),
				PriceMultiplier: sdk.NewDec(10),
				MarketDelta:     sdk.NewDec(1000),
				UnrealizedPnl:   sdk.NewDec(10),
				IsAgainstMarket: true,
			},
			tradeAction: fbot.CloseOrder,
		},
		{
			name:        "CloseAndOpenOrder",
			quoteAmount: sdk.NewDec(350), amm: perpTypes.AMM{
				Pair:            "ueth:unusd",
				BaseReserve:     sdk.NewDec(10000),
				QuoteReserve:    sdk.NewDec(10000),
				SqrtDepth:       sdk.NewDec(10),
				PriceMultiplier: sdk.NewDec(10),
				TotalLong:       sdk.NewDec(10),
				TotalShort:      sdk.NewDec(10),
			}, posExists: true, position: fbot.CurrPosStats{
				CurrMarkPrice:   sdk.NewDec(5),
				CurrIndexPrice:  sdk.NewDec(2000),
				CurrSize:        sdk.NewDec(2500),
				PriceMultiplier: sdk.NewDec(10),
				MarketDelta:     sdk.NewDec(1000),
				UnrealizedPnl:   sdk.NewDec(1000),
				IsAgainstMarket: false,
			},
			tradeAction: fbot.CloseAndOpenOrder,
		},
		{
			name:        "DontTrade",
			quoteAmount: sdk.NewDec(350), amm: perpTypes.AMM{
				Pair:            "ubtc:unusd",
				BaseReserve:     sdk.NewDec(10000),
				QuoteReserve:    sdk.NewDec(10000),
				SqrtDepth:       sdk.NewDec(10),
				PriceMultiplier: sdk.NewDec(10),
				TotalLong:       sdk.NewDec(10),
				TotalShort:      sdk.NewDec(10),
			}, posExists: false, position: fbot.CurrPosStats{
				CurrMarkPrice:   sdk.NewDec(2000),
				CurrIndexPrice:  sdk.NewDec(2000),
				CurrSize:        sdk.NewDec(10),
				PriceMultiplier: sdk.NewDec(10),
				MarketDelta:     sdk.NewDec(50),
				UnrealizedPnl:   sdk.NewDec(10),
				IsAgainstMarket: true,
			},
			tradeAction: fbot.DontTrade,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tradeAction := fbot.EvaluateTradeAction(tc.quoteAmount, tc.amm, tc.posExists, tc.position)
			require.Equal(t, tradeAction, tc.tradeAction)
		})
	}
}

func (s *BotSuite) RunTestQuoteNeededToMovePrice(t *testing.T) {

	s.bot.State.Amms = map[string]perpTypes.AMM{
		"ubtc:unusd": {
			Pair:            "ubtc:unusd",
			BaseReserve:     sdk.NewDec(10),
			QuoteReserve:    sdk.NewDec(10),
			SqrtDepth:       sdk.NewDec(10),
			PriceMultiplier: sdk.NewDec(10),
			TotalLong:       sdk.NewDec(10),
			TotalShort:      sdk.NewDec(10),
		},
		"ueth:unusd": {
			Pair:            "ueth:unusd",
			BaseReserve:     sdk.NewDec(10),
			QuoteReserve:    sdk.NewDec(10),
			SqrtDepth:       sdk.NewDec(10),
			PriceMultiplier: sdk.NewDec(10),
			TotalLong:       sdk.NewDec(10),
			TotalShort:      sdk.NewDec(10),
		},
	}

	s.bot.State.Prices = map[string]fbot.Prices{
		"ubtc:unusd": {
			IndexPrice: sdk.NewDec(100),
			MarkPrice:  sdk.NewDec(125),
		},
		"ueth:unusd": {
			IndexPrice: sdk.NewDec(100),
			MarkPrice:  sdk.NewDec(125),
		},
	}

	var quoteToMovePrice = fbot.QuoteNeededToMovePrice(*s.bot)

	btcQp, err := common.SqrtDec(sdk.NewDec(100).Quo(sdk.NewDec(125)))
	if err != nil {
		fmt.Println(err)
	}
	btcReserve := sdk.NewDec(10)

	ethQp := btcQp
	ethReserve := btcReserve

	s.Equal(quoteToMovePrice["ubtc:unusd"], btcReserve.Quo(btcQp).Sub(btcReserve).Mul(sdk.NewDec(-1)))
	s.NotNil(quoteToMovePrice["ueth:unusd"], ethReserve.Quo(ethQp).Sub(ethReserve).Mul(sdk.NewDec(-1)))
}
