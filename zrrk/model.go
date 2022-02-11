package zrrk

import "fmt"

type Msg struct {
	Cmd string `json:"cmd"`
}
type SendGift struct {
	Cmd  string `json:"cmd"`
	Data struct {
		Action            string      `json:"action"`
		BatchComboID      string      `json:"batch_combo_id"`
		BatchComboSend    interface{} `json:"batch_combo_send"`
		BeatID            string      `json:"beatId"`
		BizSource         string      `json:"biz_source"`
		BlindGift         interface{} `json:"blind_gift"`
		BroadcastID       int         `json:"broadcast_id"`
		CoinType          string      `json:"coin_type"`
		ComboResourcesID  int         `json:"combo_resources_id"`
		ComboSend         interface{} `json:"combo_send"`
		ComboStayTime     int         `json:"combo_stay_time"`
		ComboTotalCoin    int         `json:"combo_total_coin"`
		CritProb          int         `json:"crit_prob"`
		Demarcation       int         `json:"demarcation"`
		DiscountPrice     int         `json:"discount_price"`
		Dmscore           int         `json:"dmscore"`
		Draw              int         `json:"draw"`
		Effect            int         `json:"effect"`
		EffectBlock       int         `json:"effect_block"`
		Face              string      `json:"face"`
		FloatScResourceID int         `json:"float_sc_resource_id"`
		GiftID            int         `json:"giftId"`
		GiftName          string      `json:"giftName"`
		GiftType          int         `json:"giftType"`
		Gold              int         `json:"gold"`
		GuardLevel        int         `json:"guard_level"`
		IsFirst           bool        `json:"is_first"`
		IsSpecialBatch    int         `json:"is_special_batch"`
		Magnification     float64     `json:"magnification"`
		MedalInfo         struct {
			AnchorRoomid     int    `json:"anchor_roomid"`
			AnchorUname      string `json:"anchor_uname"`
			GuardLevel       int    `json:"guard_level"`
			IconID           int    `json:"icon_id"`
			IsLighted        int    `json:"is_lighted"`
			MedalColor       int    `json:"medal_color"`
			MedalColorBorder int    `json:"medal_color_border"`
			MedalColorEnd    int    `json:"medal_color_end"`
			MedalColorStart  int    `json:"medal_color_start"`
			MedalLevel       int    `json:"medal_level"`
			MedalName        string `json:"medal_name"`
			Special          string `json:"special"`
			TargetID         int    `json:"target_id"`
		} `json:"medal_info"`
		NameColor         string      `json:"name_color"`
		Num               int         `json:"num"`
		OriginalGiftName  string      `json:"original_gift_name"`
		Price             int         `json:"price"`
		Rcost             int         `json:"rcost"`
		Remain            int         `json:"remain"`
		Rnd               string      `json:"rnd"`
		SendMaster        interface{} `json:"send_master"`
		Silver            int         `json:"silver"`
		Super             int         `json:"super"`
		SuperBatchGiftNum int         `json:"super_batch_gift_num"`
		SuperGiftNum      int         `json:"super_gift_num"`
		SvgaBlock         int         `json:"svga_block"`
		TagImage          string      `json:"tag_image"`
		Tid               string      `json:"tid"`
		Timestamp         int         `json:"timestamp"`
		TopList           interface{} `json:"top_list"`
		TotalCoin         int         `json:"total_coin"`
		UID               int         `json:"uid"`
		Uname             string      `json:"uname"`
	} `json:"data"`
}
type EntryEffect struct {
	Cmd  string `json:"cmd"`
	Data struct {
		ID               int    `json:"id"`
		UID              int    `json:"uid"`
		TargetID         int    `json:"target_id"`
		MockEffect       int    `json:"mock_effect"`
		Face             string `json:"face"`
		PrivilegeType    int    `json:"privilege_type"`
		CopyWriting      string `json:"copy_writing"`
		CopyColor        string `json:"copy_color"`
		HighlightColor   string `json:"highlight_color"`
		Priority         int    `json:"priority"`
		BasemapURL       string `json:"basemap_url"`
		ShowAvatar       int    `json:"show_avatar"`
		EffectiveTime    int    `json:"effective_time"`
		WebBasemapURL    string `json:"web_basemap_url"`
		WebEffectiveTime int    `json:"web_effective_time"`
		WebEffectClose   int    `json:"web_effect_close"`
		WebCloseTime     int    `json:"web_close_time"`
		Business         int    `json:"business"`
		CopyWritingV2    string `json:"copy_writing_v2"`
		IconList         []int  `json:"icon_list"`
		MaxDelayTime     int    `json:"max_delay_time"`
		TriggerTime      int64  `json:"trigger_time"`
		Identities       int    `json:"identities"`
	} `json:"data"`
}

