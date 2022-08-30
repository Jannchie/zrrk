package gift

import (
	"log"
	"os"
	"time"

	"github.com/jannchie/zrrk/zrrk"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type GiftPlugin struct {
	RoomID   int
	DB       *gorm.DB
	giftChan chan LiveRoomGift `gorm:"-"`
}

type LiveRoomGift struct {
	ID        int64     `gorm:"primaryKey"`
	RoomID    int       `gorm:"index"`
	GiftID    int       ``
	Price     int       ``
	Count     int       `gorm:"default:0"`
	UID       int       ``
	CreatedAt time.Time ``
}

func New() *GiftPlugin {
	dsn := os.Getenv("BILIBILI_DSN")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Println(err)
	}
	p := GiftPlugin{
		DB:       db,
		giftChan: make(chan LiveRoomGift, 100),
	}
	go func() {
		var giftArray []LiveRoomGift
		ticker := time.NewTicker(time.Second * 1)
		defer ticker.Stop()
		for {
			select {
			case gift := <-p.giftChan:
				giftArray = append(giftArray, gift)
			case <-ticker.C:
				if len(giftArray) > 0 {
					if err = p.DB.Create(&giftArray).Error; err == nil {
						giftArray = []LiveRoomGift{}
					} else {
						log.Println(err)
					}
				}
			}
		}
	}()
	return &p
}

func (p *GiftPlugin) GetDescriptions() []string {
	return []string{}
}

func (p *GiftPlugin) SetRoom(id int) {
	p.RoomID = id
}

func (p *GiftPlugin) HandleData(input interface{}, channel chan<- string) {
	data, ok := input.(zrrk.GiftData)
	if !ok {
		return
	}
	if data.Gift.Currency == "GOLD" {
		var liveRoomGift = LiveRoomGift{
			RoomID: data.RoomID,
			GiftID: data.Gift.ID,
			Count:  data.Gift.Count,
			Price:  data.Gift.Price,
			UID:    data.User.UID,
		}
		p.giftChan <- liveRoomGift
	}
}
