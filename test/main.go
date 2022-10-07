package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"

	"github.com/jannchie/zrrk/zrrk"
	"github.com/jannchie/zrrk/zrrk/plugin/gift"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	log.SetFlags(log.LstdFlags)
	log.SetOutput(os.Stdout)
	giftPlugin := gift.New()
	syncMap := sync.Map{}

	var roomID int
	m := sync.Mutex{}
	bot := zrrk.Default(&m, &zrrk.BotConfig{
		RoomID:     198297,
		StayMinHot: 0,
		LogLevel:   zrrk.LogDebug,
	})
	bot.AddPlugin(giftPlugin)
	syncMap.Store(roomID, bot)
	defer syncMap.Delete(roomID)
	bot.Connect()
}
