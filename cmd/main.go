package main

import (
	"log"

	"jannchie.com/zrrk/zrrk"
	"jannchie.com/zrrk/zrrk/plugin/todayrp"
)

func main() {
	log.SetFlags(log.LstdFlags)
	bot := zrrk.Default(545068)
	bot.AddPlugin(todayrp.New())
	bot.Connect()
	done := make(chan struct{})
	<-done
}
