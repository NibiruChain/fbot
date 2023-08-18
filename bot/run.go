package fbot

import (
	"context"
	"fmt"
)

// go build -o bot main.go

type Runner struct {
	*Bot
	Server *Server
}

type Server struct {
	StartCh  chan bool
	StopCh   chan bool
	PauseCh  chan bool
	IsPaused bool
}

// After setup, check if already trading from account
// Start trade and tell user if broke
// PauseBot() Stop running
// EndBot() Exit all positions

type BotAPI interface {
	SetConfig(config BotConfig) error
	StartBot() error
	PauseBot() error
	EndBot() error
}

var _ BotAPI = (*Runner)(nil)

func (runner *Runner) SetConfig(config BotConfig) error {

	bot, err := NewBot(
		BotArgs{
			ChainId:     config.CHAIN_ID,
			GrpcEndpt:   config.GRPC_ENDPOINT,
			RpcEndpt:    config.TMRPC_ENDPOINT,
			Mnemonic:    config.MNEMONIC,
			UseMnemonic: true,
			KeyName:     "",
		},
	)

	runner.Bot = bot

	if err != nil {
		return err
	}

	return nil
}

// function for cases of channels -> choose execution path
func (runner *Runner) HandleChannels() {
	for {
		select {
		case <-runner.Server.StartCh:
			runner.StartBot()
		case <-runner.Server.PauseCh:
			if runner.Server.IsPaused {
				runner.Server.IsPaused = false
				runner.StartBot()
			} else {
				runner.Server.IsPaused = true
				runner.PauseBot()
			}
		case <-runner.Server.StopCh:
			runner.EndBot()
			return
		}
	}
}

func (runner *Runner) StartBot() error {

	return nil
}

func (runner *Runner) PauseBot() error {
	return nil
}

func (runner *Runner) EndBot() error {
	addr, err := runner.Bot.GetAddress()

	if err != nil {
		return err
	}

	positionPairs := []string{}

	for pair := range runner.Bot.State.Positions {
		positionPairs = append(positionPairs, pair)
	}

	ctx := context.Background()

	for _, pair := range positionPairs {
		_, err := runner.Bot.ClosePosition(addr, pair, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func QueryTables(bot *Bot) {
	tables, _ := bot.DB.QueryAllTablesToJson()
	fmt.Print(tables)
}

func QueryTablesByBlock(bot *Bot, height int64) {

}

func ClearDB(bot *Bot) {
	bot.DB.ClearDB()
}
