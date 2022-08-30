package aggregate

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jannchie/zrrk/zrrk/plugin/gift"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type LiveRoomGiftAggregation struct {
	ID        int64     `json:"-" gorm:"primaryKey"`
	Price     int       `json:"price"`
	Count     int       `json:"count" gorm:"default:0"`
	RoomID    int       `json:"room_id" gorm:"index"`
	Timestamp time.Time `json:"timestamp" gorm:"index"`
}

func Aggregation() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	log.SetFlags(log.LstdFlags)
	log.SetOutput(os.Stdout)
	dsn := os.Getenv("BILIBILI_DSN")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Panic(err)
	}
	db.AutoMigrate(&LiveRoomGiftAggregation{})
	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ticker.C:
			rows, err := db.Raw("select distinct room_id from live_room_gifts where created_at < ?", time.Now().Add(-time.Minute*10)).Rows()
			if err != nil {
				log.Panic(err)
			}
			for rows.Next() {
				var roomID int
				err := rows.Scan(&roomID)
				if err != nil {
					log.Println(err)
					continue
				}
				roomRows, err := db.Raw("select * from live_room_gifts where room_id = ? and created_at < ? order by created_at", roomID, time.Now().Add(-time.Minute*10)).Rows()
				var startTime time.Time
				var maxTime time.Time
				dataList := []LiveRoomGiftAggregation{}
				data := LiveRoomGiftAggregation{}
				shouldDelete := []gift.LiveRoomGift{}
				for roomRows.Next() {
					var roomGift gift.LiveRoomGift
					err := db.ScanRows(roomRows, &roomGift)
					if err != nil {
						log.Println(err)
						continue
					}
					if startTime.IsZero() {
						startTime = roomGift.CreatedAt.Truncate(time.Minute * 5)
					}
					if roomGift.CreatedAt.Truncate(time.Minute * 5).Equal(startTime) {
						data.Price += (roomGift.Price * roomGift.Count)
						data.Count += roomGift.Count
						data.RoomID = roomGift.RoomID
						data.Timestamp = startTime
					} else {
						startTime = roomGift.CreatedAt.Truncate(time.Minute * 5)
						dataList = append(dataList, data)
						data = LiveRoomGiftAggregation{}
						data.Price += (roomGift.Price * roomGift.Count)
						data.Count += roomGift.Count
						data.RoomID = roomGift.RoomID
						data.Timestamp = startTime
					}
					shouldDelete = append(shouldDelete, roomGift)
					if maxTime.IsZero() || roomGift.CreatedAt.After(maxTime) {
						maxTime = roomGift.CreatedAt
					}
				}
				if !data.Timestamp.IsZero() {
					dataList = append(dataList, data)
				}
				if len(shouldDelete) == 0 {
					continue
				}
				if err := db.Create(dataList).Error; err != nil {
					log.Println(err)
				}
				if err := db.Where("created_at <= ? and room_id = ?", maxTime, roomID).Delete(&gift.LiveRoomGift{}).Error; err != nil {
					log.Println(err)
				}
			}
		}
	}

}
