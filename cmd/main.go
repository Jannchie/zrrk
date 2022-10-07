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
	deletedMap := sync.Map{}

	go func() {
		ticker := time.NewTicker(time.Second * 60 * 30)
		for range ticker.C {
			deletedMap.Range((func(key, value interface{}) bool {
				runningMap.Delete(key)
				return true
			}))
		}
	}()

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
	go taskSender(&runningMap, &deletedMap, `SELECT room_id FROM livers WHERE guard_num > 50`, time.Second/64)
	go taskSender(&runningMap, &deletedMap, `SELECT room_id FROM livers WHERE guard_num > 1 AND live_status = 1`, time.Second/32)
	// go taskSender(&syncMap, `SELECT room_id FROM livers WHERE live_status = 1`, time.Second)
	// go taskSender(&syncMap, `SELECT room_id FROM livers WHERE live_status = 0`, time.Second/4)
	<-ctx.Done()
}

func taskSender(runningMap *sync.Map, deletedMap *sync.Map, sql string, interval time.Duration) {
	dsn := os.Getenv("BILIBILI_DSN")
	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	db.AutoMigrate(&gift.LiveRoomGift{})
	giftPlugin := gift.New()
	for {
		createBotIfNotCreated(db, sql, runningMap, deletedMap, giftPlugin, interval)
		<-time.After(interval)
	}
}

func createBotIfNotCreated(db *gorm.DB, sql string, runningMap *sync.Map, deletedMap *sync.Map, giftPlugin *gift.GiftPlugin, interval time.Duration) {
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
		if _, ok := deletedMap.Load(roomID); ok {
			continue
		}
		go func(roomID int) {
			m := sync.Mutex{}
			bot := zrrk.Default(&m, &zrrk.BotConfig{
				RoomID:     roomID,
				StayMinHot: 1,
				LogLevel:   zrrk.LogErr,
			})
			runningMap.Store(roomID, bot)
			defer runningMap.Delete(roomID)
			bot.AddPlugin(giftPlugin)
			bot.Connect()
			deletedMap.Store(roomID, true)
		}(roomID)
		<-time.After(interval)
	}
}
