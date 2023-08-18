package fbot_test

import (
	"context"
	fbot "fbot/bot"
	"fmt"
	"os"
	"reflect"
	"testing"

	"cosmossdk.io/math"
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	perpTypes "github.com/NibiruChain/nibiru/x/perp/v2/types"
	"github.com/Unique-Divine/gonibi"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	hex "encoding/hex"

	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	tmconfig "github.com/cometbft/cometbft/config"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
)

type BotSuite struct {
	suite.Suite
	bot     *fbot.Bot
	ctx     context.Context
	address sdk.AccAddress
	chain   *BlockChain
	config  *fbot.BotConfig
}

func TestBot(t *testing.T) {
	suite.Run(t, new(BotSuite))
}

func (s *BotSuite) Run(t *testing.T) {
	err := fbot.Run(s.bot)
	s.NoError(err)
}

func (s *BotSuite) RunTestInitConfig(t *testing.T) {

	//mnemonic, err := MakeValidMnemonic()
	//s.NoError(err)
	for _, tc := range []struct {
		name               string
		newConfig          fbot.BotConfig
		oldConfig          fbot.BotConfig
		numFieldsNewConfig int
	}{
		{
			name: "FILLED ENV",
			oldConfig: fbot.BotConfig{
				MNEMONIC:       "",
				CHAIN_ID:       "nibiru-localnet-0",
				GRPC_ENDPOINT:  "localhost:9090",
				TMRPC_ENDPOINT: "http://localhost:26657",
			},
			newConfig: fbot.BotConfig{
				MNEMONIC:       "",
				CHAIN_ID:       "",
				GRPC_ENDPOINT:  "",
				TMRPC_ENDPOINT: "",
			},
			numFieldsNewConfig: 0,
		},
		{
			name: "BOTH ENV MISSING FIELDS",
			oldConfig: fbot.BotConfig{
				MNEMONIC:       "",
				CHAIN_ID:       "",
				GRPC_ENDPOINT:  "localhost:9090",
				TMRPC_ENDPOINT: "",
			},
			newConfig: fbot.BotConfig{
				MNEMONIC:       "",
				CHAIN_ID:       "nibiru-localnet-0",
				GRPC_ENDPOINT:  "",
				TMRPC_ENDPOINT: "http://localhost:26657",
			},
			numFieldsNewConfig: 2,
		},
		{
			name: "EMPTY ENV",
			oldConfig: fbot.BotConfig{
				MNEMONIC:       "",
				CHAIN_ID:       "",
				GRPC_ENDPOINT:  "",
				TMRPC_ENDPOINT: "",
			},

			newConfig: fbot.BotConfig{
				MNEMONIC:       "guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host",
				CHAIN_ID:       "nibiru-localnet-0",
				GRPC_ENDPOINT:  "localhost:9090",
				TMRPC_ENDPOINT: "http://localhost:26657",
			},
			numFieldsNewConfig: 4,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {

			err := fbot.DeleteConfigFile()
			s.NoError(err)
			fmt.Printf("\"Get WD\": %v\n", "Get WD")
			fmt.Println(os.Getwd())
			envFile, _ := os.Create(fbot.EnvFilePath())
			envFile.WriteString("MNEMONIC=" + "\"" + tc.oldConfig.MNEMONIC + "\"" + "\n")
			envFile.WriteString("CHAIN_ID=" + "\"" + tc.oldConfig.CHAIN_ID + "\"" + "\n")
			envFile.WriteString("GRPC_ENDPOINT=" + "\"" + tc.oldConfig.GRPC_ENDPOINT + "\"" + "\n")
			envFile.WriteString("TMRPC_ENDPOINT=" + "\"" + tc.oldConfig.TMRPC_ENDPOINT + "\"" + "\n")

			config, err := fbot.Load()
			s.NoError(err)

			numConfigFields := 0
			configVal := reflect.ValueOf(*config)
			for i := 0; i < configVal.NumField(); i++ {
				if len(configVal.Field(i).String()) != 0 {
					numConfigFields++
				}
			}

			fbot.DeleteConfigFile()

			newEnvFile, _ := os.Create(fbot.EnvFilePath())
			newEnvFile.WriteString("MNEMONIC=" + "\"" + tc.newConfig.MNEMONIC + "\"" + "\n")
			newEnvFile.WriteString("CHAIN_ID=" + "\"" + tc.newConfig.CHAIN_ID + "\"" + "\n")
			newEnvFile.WriteString("GRPC_ENDPOINT=" + "\"" + tc.newConfig.GRPC_ENDPOINT + "\"" + "\n")
			newEnvFile.WriteString("TMRPC_ENDPOINT=" + "\"" + tc.newConfig.TMRPC_ENDPOINT + "\"" + "\n")

			newConfig, err := config.Save()

			numNewConfigFields := 0
			newConfigVal := reflect.ValueOf(*newConfig)
			for i := 0; i < newConfigVal.NumField(); i++ {
				if len(newConfigVal.Field(i).String()) != 0 {
					numNewConfigFields++
				}
			}
			//s.Equal(numConfigFields+tc.numFieldsNewConfig, numNewConfigFields)

			s.NoError(err)

		})

	}
}

