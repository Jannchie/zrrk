package main

import (
	"log"
	"os"
	"sync"

	"jannchie.com/zrrk/zrrk"
)

func main() {
	log.SetFlags(log.LstdFlags)
	log.SetOutput(os.Stdout)
	// bot := zrrk.Default(7777)
	roomIDs := []int{
		545068,
	}
	m := sync.Mutex{}
	for _, i := range roomIDs {
		bot := zrrk.Default(i)
		bot.Lock = &m
		// bot.AddPlugin(todayrp.New())
		// bot.AddPlugin(enterc.New())
		go bot.Connect()
	}
	done := make(chan struct{})
	<-done
}
