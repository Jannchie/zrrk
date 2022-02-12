package main

import (
	"log"
	"os"

	"jannchie.com/zrrk/zrrk"
	"jannchie.com/zrrk/zrrk/plugin/enterc"
	"jannchie.com/zrrk/zrrk/plugin/todayrp"
)

func main() {
	log.SetFlags(log.LstdFlags)
	log.SetOutput(os.Stdout)
	bot := zrrk.Default(11365)
	// bot := zrrk.Default(422915)
	bot.AddPlugin(todayrp.New())
	bot.AddPlugin(enterc.New())
	bot.Connect()
	done := make(chan struct{})
	<-done
}
