package enterc

import (
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"jannchie.com/zrrk/zrrk"
)

var DB *gorm.DB

type EnterCounter struct {
	UID       int       `gorm:"primaryKey"`
	RoomID    int       `gorm:"index"`
	CreatedAt time.Time ``
	Count     int       `gorm:"default:0"`
}
type EnterRecord struct {
	ID        int       `gorm:"primaryKey"`
	RoomID    int       `gorm:"index"`
	UID       int       ``
	CreatedAt time.Time ``
}
type FollowRecord struct {
	ID        int       `gorm:"primaryKey"`
	RoomID    int       `gorm:"index"`
	UID       int       ``
	CreatedAt time.Time ``
}
type EnterCounterPlugin struct {
	RoomID int
}

func New() *EnterCounterPlugin {
	p := EnterCounterPlugin{}
	DB, _ = gorm.Open(sqlite.Open("./test.db"), &gorm.Config{})
	DB.AutoMigrate(&EnterCounter{}, &EnterRecord{}, &FollowRecord{})
	return &p
}

func (p *EnterCounterPlugin) SetRoom(id int) {
	p.RoomID = id
}

func (p *EnterCounterPlugin) HandleData(input interface{}, channel chan<- string) {
	data, ok := input.(zrrk.InteractData)
	if !ok {
		return
	}
	uid := data.User.UID
	var enterCounter EnterCounter
	DB.Debug().Limit(1).Find(&enterCounter, "uid = ? & room_id = ?", uid, p.RoomID)
	enterCounter.UID = uid
	enterCounter.RoomID = p.RoomID
	switch data.Type {
	case zrrk.INTERACT_ENTER:
		enterCounter.Count++
		if !zrrk.IsSameDay(enterCounter.CreatedAt) {
			if res := DB.Save(&enterCounter); res.Error != nil {
				log.Println(res.Error)
			}
		} else {
			log.Println("same day")
		}
		DB.Save(&EnterRecord{UID: uid, RoomID: p.RoomID, CreatedAt: time.Now()})
	case zrrk.INTERACT_FOLLOW:
		DB.Save(&FollowRecord{UID: uid, RoomID: p.RoomID, CreatedAt: time.Now()})
	}
}

