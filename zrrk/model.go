package zrrk

import (
	"fmt"
)

func (d *Medal) String() string {
	if d.Level == 0 {
		return "[            ]"
	}
	length := getStringWidth(d.Title)
	space := ""

	for i := 0; i < 6-length; i++ {
		space += " "
	}
	return fmt.Sprintf("[%s%s Lv.%2d]", d.Title, space, d.Level)
}

type Gift struct {
	ID    int    `json:"giftId"`
	Name  string `json:"giftName"`
	Count int    `json:"count"`
}

type GiftData struct {
	User User `json:"user"`
	Gift Gift `json:"gift"`
}
type Medal struct {
	Title string `json:"title"`
	Level int    `json:"level"`
}
type User struct {
	UID   int       `json:"uid"`
	Name  string    `json:"name"`
	Medal Medal `json:"modal"`
}
type DanmakuData struct {
	User User   `json:"user"`
	Text string `json:"text"`
}
type SCData struct {
	User User   `json:"user"`
	Text string `json:"text"`
}
type InteractData struct {
	User User `json:"user"`
	Type int  `json:"type"`
}
