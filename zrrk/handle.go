package zrrk

import (
	"fmt"
	"io/ioutil"
	"log"
)

func WriteToFile(msg string) {
	ioutil.WriteFile("../message.txt", []byte(msg), 0644)
}

func (b *Bot) HandleInteractWord(msg InteractWord) {
	m := fmt.Sprintf("%s(UID:%d) 进入了直播间", msg.Data.Uname, msg.Data.UID)
	log.Println(m)
	WriteToFile(m)
}

func (b *Bot) HandleSendGift(msg SendGift) {
	m := fmt.Sprintf("%s(UID:%d) %s了 %d 个 %s", msg.Data.Uname, msg.Data.UID, msg.Data.Action, msg.Data.Num, msg.Data.GiftName)
	log.Println(m)
	WriteToFile(m)
}

func (b *Bot) HandleDanmuMsg(msg DanmuMsg) {
	text := msg.Info[1].(string)
	userInfo := msg.Info[2].([]interface{})
	uid := int(userInfo[0].(float64))
	uname := userInfo[1].(string)
	modal := msg.Info[3].([]interface{})
	modalStr := ""
	if len(modal) != 0 {
		lv := int(modal[0].(float64))
		modalTitle := modal[1].(string)
		// up := modal[2].(string)
		modalStr = fmt.Sprintf("[%sLv.%d] ", modalTitle, lv)
	}
	m := fmt.Sprintf("%s%s(UID:%d): %s", modalStr, uname, uid, text)
	log.Println(m)
	WriteToFile(m)
}
