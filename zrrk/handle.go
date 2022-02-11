package zrrk

import (
	"fmt"
	"log"

	"github.com/fatih/color"
)


func (b *Bot) HandleInteractWord(msg InteractWord) {
	color.Set(color.FgYellow)
	m := fmt.Sprintf("%s(UID:%d) 进入了直播间", msg.Data.Uname, msg.Data.UID)
	log.Println(m)
	color.Unset()
}

func (b *Bot) HandleSendGift(msg SendGift) {
	color.Set(color.FgRed)
	m := fmt.Sprintf("%s(UID:%d) %s了 %d 个 %s", msg.Data.Uname, msg.Data.UID, msg.Data.Action, msg.Data.Num, msg.Data.GiftName)
	log.Println(m)
	color.Unset()
}

func (b *Bot) HandleDanmuMsg(msg DanmuMsg) {
	text := msg.Info[1].(string)
	userInfo := msg.Info[2].([]interface{})
	uid := int(userInfo[0].(float64))
	uname := userInfo[1].(string)
	medal := msg.Info[3].([]interface{})
	medalData := MedalData{}
	user := UserData{
		Name: uname,
		UID:  uid,
	}
	if len(medal) != 0 {
		lv := int(medal[0].(float64))
		modalTitle := medal[1].(string)
		// up := modal[2].(string)
		medalData.Level = lv
		medalData.Title = modalTitle
	}
	m := fmt.Sprintf("%s%s(UID:%d): %s", medalData.String(), uname, uid, text)
	b.danmakuQueue <- DanmakuData{
		Medal: medalData,
		User:  user,
		Text:  text,
	}
	log.Println(m)
}