func MakeValidMnemonic() (string, error) {
	kring := gonibi.NewKeyring()
	_, mnemonic, err := kring.NewMnemonic(
		/* uid */ "10",
		/* language */ keyring.English,
		/* hdPath */ sdk.FullFundraiserPath,
		/* big39Passphrase */ "",
		/* algo */ hd.Secp256k1,
	)

	return mnemonic, err
}

func (s *BotSuite) RunTestCheckConfig(t *testing.T) {

	// mnemonic, err := MakeValidMnemonic()
	// s.NoError(err)

	config, err := fbot.Load()
	s.NoError(err)

	s.config = config
	//s.config.MNEMONIC = mnemonic

	err = s.config.CheckConfig()
	s.NoError(err)
}

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
	gosdk    *gonibi.NibiruClient
	grpcConn *grpc.ClientConn
	cfg      *cli.Config
	network  *cli.Network
	val      *cli.Validator
}

func SetupChain(t *testing.T) *BlockChain {

	chain := new(BlockChain)
	app.SetPrefixes(app.AccountAddressPrefix)
	encConfig := app.MakeEncodingConfig()
	genState := genesis.NewTestGenesisState(encConfig)
	genState = genesis.AddPerpV2Genesis(genState)

	cliCfg := cli.BuildNetworkConfig(genState)

	chain.cfg = &cliCfg
	chain.cfg.NumValidators = 1

	network, err := cli.New(
		t,
		t.TempDir(),
		*chain.cfg,
	)
	require.NoError(t, err)
	chain.network = network
	require.NoError(t, chain.network.WaitForNextBlock())

	chain.val = chain.network.Validators[0]
	AbsorbServerConfig(chain.cfg, chain.val.AppConfig)
	AbsorbTmConfig(chain.cfg, chain.val.Ctx.Config)
	chain.ConnectGrpc(t)

	return chain
}

func AbsorbTmConfig(
	cfg *cli.Config, tmCfg *tmconfig.Config,
) *cli.Config {
	cfg.RPCAddress = tmCfg.RPC.ListenAddress
	return cfg
}

func AbsorbServerConfig(
	cfg *cli.Config, srvCfg *serverconfig.Config,
) *cli.Config {
	cfg.GRPCAddress = srvCfg.GRPC.Address
	cfg.APIAddress = srvCfg.API.Address
	return cfg
}

func (chain *BlockChain) ConnectGrpc(t *testing.T) {
	grpcUrl := chain.val.AppConfig.GRPC.Address
	grpcConn, err := gonibi.GetGRPCConnection(
		grpcUrl, true, 5,
	)
	require.NoError(t, err)
	require.NotNil(t, grpcConn)
	chain.grpcConn = grpcConn
}

