package fbot

import (
	"context"
	"fmt"
	"log"
)

// go build -o bot main.go

type Runner struct {
	*Bot
	Server *Server
}

type Server struct {
	StartCh  bool
	StopCh   bool
	PauseCh  bool
	IsPaused bool
}

type BotAPI interface {
	SetConfig(config BotConfig, keyname string) error
	StartBot() error
	PauseBot() error
	EndBot() error
}

var _ BotAPI = (*Runner)(nil)

func (runner *Runner) SetConfig(config BotConfig, keyname string) error {

	bot, err := NewBot(
		BotArgs{
			ChainId:     config.CHAIN_ID,
			GrpcEndpt:   config.GRPC_ENDPOINT,
			RpcEndpt:    config.TMRPC_ENDPOINT,
			Mnemonic:    config.MNEMONIC,
			UseMnemonic: true,
			KeyName:     keyname,
		},
	)

	runner.Bot = bot

	if err != nil {
		return err
	}

	return nil
}

func (runner *Runner) StartBot() error {

	err := Run(runner.Bot)

	if err != nil {
		log.Fatalf("Cannot run bot: %v", err)
	}

	posTables, err := runner.Bot.DB.QueryPositionTable()
	fmt.Println("Position: ", posTables)

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

	posTables, err := runner.Bot.DB.QueryPositionTable()
	fmt.Println("Position: ", posTables)

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
