package cli

import (
	fbot "fbot/bot"
	"log"
	"os"

	"github.com/urfave/cli"
)

func CliMain() {

	app := cli.NewApp()
	app.Name = "Funding Bot"

	runner := fbot.Runner{
		Bot: &fbot.Bot{},
		Server: &fbot.Server{
			StartCh:  make(chan bool),
			StopCh:   make(chan bool),
			PauseCh:  make(chan bool),
			IsPaused: false,
		},
	}

	botConfig, err := fbot.Load()

	if err != nil {
		log.Fatal(err)
	}

	err = runner.SetConfig(*botConfig)

	if err != nil {
		panic(err)
	}

	app.Commands = []cli.Command{
		{
			// go run main.go start
			Name: "start",
			Action: func(c *cli.Context) error {
				runner.Server.StartCh <- true
				runner.HandleChannels()
				return nil
			},
		},
		{
			// go run main.go pause
			Name: "pause",
			Action: func(c *cli.Context) error {
				runner.Server.PauseCh <- true
				runner.HandleChannels()
				return nil
			},
		},
		{
			// go run main.go end
			Name: "end",
			Action: func(c *cli.Context) error {
				runner.Server.StopCh <- true
				runner.HandleChannels()
				return nil
			},
		},
	}

	err = app.Run(os.Args)

	if err != nil {
		panic(err)
	}
}
