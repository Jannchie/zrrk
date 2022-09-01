package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"time"

	"github.com/jannchie/zrrk/cmd/aggregate"
	"github.com/jannchie/zrrk/zrrk"
	"github.com/jannchie/zrrk/zrrk/plugin/gift"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.SetFlags(log.LstdFlags)
	log.SetOutput(os.Stdout)
	err := godotenv.Load()
	if err != nil {
		log.Panic(err)
	}
	go func() {
		heartBeatURL := os.Getenv("HEART_BEAT_URL")
		for {
			http.Get(heartBeatURL)
			time.Sleep(time.Second * 5)
		}
	}()

	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	go aggregate.Aggregation()
	ctx := context.Background()
	syncMap := sync.Map{}
	var connectCount uint64
	go func() {
		for {
			select {
			case <-time.After(time.Second * 10):
				log.Println("Running len:", connectCount)
			}
		}
	}()
	go taskSender(&syncMap, &connectCount, `SELECT room_id FROM livers WHERE live_status = 1`, time.Second/32)
	go taskSender(&syncMap, &connectCount, `SELECT room_id FROM livers WHERE live_status = 0`, time.Second/4)
	<-ctx.Done()
}

func taskSender(syncMap *sync.Map, connectCount *uint64, sql string, interval time.Duration) {
	dsn := os.Getenv("BILIBILI_DSN")
	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	db.AutoMigrate(&gift.LiveRoomGift{})
	giftPlugin := gift.New()
	for {
		rows, _ := db.Raw(sql).Rows()
		createBotIfNotCreated(rows, syncMap, giftPlugin, connectCount, interval)
		<-time.After(time.Second * 5)
	}
}

func createBotIfNotCreated(rows *sql.Rows, syncMap *sync.Map, giftPlugin *gift.GiftPlugin, connectCount *uint64, interval time.Duration) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()
	for rows.Next() {
		var roomID int
		err := rows.Scan(&roomID)
		if err != nil {
			log.Println(err)
			continue
		}
		if _, ok := syncMap.Load(roomID); ok {
			continue
		}
		go func(roomID int) {
			m := sync.Mutex{}
			bot := zrrk.Default(&m, &zrrk.BotConfig{
				RoomID:     roomID,
				StayMinHot: 200,
				LogLevel:   zrrk.LogHighLight,
			})
			syncMap.Store(roomID, bot)
			defer syncMap.Delete(roomID)
			bot.AddPlugin(giftPlugin)
			*connectCount++
			bot.Connect()
			*connectCount--
		}(roomID)
		<-time.After(interval)
	}
}
