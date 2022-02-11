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

type TodayRPPlugin struct{}

func New() *TodayRPPlugin {
	p := TodayRPPlugin{}
	return &p
}

func (p *TodayRPPlugin) HandleDanmuData(data zrrk.DanmakuData) string {
	if !strings.Contains(data.Text, "06") {
		return ""
	}
	if !(strings.Contains(data.Text, "RP") || strings.Contains(data.Text, "rp") || strings.Contains(data.Text, "人品")) {
		return ""
	}
	if data.User.UID == 0 {
		return ""
	}
	var rp TodayRP
	_ = DB.Find(&rp, "uid = ?", data.User.UID)
	yp, mp, dp := time.Unix(rp.CreatedAt.Unix(), 0).Date()
	y, m, d := time.Unix(time.Now().Unix(), 0).Date()
	if yp == y && m == mp && d == dp {
		msg := fmt.Sprintf("%s今天已经测过，今天的RP为%d。", data.User.Name, rp.RP)
		return msg
	}
	rp.UID = data.User.UID
	rp.RP = int(rand.NormFloat64()*50 + 50)
	DB.Save(&rp)
	msg := fmt.Sprintf("%s今天的RP是%d。", data.User.Name, rp.RP)
	return msg
}

func init() {
	DB, _ = gorm.Open(sqlite.Open("./test.db"), &gorm.Config{})
	DB.AutoMigrate(&TodayRP{})
}