func (s *BotSuite) TestBotSuite() {
	s.T().Run("RunTestInitConfig", s.RunTestInitConfig)
	s.T().Run("RunTestCheckConfig", s.RunTestCheckConfig)
	s.chain = SetupChain(s.T())
	s.SetupGoSdk()

	s.T().Run("RunTestPopulatePrices", s.RunTestFetchPrices)
	s.T().Run("RunTestQuoteNeededToMovePrice", s.RunTestQuoteNeededToMovePrice)
	s.T().Run("RunTestPopWalletCoins", s.RunTestPopWalletCoins)
	s.T().Run("RunTestGetBlockHeight", s.RunTestGetBlockHeight)
	s.T().Run("RunTestOpenPosition", s.RunTestOpenPosition)
	s.T().Run("RunTestClosePosition", s.RunTestClosePosition)
}

func (s *BotSuite) TearDownAllSuite() {
	s.bot.DB.ClearDB()
	s.bot.DB.DeleteDB()
	s.chain.network.Cleanup()
}

func (s *BotSuite) SetupGoSdk() {
	s.ctx = context.Background()

	keyRecord, err := s.chain.val.ClientCtx.Keyring.KeyByAddress(
		s.chain.val.Address)

	s.NoError(err)

	bot, err := fbot.NewBot(
		fbot.BotArgs{
			ChainId:     s.chain.cfg.ChainID,
			GrpcEndpt:   s.chain.cfg.GRPCAddress,
			RpcEndpt:    s.chain.val.RPCAddress,
			Mnemonic:    "",
			UseMnemonic: false,
			KeyName:     keyRecord.Name,
		},
	)

	s.NoError(err)
	s.bot = bot
	s.bot.Gosdk.Keyring = s.chain.val.ClientCtx.Keyring

	s.address = s.chain.val.Address

}

func (s *BotSuite) RunTestPopulateAmms(t *testing.T) {

	resp, err := s.bot.Gosdk.Querier.Perp.QueryMarkets(
		s.ctx, &perpTypes.QueryMarketsRequest{},
	)
	s.NoErrorf(err, "Perp Resp: %v", resp)

	bot := s.bot

	bot.PopulateAmms(resp)
	s.NotNil(bot.State.Amms["ubtc:unusd"])
	s.NotNil(bot.State.Amms["ueth:unusd"])

}

func (s *BotSuite) RunTestFetchPrices(t *testing.T) {
	s.NotNil(s.bot.DB)
	err := s.bot.FetchNewPrices(s.ctx)
	s.NoError(err)

}

func (s *BotSuite) RunTestUpdateTradeBalance(t *testing.T) {

	s.bot.UpdateTradeBalance(fbot.OpenOrder, "ubtc:unusd", sdk.NewInt(100))

	s.Equal(t, s.bot.State.PortfolioBalances.Balances.TradedBalances["ubtc:unusd"], 100)

	s.bot.UpdateTradeBalance(fbot.CloseOrder, "ubtc:unusd", sdk.NewInt(100))

	s.Equal(t, s.bot.State.PortfolioBalances.Balances.TradedBalances["ubtc:unusd"], 0)

}

func (s *BotSuite) RunTestPopWalletCoins(t *testing.T) {
	balancesResp, err := s.bot.State.PortfolioBalances.Balances.QueryWalletCoins(s.ctx, s.address, s.bot.Gosdk.GrpcClient)
	s.NotNil(balancesResp)
	s.NoError(err)

	s.bot.State.PortfolioBalances.Balances.PopWalletCoins(balancesResp)
}

