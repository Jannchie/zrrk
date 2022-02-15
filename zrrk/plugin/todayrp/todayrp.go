package todayrp

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"jannchie.com/zrrk/zrrk"
)

var DB *gorm.DB

type TodayRP struct {
	ID        int `gorm:"primaryKey"`
	UID       int `gorm:"index"`
	RP        int
	CreatedAt time.Time `gorm:"index"`
}

type TodayRPPlugin struct {
	RoomID int
}

func New() *TodayRPPlugin {
	DB, _ = gorm.Open(sqlite.Open("./test.db"), &gorm.Config{})
	DB.AutoMigrate(&TodayRP{})
	p := TodayRPPlugin{}
	return &p
}

func (p *TodayRPPlugin) SetRoom(id int) {
	p.RoomID = id
}
func (p *TodayRPPlugin) HandleData(input interface{}, channel chan<- string) {
	data, ok := input.(zrrk.DanmakuData)
	if !ok {
		return
	}
	if !strings.Contains(data.Text, "06") {
		return
	}
	if !zrrk.ContainStrings(data.Text, "RP", "rp", "人品", "求签", "抽签", "运") {
		return
	}
	if data.User.UID == 0 {
		return
	}
	var rp TodayRP
	_ = DB.Limit(1).Order("created_at DESC").Find(&rp, "uid = ?", data.User.UID)
	isSameDay := zrrk.IsSameDay(rp.CreatedAt)
	if isSameDay {
		msg := fmt.Sprintf("%s今天已经测过，今天的运势是%d · %s。", data.User.Name, rp.RP, getRPLevel(rp.RP))
		channel <- msg
		return
	}
	rp.UID = data.User.UID
	rp.RP = int(rand.NormFloat64()*50 + 50)
	DB.Create(&TodayRP{
		UID: rp.UID,
		RP:  rp.RP,
	})
	msg := fmt.Sprintf("%s今天的运势是%d · %s。", data.User.Name, rp.RP, getRPLevel(rp.RP))
	channel <- msg
}

func getRPLevel(rp int) string {
	switch {
	case rp <= 0:
		return "大凶"
	case 0 < rp && rp <= 50:
		return "凶"
	case 50 < rp && rp <= 60:
		return "末吉"
	case 60 < rp && rp <= 70:
		return "吉"
	case 70 < rp && rp <= 80:
		return "小吉"
	case 80 < rp && rp < 90:
		return "中吉"
	case 90 < rp && rp < 100:
		return "中吉"
	}
	return "大吉"
}
