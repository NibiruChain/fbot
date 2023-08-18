package cli

import (
	fbot "fbot/bot"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/urfave/cli"
)

// fix .env placement in tests

func CliMain() {

	app := cli.NewApp()
	app.Name = "Funding Bot"

	runner := &fbot.Runner{
		Bot: &fbot.Bot{},
		Server: &fbot.Server{
			StartCh:  false,
			StopCh:   false,
			PauseCh:  false,
			IsPaused: false,
		},
	}

	botConfig, err := fbot.Load()

	if err != nil {
		log.Fatal(err)
	}

	err = runner.SetConfig(*botConfig, "bot")

	if err != nil {
		panic(err)
	}

	app.Commands = []cli.Command{
		{
			Name: "start",
			Action: func(c *cli.Context) error {
				startTrade(runner)
				return nil
			},
		},
		{
			Name: "clear-db",
			Action: func(c *cli.Context) error {
				runner.Bot.DB.ClearDB()
				return nil
			},
		},
	}

	err = app.Run(os.Args)

	if err != nil {
		panic(err)
	}

}

func startTrade(runner *fbot.Runner) {
	ch := make(chan string)

	runner.Server.StartCh = true

	for runner.Server.StartCh {

		go func() {
			var input string
			time.Sleep(2 * time.Second)
			fmt.Scanln(&input)
			ch <- input
		}()

		select {
		case cmd := <-ch:
			if cmd == "pause" {
				runner.Server.StartCh = false
				runner.Server.PauseCh = true
				pauseTrade(runner)
			} else if cmd == "stop" {
				runner.Server.StartCh = false
				runner.Server.StopCh = true
				stopTrade(runner)
			}
		case <-time.After(1 * time.Second):
			runner.StartBot()
			time.Sleep(2 * time.Second)
		}
	}
}

func pauseTrade(runner *fbot.Runner) {
	ch := make(chan string)

	for runner.Server.PauseCh {

		go func() {
			var input string
			time.Sleep(2 * time.Second)
			fmt.Scanln(&input)
			ch <- input
		}()

		select {
		case cmd := <-ch:
			if cmd == "start" {
				runner.Server.StartCh = true
				runner.Server.PauseCh = false
				startTrade(runner)
			} else if cmd == "stop" {
				runner.Server.StopCh = true
				runner.Server.PauseCh = false
				stopTrade(runner)
			}
		case <-time.After(1 * time.Second):
			fmt.Println("Pausing Bot")
			time.Sleep(2 * time.Second)

		}
	}
}

func stopTrade(runner *fbot.Runner) {

	fmt.Println("Exiting positions")
	err := runner.EndBot()
	if err != nil {
		log.Fatalf("Failed to exit positions")
	}

}
