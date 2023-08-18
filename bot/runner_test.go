package fbot_test

import (
	fbot "fbot/bot"
	"fmt"
)

func (s *BotSuite) TestMain() {

	runner := fbot.Runner{
		Bot: s.bot,
		Server: &fbot.Server{
			StartCh:  make(chan bool),
			StopCh:   make(chan bool),
			PauseCh:  make(chan bool),
			IsPaused: false,
		},
	}
	fmt.Printf("runner: %v\n", runner)

}
