package zrrk

import (
	"fmt"
	"log"

	"github.com/fatih/color"
)

func (b *Bot) HandleInteractWord(msg InteractWord) {
	level := msg.Data.FansMedal.MedalLevel
	medalTitle := msg.Data.FansMedal.MedalName
	md := Medal{
		Level: level,
		Title: medalTitle,
	}
	var m string
	switch msg.Data.MsgType {
	case INTERACT_ENTER:
		color.Set(color.FgBlack)
		m = fmt.Sprintf("%s %s(UID: %10d) 进入了直播间", md.String(), msg.Data.Uname, msg.Data.UID)
	case INTERACT_FOLLOW:
		color.Set(color.FgMagenta)
		m = fmt.Sprintf("%s %s(UID: %10d) 关注了主播", md.String(), msg.Data.Uname, msg.Data.UID)
	}
	log.Println(m)
	b.dataChan <- InteractData{
		User: User{
			Name:  msg.Data.Uname,
			UID:   msg.Data.UID,
			Medal: md,
		},
		Type: msg.Data.MsgType,
	}
	color.Unset()
}

func (b *Bot) HandleSendGift(msg SendGift) {
	md := Medal{
		Title: msg.Data.MedalInfo.MedalName,
		Level: msg.Data.MedalInfo.MedalLevel,
	}
	ud := User{
		Name:  msg.Data.Uname,
		UID:   msg.Data.UID,
		Medal: md,
	}
	color.Set(color.FgCyan)
	m := fmt.Sprintf("%s %s(UID: %10d) %s了 %d 个 %s", md.String(), msg.Data.Uname, msg.Data.UID, msg.Data.Action, msg.Data.Num, msg.Data.GiftName)
	log.Println(m)
	color.Unset()
	gm := GiftData{
		User: ud,
		Gift: Gift{
			ID:    msg.Data.GiftID,
			Name:  msg.Data.GiftName,
			Count: msg.Data.Num,
		},
	}
	b.dataChan <- gm
}

func (b *Bot) HandleDanmuMsg(msg DanmuMsg) {
	text := msg.Info[1].(string)
	userInfo := msg.Info[2].([]interface{})
	uid := int(userInfo[0].(float64))
	uname := userInfo[1].(string)
	medal := msg.Info[3].([]interface{})
	medalData := Medal{}
	user := User{
		Name:  uname,
		UID:   uid,
		Medal: medalData,
	}
	if len(medal) != 0 {
		lv := int(medal[0].(float64))
		modalTitle := medal[1].(string)
		// up := modal[2].(string)
		medalData.Level = lv
		medalData.Title = modalTitle
	}
	m := fmt.Sprintf("%s %s(UID:%d): %s", medalData.String(), uname, uid, text)
	b.dataChan <- DanmakuData{
		User: user,
		Text: text,
	}
	log.Println(m)
}

func (b *Bot) handleSC(msg SuperChatMessage) {
	md := Medal{
		Level: msg.Data.MedalInfo.MedalLevel,
		Title: msg.Data.MedalInfo.MedalName,
	}
	ud := User{
		Name:  msg.Data.UserInfo.Uname,
		UID:   msg.Data.UID,
		Medal: md,
	}
	b.dataChan <- SCData{
		User: ud,
		Text: msg.Data.Message,
	}
}
