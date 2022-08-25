package zrrk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

type Bot struct {
	RoomID        int
	dataChan      chan interface{}
	cookies       string
	infoURL       string
	conn          *websocket.Conn
	token         string
	host          string
	plugins       []BotPlugin
	outChannel    chan string
	descriptions  []string
	reconnectChan chan struct{}
	Lock          *sync.Mutex
}

const (
	LogHighLight = 6
	LogGift      = 1
	LogInfo      = 2
	LogWarn      = 3
	LogErr       = 4
	LogDebug     = 5
)

func (b *Bot) Log(logType int, args ...any) {
	b.Lock.Lock()
	msg := fmt.Sprint(args...)
	switch logType {
	case LogHighLight:
		color.Set(color.FgHiMagenta)
	case LogInfo:
		color.Set(color.FgHiCyan)
	case LogWarn:
		color.Set(color.FgYellow)
	case LogErr:
		color.Set(color.FgRed)
	case LogDebug:
		color.Set(color.FgHiBlack)
	case LogGift:
		color.Set(color.FgHiBlue)
	}
	log.Println(fmt.Sprintf("[ROOM %d] %s", b.RoomID, msg))
	color.Unset()
	b.Lock.Unlock()
}
func (b *Bot) HIGHLIGHT(args ...any) {
	b.Log(LogHighLight, args...)
}
func (b *Bot) GIFT(args ...any) {
	b.Log(LogGift, args...)
}
func (b *Bot) INFO(args ...any) {
	b.Log(LogInfo, args...)
}
func (b *Bot) WARNING(args ...any) {
	b.Log(LogWarn, args...)
}
func (b *Bot) ERROR(args ...any) {
	b.Log(LogErr, args...)
}
func (b *Bot) DEBUG(args ...any) {
	b.Log(LogDebug, args...)
}

type BotPlugin interface {
	HandleData(data interface{}, channel chan<- string)
	GetDescriptions() []string
	SetRoom(id int)
}

func New() *Bot {
	return &Bot{
		infoURL:       "https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuInfo?id=%d&type=0",
		dataChan:      make(chan interface{}, 100),
		outChannel:    make(chan string, 100),
		descriptions:  []string{},
		reconnectChan: make(chan struct{}),
	}
}

func Default(roomID int, m *sync.Mutex) *Bot {
	b := New()
	b.RoomID = roomID
	b.Lock = m
	return b
}

func (b *Bot) AddPlugin(plugin BotPlugin) {
	plugin.SetRoom(b.RoomID)
	b.plugins = append(b.plugins, plugin)
}

func (b *Bot) SetCookies(cookies string) {
	b.cookies = cookies
}

func (b *Bot) Connect() {
	b.Log(LogDebug, "ZRRK已开始运行")
	b.Log(LogDebug, "尝试接续直播间")
	for {
		ctx, cancel := context.WithCancel(context.Background())
		info, err := b.getDanmakuInfo()
		if err != nil {
			b.Log(LogErr, "无法获取到信息: ", err)
			cancel()
			return
		}
		b.setHostAndToken(info)
		b.makeConnection()
		if err != nil {
			log.Fatal(err)
		}
		go b.recieve(ctx)
		go b.send(ctx)
		b.sendFirstMsg()
		for i := range b.plugins {
			descriptions := b.plugins[i].GetDescriptions()
			b.descriptions = append(b.descriptions, descriptions...)
		}
		go func() {
			for dd := range b.dataChan {
				for i := range b.plugins {
					plugin := b.plugins[i]
					plugin.HandleData(dd, b.outChannel)
				}
			}
		}()
		// TODO: 优先消化 Primary，如果没有，则消化 Secondary
		go func(ctx context.Context) {
			for {
				select {
				case msg := <-b.outChannel:
					WriteToFile(msg)
				case <-time.NewTicker(time.Second * 10).C:
					if len(b.descriptions) > 0 {
						randomDescription := b.descriptions[rand.Intn(len(b.descriptions))]
						WriteToFile(randomDescription)
					}
				case <-ctx.Done():
					return
				}
			}
		}(ctx)
		<-b.reconnectChan
		b.Log(LogWarn, "检测到重连信号")
		cancel()
		b.Log(LogWarn, "重新接续直播间")
	}
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
	b.DEBUG("将进行初次接触")
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
		b.ERROR("初次接触未成功: ", err)
	}
	b.DEBUG("发送初次接触包")
}

func (b *Bot) sendHeartbeat() {
	var obj = `[object Object]`
	var buffer bytes.Buffer
	buffer.Write(HeadGen(len(obj), WS_OP_HEARTBEAT, WS_HEADER_DEFAULT_SEQUENCE))
	buffer.Write([]byte(obj))
	err := b.conn.WriteMessage(websocket.BinaryMessage, buffer.Bytes())
	if err != nil {
		b.ERROR("发送心跳包失败:", err)
	}
	b.DEBUG("成功发送心跳包")
}

