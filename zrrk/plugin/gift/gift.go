package gift

import (
	"log"

	"gorm.io/gorm"
	"jannchie.com/zrrk/zrrk"
)

var DB *gorm.DB

type GiftPlugin struct {
	RoomID int
}

func New() *GiftPlugin {
	p := GiftPlugin{}
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
		log.Printf("+RMB: %.2f\n", float64(data.Gift.Price)/1000)
	}
}