func (s *BotSuite) RunTestEvaluateTradeAction(t *testing.T) {
	for _, tc := range []struct {
		name        string
		quoteAmount sdk.Int
		amm         perpTypes.AMM
		posExists   bool
		position    fbot.CurrPosStats
		tradeAction fbot.TradeAction
	}{
		{
			name:        "Open Order",
			quoteAmount: sdk.NewInt(3500), amm: perpTypes.AMM{
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
			quoteAmount: sdk.NewInt(350), amm: perpTypes.AMM{
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
			quoteAmount: sdk.NewInt(350), amm: perpTypes.AMM{
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
			quoteAmount: sdk.NewInt(350), amm: perpTypes.AMM{
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

	s.bot.State.Amms = map[string]fbot.AmmFields{
		"ubtc:unusd": {
			Markets: perpTypes.AMM{
				Pair:            "ubtc:unusd",
				BaseReserve:     sdk.NewDec(10),
				QuoteReserve:    sdk.NewDec(10),
				SqrtDepth:       sdk.NewDec(10),
				PriceMultiplier: sdk.NewDec(10),
				TotalLong:       sdk.NewDec(10),
				TotalShort:      sdk.NewDec(10),
			},
			Bias: sdk.NewDec(10),
		},
		"ueth:unusd": {
			Markets: perpTypes.AMM{
				Pair:            "ueth:unusd",
				BaseReserve:     sdk.NewDec(10),
				QuoteReserve:    sdk.NewDec(10),
				SqrtDepth:       sdk.NewDec(10),
				PriceMultiplier: sdk.NewDec(10),
				TotalLong:       sdk.NewDec(10),
				TotalShort:      sdk.NewDec(10),
			},
			Bias: sdk.NewDec(10),
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

	quoteToMovePrice, err := s.bot.QuoteNeededToMovePrice()
	s.NoError(err)
	btcQp, err := common.SqrtDec(sdk.NewDec(100).Quo(sdk.NewDec(125)))

	s.NoError(err)

	btcReserve := sdk.NewDec(10)

	ethQp := btcQp
	ethReserve := btcReserve

	s.Equal(quoteToMovePrice["ubtc:unusd"], btcReserve.Quo(btcQp).Sub(btcReserve).Mul(sdk.NewDec(-1)))
	s.NotNil(quoteToMovePrice["ueth:unusd"], ethReserve.Quo(ethQp).Sub(ethReserve).Mul(sdk.NewDec(-1)))
}

func (s *BotSuite) RunTestGetBlockHeight(t *testing.T) {
	_, err := s.bot.GetBlockHeight(s.ctx, s.chain.val.RPCAddress)
	s.NoError(err)
}

func (s *BotSuite) RunTestOpenPosition(t *testing.T) {

	addr, err := s.bot.GetAddress()
	s.NoError(err)

	s.NoError(s.chain.network.WaitForNextBlock())

	resp, err := s.bot.OpenPosition(addr, math.NewInt(10),
		math.LegacyNewDec(1), "ubtc:unusd", s.ctx)
	s.NoError(err)

	s.NoError(s.chain.network.WaitForNextBlock())

	hashBz, err := hex.DecodeString(resp.TxHash)
	txHashQueryResp, err := s.bot.Gosdk.CometRPC.Tx(s.ctx, hashBz, false)
	s.NoErrorf(err, "Query Response: %s", txHashQueryResp.Hash)
	s.NotNil(txHashQueryResp)
	s.FetchPositions(t)

}

func (s *BotSuite) RunTestClosePosition(t *testing.T) {

	s.NoError(s.chain.network.WaitForNextBlock())
	_, err := s.bot.ClosePosition(s.address, "ubtc:unusd", s.ctx)
	s.NoError(err)
	s.FetchPositions(t)

}

func (s *BotSuite) FetchPositions(t *testing.T) {
	s.NotNil(s.bot.DB)

	err := s.bot.FetchAndPopPositionsDB(s.address, s.ctx)
	s.NoError(err)
	positions, err := s.bot.DB.QueryPositionTable()
	s.NoError(err)
	s.NotNil(positions)

}
