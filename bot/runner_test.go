package fbot_test

import (
	fbot "fbot/bot"
)

func (s *BotSuite) TestMain() {

	_ = fbot.Runner{
		Bot: s.bot,
		Server: &fbot.Server{
			StartCh:  false,
			StopCh:   false,
			PauseCh:  false,
			IsPaused: false,
		},
	}

}
