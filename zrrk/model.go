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

func (u *User) String() string {
	medalStr := u.Medal.String()
	userStr := fmt.Sprintf("%s(UID: %d)", u.Name, u.UID)
	length := getStringWidth(userStr)
	space := ""
	for i := 0; i < 36-length; i++ {
		space += " "
	}
	return fmt.Sprintf("%s %s%s", medalStr, space, userStr)
}

type Gift struct {
	ID       int    `json:"giftId"`
	Currency string `json:"typcurrencye"`
	Name     string `json:"giftName"`
	Count    int    `json:"count"`
	Price    int    `json:"price"`
}

type RoomBlockMsg struct {
	Cmd  string `json:"cmd"`
	Data struct {
		Dmscore  int    `json:"dmscore"`
		Operator int    `json:"operator"`
		UID      int    `json:"uid"`
		Uname    string `json:"uname"`
	} `json:"data"`
	UID   string `json:"uid"`
	Uname string `json:"uname"`
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
	UID   int    `json:"uid"`
	Name  string `json:"name"`
	Medal Medal  `json:"modal"`
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
