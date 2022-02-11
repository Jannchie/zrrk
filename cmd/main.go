package main

import (
	"log"

	"jannchie.com/zeroroku/zrrk"
)

func main() {
	log.SetFlags(log.LstdFlags)
	bot := zrrk.Default(545068)
	bot.Connect()
	done := make(chan struct{})
	<-done
}
