package main

import (
	"context"
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
	runningMap := sync.Map{}

	go func() {
		for {
			cnt := 0
			select {
			case <-time.After(time.Second * 10):
				runningMap.Range(func(key, value interface{}) bool {
					if bot, ok := value.(*zrrk.Bot); ok {
						if bot.IsConnecting {
							cnt += 1
						}
					}
					return true
				})
				log.Println("Bot Count:", cnt)
			}
		}
	}()
	go taskSender(&runningMap, `SELECT room_id FROM livers WHERE room_id != 0 AND guard_num > 100`, time.Second/16, 0)
	go taskSender(&runningMap, `SELECT room_id FROM livers WHERE room_id != 0 AND guard_num >= 1 AND guard_num < 100`, time.Second/10, 1)
	go taskSender(&runningMap, `SELECT room_id FROM livers WHERE room_id != 0 AND live_status = 1`, time.Second/5, 1)
	<-ctx.Done()
}

func taskSender(runningMap *sync.Map, sql string, interval time.Duration, stayMinHot int32) {
	dsn := os.Getenv("BILIBILI_DSN")
	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	db.AutoMigrate(&gift.LiveRoomGift{})
	giftPlugin := gift.New()
	for {
		createBotIfNotCreated(db, sql, runningMap, giftPlugin, interval, stayMinHot)
		<-time.After(time.Second * 5)
	}
}

func createBotIfNotCreated(db *gorm.DB, sql string, runningMap *sync.Map, giftPlugin *gift.GiftPlugin, interval time.Duration, stayMinHot int32) {
	ctx := context.Background()
	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		if err := recover(); err != nil {
			log.Println(err)
		}
		<-ctxWithCancel.Done()
		<-time.After(time.Second * 5)
	}()
	rows, _ := db.WithContext(ctxWithCancel).Raw(sql).Rows()
	for rows.Next() {
		var roomID int
		err := rows.Scan(&roomID)
		if err != nil {
			log.Println(err)
			continue
		}
		if _, ok := runningMap.Load(roomID); ok {
			continue
		}
		go func(roomID int) {
			m := sync.Mutex{}
			bot := zrrk.Default(&m, &zrrk.BotConfig{
				RoomID:     roomID,
				StayMinHot: stayMinHot,
				LogLevel:   zrrk.LogErr,
			})
			runningMap.Store(roomID, bot)
			defer func() {
				runningMap.Delete(roomID)
			}()
			bot.AddPlugin(giftPlugin)
			bot.Connect()
		}(roomID)
		<-time.After(interval)
	}
}
