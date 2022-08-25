package zrrk

import (
	"fmt"
	"log"
)

func (b *Bot) HandleInteractWord(msg InteractWord) {
	level := msg.Data.FansMedal.MedalLevel
	medalTitle := msg.Data.FansMedal.MedalName
	md := Medal{
		Level: level,
		Title: medalTitle,
	}
	ud := User{
		Name:  msg.Data.Uname,
		UID:   msg.Data.UID,
		Medal: md,
	}
	switch msg.Data.MsgType {
	case INTERACT_ENTER:
		b.INFO(fmt.Sprintf("%s：进入了直播间", ud.String()))
	case INTERACT_FOLLOW:
		b.INFO(fmt.Sprintf("%s：关注了主播", ud.String()))
	default:
		b.INFO(fmt.Sprintf("%s", ud.String()))
	}
	b.dataChan <- InteractData{
		User: User{
			Name:  msg.Data.Uname,
			UID:   msg.Data.UID,
			Medal: md,
		},
		Type: msg.Data.MsgType,
	}
}

func (b *Bot) HandleUserToastMsg(msg UserToastMsg) {
	ud := User{
		Name: msg.Data.Username,
		UID:  msg.Data.UID,
	}
	b.HIGHLIGHT(fmt.Sprintf("%s：%s！舰长等级Lv.%d, [%s] 价值: %dRMB", ud.String(),
		msg.Data.ToastMsg, msg.Data.GuardLevel, msg.Data.RoleName, msg.Data.Price/1000))
	log.Printf("%+v", msg)
	gm := GiftData{
		User: ud,
		Gift: Gift{
			ID:       msg.Data.EffectID,
			Name:     msg.Data.RoleName,
			Count:    msg.Data.Num,
			Price:    msg.Data.Price,
			Currency: "GOLD",
		},
	}
	b.dataChan <- gm
}

func (b *Bot) HandleGuardBuy(msg GuardBuy) {
	ud := User{
		Name: msg.Data.Username,
		UID:  msg.Data.UID,
	}
	b.HIGHLIGHT(fmt.Sprintf("%s：上舰了！Lv.%d, [%s] 价值: %dRMB", ud.String(), msg.Data.GuardLevel, msg.Data.GiftName, msg.Data.Price/1000))
	log.Printf("%+v", msg)
	gm := GiftData{
		User: ud,
		Gift: Gift{
			ID:       msg.Data.GiftID,
			Name:     msg.Data.GiftName,
			Count:    msg.Data.Num,
			Price:    msg.Data.Price,
			Currency: "GOLD",
		},
	}
	b.dataChan <- gm
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
	if msg.Data.CoinType == "silver" && (msg.Data.Price > 0) {
		b.GIFT(fmt.Sprintf("%s：%s了 %d 个 %s, [SILVER] 价值: %d", ud.String(), msg.Data.Action, msg.Data.Num, msg.Data.GiftName, msg.Data.Price))
	} else if (msg.Data.CoinType == "gold") && (msg.Data.Price > 0) {
		b.HIGHLIGHT(fmt.Sprintf("%s：%s了 %d 个 %s, [ GOLD ] 价值: %.1fRMB", ud.String(), msg.Data.Action, msg.Data.Num, msg.Data.GiftName, float64(msg.Data.Price)/1000))
	} else {
		b.DEBUG(fmt.Sprintf("%s：%s了 %d 个 %s, [OTHERS] 价值: %d", ud.String(), msg.Data.Action, msg.Data.Num, msg.Data.GiftName, msg.Data.Price))
	}
	price := msg.Data.Price
	currency := "SILVER"
	if msg.Data.CoinType == "gold" {
		currency = "GOLD"
		price = msg.Data.Price
	}
	gm := GiftData{
		User: ud,
		Gift: Gift{
			ID:       msg.Data.GiftID,
			Name:     msg.Data.GiftName,
			Count:    msg.Data.Num,
			Price:    price,
			Currency: currency,
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
	ud := User{
		Name:  uname,
		UID:   uid,
		Medal: medalData,
	}
	b.INFO(fmt.Sprintf("%s: %s", ud.String(), text))
	b.dataChan <- DanmakuData{
		User: user,
		Text: text,
	}
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
	b.HIGHLIGHT(fmt.Sprintf("%s：<%d RMB> SC ** %s **", ud.String(), msg.Data.Price, msg.Data.Message))
	b.dataChan <- GiftData{
		User: ud,
		Gift: Gift{
			ID:       msg.Data.Gift.GiftID,
			Name:     msg.Data.Gift.GiftName,
			Count:    msg.Data.Gift.Num,
			Price:    msg.Data.Price * 1000,
			Currency: "GOLD",
		},
	}
}
