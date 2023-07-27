package fbot_test

import (
	"bufio"
	"context"
	"fbot"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	//"github.com/NibiruChain/nibiru/x/common/testutil/cli"
	perpTypes "github.com/NibiruChain/nibiru/x/perp/v2/types"
	"github.com/Unique-Divine/gonibi"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BotSuite struct {
	suite.Suite
	bot   *fbot.Bot
	gosdk *gonibi.NibiruClient
	ctx   context.Context
}

func TestBot(t *testing.T) {
	suite.Run(t, new(BotSuite))
}

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

func TestShouldTrade(t *testing.T) {
	var shouldTrade = fbot.ShouldTrade(sdk.NewDec(50), perpTypes.AMM{
		Pair:            "ueth:unusd",
		BaseReserve:     sdk.NewDec(10),
		QuoteReserve:    sdk.NewDec(10),
		SqrtDepth:       sdk.NewDec(10),
		PriceMultiplier: sdk.NewDec(10),
		TotalLong:       sdk.NewDec(10),
		TotalShort:      sdk.NewDec(10),
	})

	require.True(t, shouldTrade)
}

func (s *BotSuite) TestBotSuite() {
	s.SetupGoSdk()
	s.T().Run("RunTestPopulatePrices", s.RunTestFetchPrices)
	s.T().Run("RunTestPopulateAmms", s.RunTestPopulateAmms)
	s.T().Run("RunQuoteNeededToMovePrice", s.RunQuoteNeededToMovePrice)
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

// func GeneratePrivKey(nodeDirName string) {
// 	mnemonic := ""
// 	nodeDirName := s.T().TempDir()

// 	addr, secret, err := sdktestutil.GenerateSaveCoinKey(
// 		kb, nodeDirName, mnemonic, true, algo,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}
// }

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

// func (s *BotSuite) RunFetchPositions(t *testing.T) {

// 	s.gosdk.Query.Perp.QueryPositions(s.ctx, &perpTypes.QueryPositionsRequest{
// 		Trader: "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
// 	})

// }

// 	s.NoErrorf(err, "Positions Resp: %v", positionsResp)

// 	//	var bot fbot.Bot = fbot.NewBot()

// }

func (s *BotSuite) RunQuoteNeededToMovePrice(t *testing.T) {

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
