package zrrk

import (
	"fmt"
	"log"

	"github.com/fatih/color"
)

func (b *Bot) HandleInteractWord(msg InteractWord) {
	level := msg.Data.FansMedal.MedalLevel
	medalTitle := msg.Data.FansMedal.MedalName
	md := MedalData{
		Level: level,
		Title: medalTitle,
	}
	var m string
	switch msg.Data.MsgType {
	case 1:
		color.Set(color.FgBlack)
		m = fmt.Sprintf("%s %s(UID:%9d) 进入了直播间", md.String(), msg.Data.Uname, msg.Data.UID)
	case 2:
		color.Set(color.FgMagenta)
		m = fmt.Sprintf("%s %s(UID:%9d) 关注了主播", md.String(), msg.Data.Uname, msg.Data.UID)
	}
	log.Println(m)
	color.Unset()
}

func (b *Bot) HandleSendGift(msg SendGift) {
	color.Set(color.FgCyan)
	m := fmt.Sprintf("%s(UID:%9d) %s了 %d 个 %s", msg.Data.Uname, msg.Data.UID, msg.Data.Action, msg.Data.Num, msg.Data.GiftName)
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
	m := fmt.Sprintf("%s %s(UID:%d): %s", medalData.String(), uname, uid, text)
	b.danmakuQueue <- DanmakuData{
		Medal: medalData,
		User:  user,
		Text:  text,
	}
	log.Println(m)
}

func (b *Bot) handleSC(msg SuperChatMessage) {
	md := MedalData{
		Level: msg.Data.MedalInfo.MedalLevel,
		Title: msg.Data.MedalInfo.MedalName,
	}
	ud := UserData{
		Name: msg.Data.UserInfo.Uname,
		UID:  msg.Data.UID,
	}
	b.danmakuQueue <- DanmakuData{
		Medal: md,
		User:  ud,
		SC:    true,
	}
}
