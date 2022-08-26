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
	RoomID int
	DB     *gorm.DB
}

type LiveRoomGift struct {
	ID        int       `gorm:"primaryKey"`
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
	p := GiftPlugin{DB: db}
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
			RoomID: p.RoomID,
			GiftID: data.Gift.ID,
			Count:  data.Gift.Count,
			Price:  data.Gift.Price,
			UID:    data.User.UID,
		}
		_ = p.DB.Create(&liveRoomGift)
	}
}
