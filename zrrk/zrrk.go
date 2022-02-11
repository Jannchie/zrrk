package zrrk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type Bot struct {
	RoomID       int
	danmakuQueue chan DanmakuData
	cookies      string
	infoURL      string
	conn         *websocket.Conn
	done         chan struct{}
	token        string
	host         string
	plugins      []BotPlugin
}

type BotPlugin interface {
	HandleDanmuData(data DanmakuData) string
}

func New() *Bot {
	return &Bot{
		infoURL:      "https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuInfo?id=%d&type=0",
		danmakuQueue: make(chan DanmakuData, 5),
	}
}

func Default(roomID int) *Bot {
	b := New()
	b.RoomID = roomID
	return b
}

func (b *Bot) AddPlugin(plugin BotPlugin) {
	b.plugins = append(b.plugins, plugin)
}

func (b *Bot) SetCookies(cookies string) {
	b.cookies = cookies
}

func (b *Bot) Connect() {
	log.Println("尝试接续直播间弹幕服务器")
	info, err := b.getDanmakuInfo()
	if err != nil {
		log.Fatal("获取弹幕服务器信息失败: ", err)
	}
	b.setHostAndToken(info)
	b.makeConnection()
	if err != nil {
		log.Fatal(err)
	}
	b.done = make(chan struct{})
	go b.recieve()
	go b.send()
	b.sendFirstMsg()
	go func() {
		for dd := range b.danmakuQueue {
			for i := range b.plugins {
				plugin := b.plugins[i]
				str := plugin.HandleDanmuData(dd)
				if str != "" {
					WriteToFile(str)
				}
			}
		}
	}()
}

func (b *Bot) makeConnection() {
	var err error
	b.conn, _, err = websocket.DefaultDialer.Dial(fmt.Sprintf("wss://%s/sub", b.host), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func (b *Bot) setHostAndToken(info *DanmakuInfoResp) {
	b.host = info.Data.HostList[0].Host
	b.token = info.Data.Token
}

func (b *Bot) sendFirstMsg() {
	log.Println("准备进行初次接触")
	data := map[string]interface{}{
		"key":      b.token,
		"protover": 2,
		"platform": "web",
		"type":     2,
		"uid":      0,
		"roomid":   b.RoomID,
	}
	body, _ := json.Marshal(data)
	var buffer bytes.Buffer
	buffer.Write(HeadGen(len(body), WS_OP_USER_AUTHENTICATION, WS_HEADER_DEFAULT_SEQUENCE))
	buffer.Write(body)
	err := b.conn.WriteMessage(websocket.BinaryMessage, buffer.Bytes())
	if err != nil {
		log.Println("初次接触失败: ", err)
	}
	log.Println("成功发送初次接触包")
}

func (b *Bot) sendHeartbeat() {
	var obj = `[object Object]`
	var buffer bytes.Buffer
	buffer.Write(HeadGen(len(obj), WS_OP_HEARTBEAT, WS_HEADER_DEFAULT_SEQUENCE))
	buffer.Write([]byte(obj))
	err := b.conn.WriteMessage(websocket.BinaryMessage, buffer.Bytes())
	if err != nil {
		log.Println("发送心跳包失败:", err)
	}
	log.Println("成功发送心跳包")
}

func (b *Bot) send() {
	interrupt := make(chan os.Signal, 1)
	ticker := time.NewTicker(time.Second * 30)
	log.Println("发送协程已启动")
	for {
		select {
		case <-b.done:
			return
		case <-interrupt:
			shouldReturn := b.doInterrupt()
			if shouldReturn {
				return
			}
		case <-ticker.C:
			b.sendHeartbeat()
		}
	}
}

func (b *Bot) doInterrupt() bool {
	log.Println("interrupt")
	err := b.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close:", err)
		return true
	}
	select {
	case <-b.done:
	case <-time.After(time.Second):
	}
	return false
}

func (b *Bot) recieve() {
	log.Println("接收协程已启动")
	defer close(b.done)
	for {
		_, message, err := b.conn.ReadMessage()
		rawHead := message[:16]
		head := GetHeader(rawHead)
		rawBody := message[16:]
		if err != nil {
			log.Println("消息读取错误: ", err)
			return
		}
		switch head.OpeaT {
		case 3:
			data := rawBody[:4]
			value := btoi32(data)
			log.Printf("当前直播间热度: %d\n", value)
		case 5:
			if head.BodyV == WS_BODY_PROTOCOL_VERSION_DEFLATE {
				body := ZlibParse(rawBody)
				// log.Printf("开始解析消息包")
				var offset int
				// 根据长度，切割出单条消息
				for offset < len(body) {
					curRawHead := body[offset : offset+16]
					curHead := GetHeader(curRawHead)
					curBody := body[offset+16 : offset+int(curHead.PackL)]
					cmd := newFunction(curBody)
					switch cmd {
					case "ONLINE_RANK_V2":
						var msg OnlineRankV2
						_ = json.Unmarshal(curBody, &msg)
						// 高能榜
					case "LIVE_INTERACTIVE_GAME":
						// 不明
					case "ONLINE_RANK_COUNT":
						//高能榜数量更新
					case "ENTRY_EFFECT":
						// 进入特效
					case "COMBO_SEND":
						// 送礼连击
					case "DANMU_MSG":
						b.handleDanmuMsg(curBody)
					case "SEND_GIFT":
						var msg SendGift
						_ = json.Unmarshal(curBody, &msg)
						b.HandleSendGift(msg)
					case "INTERACT_WORD":
						b.handleInteractWord(curBody)
					default:
						log.Printf("收到未解析的命令: %s\n %s", cmd, curBody)
					}
					// log.Printf("%+v\n%s\n", curHead, curBody)
					offset += int(curHead.PackL)
				}
				// log.Println("消息包解析结束")
			}
		case 8:
			log.Printf("初次接触成功\n")
		default:
			log.Printf("未知消息: %s\n", rawBody)
			// log.Printf("recv: %+v %s\n", head, rawBody)
		}
	}
}

func (b *Bot) handleInteractWord(data []byte) {
	var msg InteractWord
	err := json.Unmarshal(data, &msg)
	if err != nil {
		log.Println("解析互动字失败: ", err)
		return
	}
	b.HandleInteractWord(msg)
}

func (b *Bot) handleDanmuMsg(data []byte) {
	var msg DanmuMsg
	err := json.Unmarshal(data, &msg)
	if err != nil {
		log.Println("解析弹幕失败: ", err)
		return
	}
	b.HandleDanmuMsg(msg)
}

func newFunction(curBody []byte) string {
	var msg Msg
	err := json.Unmarshal(curBody, &msg)
	if err != nil {
		log.Fatalln(err)
	}
	return msg.Cmd
}

func (b *Bot) getDanmakuInfo() (*DanmakuInfoResp, error) {
	log.Println("已发起弹幕池信息请求")
	resp, err := GetResponse(fmt.Sprintf(b.infoURL, b.RoomID))
	if err != nil {
		return nil, err
	}
	var danmakuInfoResp DanmakuInfoResp
	err2 := json.NewDecoder(resp.Body).Decode(&danmakuInfoResp)
	if err2 != nil {
		return nil, err
	}
	log.Println("弹幕池信息检索成功")
	return &danmakuInfoResp, nil
}
