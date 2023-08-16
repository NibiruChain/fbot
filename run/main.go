package main

import (
	"context"
	"fbot"
	"fmt"
	///	"cli"
)

// go build -o bot main.go
func main() {
}

type Runner struct {
	*fbot.Bot
}

// After setup, check if already trading from account
// Start trade and tell user if broke
// PauseBot() Stop running
// EndBot() Exit all positions

type BotAPI interface {
	SetConfig(config fbot.BotConfig) error
	StartBot() error
	PauseBot() error
	EndBot() error
}

var _ BotAPI = (*Runner)(nil)

var bot fbot.Bot

func (runner *Runner) SetConfig(config fbot.BotConfig) error {

	bot, err := fbot.NewBot(
		fbot.BotArgs{
			ChainId:     config.CHAIN_ID,
			GrpcEndpt:   config.GRPC_ENDPOINT,
			RpcEndpt:    config.TMRPC_ENDPOINT,
			Mnemonic:    config.MNEMONIC,
			UseMnemonic: false,
			KeyName:     "",
		},
	)

	runner.Bot = bot

	if err != nil {
		return err
	}

	return nil

}

func (runner *Runner) StartBot() error {

	return nil
}

func (runner *Runner) PauseBot() error {
	return nil
}

func (runner *Runner) EndBot() error {
	addr, err := bot.GetAddress()

	if err != nil {
		return err
	}

	positionPairs := []string{}

	for pair := range bot.State.Positions {
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

func QueryTables(bot *fbot.Bot) {
	tables, _ := bot.DB.QueryAllTablesToJson()
	fmt.Print(tables)
}

func QueryTablesByBlock(bot *fbot.Bot, height int64) {

}

func ClearDB(bot *fbot.Bot) {
	bot.DB.ClearDB()
}