func (b *Bot) send(ctx context.Context) {
	interrupt := make(chan os.Signal, 1)
	ticker := time.NewTicker(time.Second * 30)
	b.DEBUG("发送协程已启动")
	defer func() {
		b.DEBUG("发送协程已退出")
	}()
	for {
		select {
		case <-ctx.Done():
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
	b.ERROR("interrupt")
	err := b.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		b.ERROR("write close:", err)
		return true
	}
	select {
	case <-time.After(time.Second):
	}
	return false
}

func (b *Bot) recieve(ctx context.Context) {
	b.DEBUG("接收协程已启动")
	defer b.DEBUG("接收协程已退出")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, message, err := b.conn.ReadMessage()
			if err != nil {
				b.ERROR(err)
				b.reconnectChan <- struct{}{}
				return
			}
			if len(message) <= 16 {
				continue
			}
			rawHead := message[:16]
			head := GetHeader(rawHead)
			rawBody := message[16:]
			if err != nil {
				b.ERROR("消息读取错误: ", err)
				return
			}
			switch head.OpeaT {
			case 3:
				data := rawBody[:4]
				value := btoi32(data)
				b.INFO("当前直播间热度: ", value)
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
						cmd := getCMD(curBody)
						switch cmd {
						case "ONLINE_RANK_V2":
							var msg OnlineRankV2
							_ = json.Unmarshal(curBody, &msg)
						case "LIVE_INTERACTIVE_GAME":
							// log.Printf("直播间特殊表情: %s\n %s", cmd, curBody)
						case "ONLINE_RANK_COUNT":
							b.DEBUG("高能榜数量更新")
						case "ENTRY_EFFECT":
							b.DEBUG("收到了入场特效")
						case "COMBO_SEND":
							b.DEBUG("进行了送礼连击")
						case "LIVE":
							b.HIGHLIGHT("现在已开始直播")
						case "PREPARING":
							b.HIGHLIGHT("直播间正准备中")
						case "ONLINE_RANK_TOP3":
							b.DEBUG("高能榜发生变动")
						case "ROOM_CHANGE":
							b.HIGHLIGHT("修改了房间信息")
						case "GUARD_BUY":
							var msg GuardBuy
							_ = json.Unmarshal(curBody, &msg)
							b.HandleGuardBuy(msg)
						case "USER_TOAST_MSG":
							var msg UserToastMsg
							_ = json.Unmarshal(curBody, &msg)
							b.HandleUserToastMsg(msg)
							// 自动续费舰长之类的
						case "NOTICE_MSG":
							// 跑马灯
						case "DANMU_MSG":
							b.handleDanmuMsg(curBody)
						case "SUPER_CHAT_MESSAGE":
							// SC
							var msg SuperChatMessage
							_ = json.Unmarshal(curBody, &msg)
							b.handleSC(msg)
						case "SUPER_CHAT_MESSAGE_JPN":
							b.INFO("日本语超级弹幕")
						case "ANCHOR_LOT_END":
							b.INFO("检测到抽奖结束")
						case "ANCHOR_LOT_AWARD":
							b.INFO("检测到抽奖结果")
						case "WATCHED_CHANGE":
							// 观看人数变动
							var msg WatchedChange
							_ = json.Unmarshal(curBody, &msg)
							b.INFO("观看人数有变动: ", msg.Data.TextLarge)
						case "SEND_GIFT":
							var msg SendGift
							_ = json.Unmarshal(curBody, &msg)
							b.HandleSendGift(msg)
						case "INTERACT_WORD":
							b.handleInteractWord(curBody)
						case "POPULARITY_RED_POCKET_WINNER_LIST":
							var msg PopularityRedPocketWinnerList
							_ = json.Unmarshal(curBody, &msg)
							b.handlePopularityRedPocketWinnerList(&msg)
						case "COMMON_NOTICE_DANMAKU":
							var msg CommonNoticeDanmaku
							_ = json.Unmarshal(curBody, &msg)
							b.handleCommonNoticeDanmaku(&msg)
						case "ROOM_BLOCK_MSG":
							var msg RoomBlockMsg
							_ = json.Unmarshal(curBody, &msg)
							b.INFO("用户被房管封禁: ", msg.Data.UID)
						default:
							log.Printf("收到未解析的命令: %s\n %s", cmd, curBody)
						}
						// log.Printf("%+v\n%s\n", curHead, curBody)
						offset += int(curHead.PackL)
					}
					// b.Log("消息包解析结束")
				}
			case 8:
				b.DEBUG("初次接触已成功")
			default:
				b.INFO("未知消息: ", rawBody)
				// log.Printf("recv: %+v %s\n", head, rawBody)
			}
		}
	}
}

func (b *Bot) handlePopularityRedPocketWinnerList(msg *PopularityRedPocketWinnerList) {
}
func (b *Bot) handleCommonNoticeDanmaku(msg *CommonNoticeDanmaku) {
}

func (b *Bot) handleInteractWord(data []byte) {
	var msg InteractWord
	err := json.Unmarshal(data, &msg)
	if err != nil {
		b.ERROR("解析互动字失败: ", err)
		return
	}
	b.HandleInteractWord(msg)
}

func (b *Bot) handleDanmuMsg(data []byte) {
	var msg DanmuMsg
	err := json.Unmarshal(data, &msg)
	if err != nil {
		b.ERROR("解析弹幕失败: ", err)
		return
	}
	b.HandleDanmuMsg(msg)
}

func (b *Bot) handleGuardBuy(data []byte) {
	var msg DanmuMsg
	err := json.Unmarshal(data, &msg)
	if err != nil {
		b.ERROR("解析弹幕失败: ", err)
		return
	}
	b.HandleDanmuMsg(msg)
}

func getCMD(curBody []byte) string {
	var msg Msg
	err := json.Unmarshal(curBody, &msg)
	if err != nil {
		log.Fatalln(err)
	}
	return msg.Cmd
}

func (b *Bot) getDanmakuInfo() (*DanmakuInfoResp, error) {
	b.DEBUG("弹幕池情报请求")
	resp, err := GetResponse(fmt.Sprintf(b.infoURL, b.RoomID))
	if err != nil {
		return nil, err
	}
	var danmakuInfoResp DanmakuInfoResp
	err2 := json.NewDecoder(resp.Body).Decode(&danmakuInfoResp)
	if err2 != nil {
		return nil, err
	}
	b.DEBUG("连接情报已确保")
	return &danmakuInfoResp, nil
}
