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
	// bot := zrrk.Default(11365)
	m := sync.Mutex{}
	for i := 1; i <= 3000; i++ {
		bot := zrrk.Default(i)
		bot.Lock = &m
		// bot.AddPlugin(todayrp.New())
		// bot.AddPlugin(enterc.New())
		go bot.Connect()
	}
	done := make(chan struct{})
	<-done
}
