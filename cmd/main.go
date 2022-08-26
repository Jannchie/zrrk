package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"time"

	"github.com/jannchie/zrrk/zrrk"
	"github.com/jannchie/zrrk/zrrk/plugin/gift"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	log.SetFlags(log.LstdFlags)
	log.SetOutput(os.Stdout)

	syncMap := sync.Map{}
	var connectCount uint64

	dsn := os.Getenv("BILIBILI_DSN")
	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	sql := `SELECT room_id FROM livers WHERE live_status = 1`
	giftPlugin := gift.New()

	go func() {
		for {
			select {
			case <-time.After(time.Second * 10):
				log.Println("Running len:", connectCount)
			}
		}
	}()

	for {
		rows, _ := db.Raw(sql).Rows()
		for rows.Next() {
			var roomID int
			err := rows.Scan(&roomID)
			if err != nil {
				log.Println(err)
				continue
			}
			m := sync.Mutex{}
			// if contain
			if _, ok := syncMap.Load(roomID); ok {
				continue
			}
			go func(roomID int) {
				bot := zrrk.Default(&m, &zrrk.BotConfig{
					RoomID:     roomID,
					StayMinHot: 200,
					LogLevel:   zrrk.LogHighLight,
				})
				syncMap.Store(roomID, bot)
				defer syncMap.Delete(roomID)
				bot.AddPlugin(giftPlugin)
				connectCount++
				bot.Connect()
				connectCount--
			}(roomID)
			<-time.After(time.Second / 16)
		}
		<-time.After(time.Second * 5)
	}
}
