package main

import (
	"log"

	"jannchie.com/zrrk/zrrk"
	"jannchie.com/zrrk/zrrk/plugin/todayrp"
)

func main() {
	log.SetFlags(log.LstdFlags)
	bot := zrrk.Default(22907643)
	bot.AddPlugin(todayrp.New())
	bot.Connect()
	done := make(chan struct{})
	<-done
}