type OnlineRankTop3 struct {
	Cmd  string `json:"cmd"`
	Data struct {
		Dmscore int `json:"dmscore"`
		List    []struct {
			Msg  string `json:"msg"`
			Rank int    `json:"rank"`
		} `json:"list"`
	} `json:"data"`
}

type Preparing struct {
	Cmd    string `json:"cmd"`
	Round  int    `json:"round"`
	Roomid string `json:"roomid"`
}

type RoomChange struct {
	Cmd  string `json:"cmd"`
	Data struct {
		Title          string `json:"title"`
		AreaID         int    `json:"area_id"`
		ParentAreaID   int    `json:"parent_area_id"`
		AreaName       string `json:"area_name"`
		ParentAreaName string `json:"parent_area_name"`
		LiveKey        string `json:"live_key"`
		SubSessionKey  string `json:"sub_session_key"`
	} `json:"data"`
}

type Live struct {
	Cmd             string `json:"cmd"`
	LiveKey         string `json:"live_key"`
	VoiceBackground string `json:"voice_background"`
	SubSessionKey   string `json:"sub_session_key"`
	LivePlatform    string `json:"live_platform"`
	LiveModel       int    `json:"live_model"`
	LiveTime        int    `json:"live_time"`
	Roomid          int    `json:"roomid"`
}

type InteractWord struct {
	Cmd  string `json:"cmd"`
	Data struct {
		Contribution struct {
			Grade int `json:"grade"`
		} `json:"contribution"`
		Dmscore   int `json:"dmscore"`
		FansMedal struct {
			AnchorRoomid     int    `json:"anchor_roomid"`
			GuardLevel       int    `json:"guard_level"`
			IconID           int    `json:"icon_id"`
			IsLighted        int    `json:"is_lighted"`
			MedalColor       int    `json:"medal_color"`
			MedalColorBorder int    `json:"medal_color_border"`
			MedalColorEnd    int    `json:"medal_color_end"`
			MedalColorStart  int    `json:"medal_color_start"`
			MedalLevel       int    `json:"medal_level"`
			MedalName        string `json:"medal_name"`
			Score            int    `json:"score"`
			Special          string `json:"special"`
			TargetID         int    `json:"target_id"`
		} `json:"fans_medal"`
		Identities  []int  `json:"identities"`
		IsSpread    int    `json:"is_spread"`
		MsgType     int    `json:"msg_type"`
		Roomid      int    `json:"roomid"`
		Score       int64  `json:"score"`
		SpreadDesc  string `json:"spread_desc"`
		SpreadInfo  string `json:"spread_info"`
		TailIcon    int    `json:"tail_icon"`
		Timestamp   int    `json:"timestamp"`
		TriggerTime int64  `json:"trigger_time"`
		UID         int    `json:"uid"`
		Uname       string `json:"uname"`
		UnameColor  string `json:"uname_color"`
	} `json:"data"`
}

type DanmuMsg struct {
	Cmd  string        `json:"cmd"`
	Info []interface{} `json:"info"`
}
type OnlineRankV2 struct {
	Cmd  string `json:"cmd"`
	Data struct {
		List []struct {
			UID        int    `json:"uid"`
			Face       string `json:"face"`
			Score      string `json:"score"`
			Uname      string `json:"uname"`
			Rank       int    `json:"rank"`
			GuardLevel int    `json:"guard_level"`
		} `json:"list"`
		RankType string `json:"rank_type"`
	} `json:"data"`
}

type MedalData struct {
	Title string `json:"title"`
	Level int    `json:"level"`
}
type UserData struct {
	UID  int    `json:"uid"`
	Name string `json:"name"`
}
type DanmakuData struct {
	Medal MedalData `json:"modal"`
	User  UserData  `json:"user"`
	Text  string    `json:"text"`
}

func (d *MedalData) String() string {
	if d.Level == 0 {
		return "[无勋章Lv. 0] "
	}
	return fmt.Sprintf("[%sLv.%2d] ", d.Title, d.Level)
}
