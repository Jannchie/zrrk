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
	for _, roomID := range roomIDs {
		bot := zrrk.Default(roomID, &m)
		// bot.AddPlugin(todayrp.New())
		// bot.AddPlugin(enterc.New())
		// bot.AddPlugin(gift.New())
		go bot.Connect()
	}
	done := make(chan struct{})
	<-done
}
