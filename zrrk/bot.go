package zrrk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	ReconnectChan chan struct{}
	ExitChan      chan struct{}
	Lock          *sync.Mutex
	StayMinHot    int32
	LogLevel      int
}

const (
	LogHighLight = 5
	LogGift      = 4
	LogInfo      = 3
	LogWarn      = 2
	LogErr       = 1
	LogDebug     = 0
)

func (b *Bot) Log(logType int, args ...any) {
	if b.LogLevel > logType {
		return
	}
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
	log.Println(fmt.Sprintf("[ROOM %10d] %s", b.RoomID, msg))
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
		ReconnectChan: make(chan struct{}),
		ExitChan:      make(chan struct{}),
	}
}

type BotConfig struct {
	RoomID     int
	StayMinHot int32
	LogLevel   int
}

func Default(m *sync.Mutex, config *BotConfig) *Bot {
	b := New()
	b.RoomID = config.RoomID
	b.Lock = m
	b.StayMinHot = config.StayMinHot
	b.LogLevel = config.LogLevel
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
	b.DEBUG("ZRRK已开始运行")
	b.DEBUG("尝试接续直播间")
	for {
		ctx, cancel := context.WithCancel(context.Background())
		info, err := b.getDanmakuInfo()
		if err != nil {
			b.ERROR("无法获取到信息: ", err)
			cancel()
			return
		}
		err = b.setHostAndToken(info)
		if err != nil {
			b.ERROR("无法获取到信息: ", err)
			cancel()
			<-time.After(time.Second * 5)
			continue
		}
		err = b.makeConnection()
		if err != nil {
			b.ERROR("建立该连接失败: ", err)
			cancel()
			<-time.After(time.Second * 5)
			continue
		}
		go b.recieve(ctx)
		go b.send(ctx)
		err = b.sendFirstMsg()
		if err != nil {
			b.ERROR("初次接触未成功: ", err)
			cancel()
			<-time.After(time.Second * 5)
			continue
		}
		for i := range b.plugins {
			descriptions := b.plugins[i].GetDescriptions()
			b.descriptions = append(b.descriptions, descriptions...)
		}
		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				case dd := <-b.dataChan:
					for i := range b.plugins {
						plugin := b.plugins[i]
						plugin.HandleData(dd, b.outChannel)
					}
				}
			}
		}(ctx)
		// TODO: 优先消化 Primary，如果没有，则消化 Secondary
		go func(ctx context.Context) {
			ticker := time.NewTicker(time.Second * 10)
			defer ticker.Stop()
			for {
				select {
				case msg := <-b.outChannel:
					WriteToFile(msg)
				case <-ticker.C:
					if len(b.descriptions) > 0 {
						randomDescription := b.descriptions[rand.Intn(len(b.descriptions))]
						WriteToFile(randomDescription)
					}
				case <-ctx.Done():
					return
				}
			}
		}(ctx)
		select {
		case <-b.ReconnectChan:
			b.WARNING("检测到重连信号")
			cancel()
			b.HIGHLIGHT("重新接续直播间")
		case <-b.ExitChan:
			b.WARNING("检测到退出信号")
			cancel()
			b.DEBUG("已经退出直播间")
			return
		}
	}
}

func (b *Bot) makeConnection() error {
	var err error
	b.conn, _, err = websocket.DefaultDialer.Dial(fmt.Sprintf("wss://%s/sub", b.host), nil)
	if err != nil {
		return err
	}
	return nil
}

func (b *Bot) setHostAndToken(info *DanmakuInfoResp) error {
	if info == nil || info.Data.HostList == nil || len(info.Data.HostList) == 0 {
		return errors.New("无法获取到主播信息")
	}
	b.host = info.Data.HostList[0].Host
	b.token = info.Data.Token
	return nil
}

func (b *Bot) sendFirstMsg() error {
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
		return err
	}
	b.DEBUG("发送初次接触包")
	return nil
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
	defer ticker.Stop()
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
				b.ReconnectChan <- struct{}{}
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
				if value < int32(b.StayMinHot) {
					b.WARNING("热度低于设定值")
					b.ExitChan <- struct{}{}
					return
				}
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
						cmd, err := getCMD(curBody)
						if err != nil {
							b.ERROR("解析消息包错误: ", err)
						}
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
							b.INFO("修改了房间信息")
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
						case "ROOM_ADMIN_REVOKE":
							// {"cmd":"ROOM_ADMIN_REVOKE","msg":"撤销房管","uid":1991698735}
						case "CUT_OFF":
							// {"cmd":"CUT_OFF","msg":"\u76f4\u64ad\u5185\u5bb9\u4e0d\u9002\u5b9c","roomid":25234878}
						case "PLAY_TOGETHER":
							// {"cmd":"PLAY_TOGETHER","data":{"ruid":95546001,"roomid":22631750,"action":"switch_on","uid":0,"timestamp":1661460719,"message":"","message_type":0,"jump_url":"","web_url":"","apply_number":0,"refresh_tool":false,"cur_fleet_num":0,"max_fleet_num":0}}
						case "LIVE_PANEL_CHANGE":
							// {"cmd":"LIVE_PANEL_CHANGE","data":{"type":2,"scatter":{"max":150,"min":5}}}
						case "LIVE_OPEN_PLATFORM_GAME":
							// {"cmd":"LIVE_OPEN_PLATFORM_GAME","data":{"msg_type":"game_end","msg_sub_type":"game_end","game_name":"炫彩钓鱼王","game_code":"1659814658645","game_id":"fb2891c8-651d-4de9-8727-154c7b98e4c3","game_status":"","game_msg":"","game_conf":"","interactive_panel_conf":"","timestamp":1661460722,"block_uids":[]}}
						case "LIVE_PANEL_CHANGE_CONTENT":
							// {"cmd":"LIVE_PANEL_CHANGE_CONTENT","data":{"setting_list":[{"biz_id":1001,"icon":"http://i0.hdslb.com/bfs/live/afd5bc2424ebf7c7c9c68d71ba5a1f7d08154519.png","title":"分享","note":"分享","weight":100,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":1012,"icon":"http://i0.hdslb.com/bfs/live/1e3cb35056ebbcc1af5f08f4fe7916f095db26a5.png","title":"管理员","note":"管理员","weight":36,"status_type":1,"notification":null,"custom":null,"jump_url":"https://live.bilibili.com/p/html/live-app-room-admin/index.html?is_live_half_webview=1#/roomManagement","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":1011,"icon":"http://i0.hdslb.com/bfs/live/7dbaf07b4c10182aeb0e7a8eda3273d40bb9b9b5.png","title":"小窗播放","note":"小窗播放","weight":15.001,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":1003,"icon":"http://i0.hdslb.com/bfs/live/a5407c843e72d5efb678b649aecd7184f0d68494.png","title":"播放设置","note":"播放设置","weight":9,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":1004,"icon":"http://i0.hdslb.com/bfs/live/1a1b3b9819f78df76f66b3657a6be2cc0e9b8853.png","title":"弹幕设置","note":"弹幕设置","weight":8,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":1002,"icon":"http://i0.hdslb.com/bfs/live/1b19309441c997d8e9a19ddb939ff6dda2a04a64.png","title":"画质","note":"画质","weight":7,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":1005,"icon":"http://i0.hdslb.com/bfs/live/12d66e639a677df2e8b6630a9abe06806acce87d.png","title":"隐藏特效","note":"隐藏特效","weight":6,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":1013,"icon":"https://i0.hdslb.com/bfs/live/856061fa98257d996a34850ef4f7a052af6fb3a3.png","title":"清屏","note":"清屏","weight":5,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":1007,"icon":"http://i0.hdslb.com/bfs/live/7e25a262e1cdf294a5d6ca2b1b1527ef4f7caf62.png","title":"举报","note":"举报","weight":5,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":1009,"icon":"http://i0.hdslb.com/bfs/live/8e41f28e574952208fe73d09d464c8b369a1a4e9.png","title":"反馈","note":"反馈","weight":4,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":1008,"icon":"http://i0.hdslb.com/bfs/live/fe04b9ab783d3a0a4798c20303166b07dcdf8f1d.png","title":"投屏","note":"投屏","weight":3,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":1006,"icon":"http://i0.hdslb.com/bfs/live/628cdab93480f1f3dfcb4430a1ff08c81c1b6aec.png","title":"仅播声音","note":"仅播声音","weight":2,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":1014,"icon":"http://i0.hdslb.com/bfs/live/0884ed6a7c55baf37554c15d79e03c7948421d9b.png","title":"色觉优化","note":"色觉优化","weight":1,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":1010,"icon":"http://i0.hdslb.com/bfs/live/1c8331a2c520093a830df0ebf9b5f58eb28cd22d.png","title":"添至桌面","note":"添至桌面","weight":1,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":1,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0}],"interaction_list":[{"biz_id":5,"icon":"https://i0.hdslb.com/bfs/live/9642030f43c085b5b4ac9f0903ea03ff85d2544c.png","title":"限时 热门榜","note":"未上榜","weight":2,"status_type":1,"notification":null,"custom":[{"icon":"","title":"限时热门榜","note":"未上榜","jump_url":"https://live.bilibili.com/p/html/live-app-hotrank/index.html?clientType=1\u0026area_id=0\u0026parent_area_id=0\u0026second_area_id=0\u0026is_live_half_webview=1\u0026hybrid_rotate_d=1\u0026hybrid_half_ui=1,3,100p,70p,ffffff,0,30,100,12,0;2,2,375,100p,ffffff,0,30,100,0,0;3,3,100p,70p,ffffff,0,30,100,12,0;4,2,375,100p,ffffff,0,30,100,0,0;5,3,100p,70p,ffffff,0,30,100,0,0;6,3,100p,70p,ffffff,0,30,100,0,0;7,3,100p,70p,ffffff,0,30,100,0,0;8,3,100p,70p,ffffff,0,30,100,0,0","status":0,"sub_icon":""}],"jump_url":"https://live.bilibili.com/p/html/live-app-hotrank/index.html?clientType=1\u0026area_id=0\u0026parent_area_id=0\u0026second_area_id=0\u0026is_live_half_webview=1\u0026hybrid_rotate_d=1\u0026hybrid_half_ui=1,3,100p,70p,ffffff,0,30,100,12,0;2,2,375,100p,ffffff,0,30,100,0,0;3,3,100p,70p,ffffff,0,30,100,12,0;4,2,375,100p,ffffff,0,30,100,0,0;5,3,100p,70p,ffffff,0,30,100,0,0;6,3,100p,70p,ffffff,0,30,100,0,0;7,3,100p,70p,ffffff,0,30,100,0,0;8,3,100p,70p,ffffff,0,30,100,0,0","type_id":2,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0}],"outer_list":[{"biz_id":997,"icon":"https://i0.hdslb.com/bfs/live/273904e5c84d293f5f9df5ade5ac0fadc34e9fad.png","title":"送礼","note":"","weight":100,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":2,"tab":null,"dynamic_icon":"https://i0.hdslb.com/bfs/live/a812dfafd427714b3623a352618ca70fa0379c75.webp","sub_icon":"https://i0.hdslb.com/bfs/live/b0b675140c28310a0ff54b05b2fd9a11a5898acf.png","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":33,"icon":"https://i0.hdslb.com/bfs/live/a0e4a9381f9627d2ed89ab67d5ccce1bc1de7ea3.png","title":"购物车","note":"购物车","weight":100,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":2,"tab":null,"dynamic_icon":"","sub_icon":"https://i0.hdslb.com/bfs/live/76b00ae4363ab572be565dbb62fd44d7c6c7d198.png","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":998,"icon":"https://i0.hdslb.com/bfs/live/ec39c5ec3185f58608e4c143f2461726794403b0.png","title":"更多","note":"","weight":99,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":2,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":16,"icon":"https://i0.hdslb.com/bfs/live/024b6050b1cf11ed656a499f013ca14681a131c6.png","title":"表情包","note":"表情包","weight":98,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":2,"tab":null,"dynamic_icon":"","sub_icon":"https://i0.hdslb.com/bfs/live/57b7d3953b5663931c59f7e889cef76950591f03.png","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":30,"icon":"https://s1.hdslb.com/bfs/live/4d6577503048f219aa8c9a3a7b6a1a61fb3ee0ba.png","title":"快捷送礼","note":"快捷送礼","weight":97,"status_type":1,"notification":null,"custom":[{"icon":"https://s1.hdslb.com/bfs/live/4d6577503048f219aa8c9a3a7b6a1a61fb3ee0ba.png","title":"","note":"{\"bubble_text\":\"点击投喂一个%s，让主播感受到你的支持！\",\"desc_text\":\"投喂一个%s支持主播~\",\"duration\":3,\"gift_id\":31036}","jump_url":"","status":0,"sub_icon":"https://s1.hdslb.com/bfs/live/4d6577503048f219aa8c9a3a7b6a1a61fb3ee0ba.png"}],"jump_url":"","type_id":2,"tab":null,"dynamic_icon":"","sub_icon":"https://s1.hdslb.com/bfs/live/4d6577503048f219aa8c9a3a7b6a1a61fb3ee0ba.png","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":2,"icon":" ","title":"语音连麦","note":" ","weight":5,"status_type":1,"notification":null,"custom":[{"icon":"https://i0.hdslb.com/bfs/live/e3a8c212bc493b88a33fe1853a16270e22d9a70b.png","title":"","note":"连麦功能关闭","jump_url":"","status":2,"sub_icon":"https://i0.hdslb.com/bfs/live/e429e283dbd9e25092a5a73b604527a646cbad32.png"},{"icon":"https://i0.hdslb.com/bfs/live/b8cabd73def53d85bd092f4e8b3f9f6534ec2dc6.png","title":"","note":"连麦","jump_url":"","status":1,"sub_icon":"https://i0.hdslb.com/bfs/live/9500b71c99451040e96312a0f60f269f5c6f0100.png"},{"icon":"https://i0.hdslb.com/bfs/live/c25451d846c5c36a56874626c6496743e6c8b726.webp","title":"","note":"等待中","jump_url":"","status":3,"sub_icon":"https://i0.hdslb.com/bfs/live/0a4e8a81ccc673d7985b6a3c9ecc88baaa0c1e35.webp"},{"icon":"https://i0.hdslb.com/bfs/live/bcf5f48883ddbb96c8680bcc9ed2d4c11798e526.webp","title":"","note":"连麦中","jump_url":"","status":4,"sub_icon":"https://i0.hdslb.com/bfs/live/846230df75319bbe171db0e0d18ec5a8a80e514b.webp"}],"jump_url":"","type_id":2,"tab":null,"dynamic_icon":"","sub_icon":"","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0},{"biz_id":3,"icon":"https://i0.hdslb.com/bfs/live/a02f9edd13bf77588ec8ed800cf246fbbc158ff3.png","title":"醒目留言","note":"留言传递心意吧","weight":2.001,"status_type":1,"notification":null,"custom":null,"jump_url":"","type_id":2,"tab":null,"dynamic_icon":"","sub_icon":"https://i0.hdslb.com/bfs/live/da519a9d33dd9cf8d6bb38c481cea9180341abbe.png","panel_icon":"https://i0.hdslb.com/bfs/live/98e692836d408ab7f2b321c717e866a8fd9b3bfd.png","match_entrance":0}],"panel_data":null,"is_fixed":0,"is_match":0,"match_cristina":"","match_icon":"","match_bg_image":""}}
						case "DANMU_AGGREGATION":
							// {"cmd":"DANMU_AGGREGATION","data":{"activity_identity":"3092106","activity_source":1,"aggregation_cycle":1,"aggregation_icon":"https://i0.hdslb.com/bfs/live/c8fbaa863bf9099c26b491d06f9efe0c20777721.png","aggregation_num":6,"dmscore":144,"msg":"国服 玄策，双区可带，秒刷秒上。","show_rows":1,"show_time":2,"timestamp":1661461698}}
						case "PK_BATTLE_FINAL_PROCESS":
							// {"cmd":"PK_BATTLE_FINAL_PROCESS","data":{"battle_type":1,"pk_frozen_time":1661462395},"pk_id":304995704,"pk_status":201,"timestamp":1661462276}
						case "PK_BATTLE_SETTLE":
							// {"cmd":"PK_BATTLE_SETTLE","pk_id":304995704,"pk_status":401,"settle_status":1,"timestamp":1661462396,"data":{"battle_type":1,"result_type":1,"star_light_msg":""},"roomid":"22537565"}
						case "PK_BATTLE_START":
							// {"cmd":"ANCHOR_LOT_START","data":{"asset_icon":"https://i0.hdslb.com/bfs/live/627ee2d9e71c682810e7dc4400d5ae2713442c02.png","award_image":"","award_name":"情书","award_num":1,"award_price_text":"价值52电池","award_type":1,"cur_gift_num":0,"current_time":1661462392,"danmu":"舰长包帮打王者+永久车位+指导","gift_id":0,"gift_name":"","gift_num":0,"gift_price":0,"goaway_time":180,"goods_id":-99998,"id":3092118,"is_broadcast":1,"join_type":0,"lot_status":0,"max_time":900,"require_text":"至少成为主播的提督","require_type":3,"require_value":2,"room_id":21710524,"send_gift_ensure":0,"show_panel":1,"start_dont_popup":0,"status":1,"time":899,"url":"https://live.bilibili.com/p/html/live-lottery/anchor-join.html?is_live_half_webview=1\u0026hybrid_biz=live-lottery-anchor\u0026hybrid_half_ui=1,5,100p,100p,000000,0,30,0,0,1;2,5,100p,100p,000000,0,30,0,0,1;3,5,100p,100p,000000,0,30,0,0,1;4,5,100p,100p,000000,0,30,0,0,1;5,5,100p,100p,000000,0,30,0,0,1;6,5,100p,100p,000000,0,30,0,0,1;7,5,100p,100p,000000,0,30,0,0,1;8,5,100p,100p,000000,0,30,0,0,1","web_url":"https://live.bilibili.com/p/html/live-lottery/anchor-join.html"}}
						case "INTERACTIVE_THE_CHOSEN_ONE":
							// {"cmd":"INTERACTIVE_THE_CHOSEN_ONE","data":{"id":5872,"status":2,"user_num":0,"smh_num":100,"winner_uid":0,"winner_name":"","delay":30,"start_ts":1661462074,"end_ts":1661462399,"icon_app":"https://i0.hdslb.com/bfs/live/fc09eb17bc674e635f1ac9ab94097f92ebd8d67d.png","icon_web":"https://i0.hdslb.com/bfs/live/d08025c2b8d25d947bcb5af01c754155427f6246.png","h5_url":"https://live.bilibili.com/p/html/live-app-the-chosen-one/user.html?is_live_half_webview=1\u0026hybrid_half_ui=1,5,100p,100p,d56a76,0,30,0,0,0;2,5,100p,100p,d56a76,0,30,0,0,0;3,5,100p,100p,d56a76,0,30,0,0,0;4,5,100p,100p,d56a76,0,30,0,0,0;5,5,100p,100p,d56a76,0,30,0,0,0;6,5,100p,100p,d56a76,0,30,0,0,0;7,5,100p,100p,d56a76,0,30,0,0,0;8,5,100p,100p,d56a76,0,30,0,0,0","new_fans_num":0}}
						case "ANCHOR_LOT_START":
							// {"cmd":"ANCHOR_LOT_START","data":{"asset_icon":"https://i0.hdslb.com/bfs/live/627ee2d9e71c682810e7dc4400d5ae2713442c02.png","award_image":"","award_name":"2元红包","award_num":1,"award_type":0,"cur_gift_num":0,"current_time":1661461664,"danmu":"国服玄策，双区可带，秒刷秒上。","gift_id":0,"gift_name":"","gift_num":1,"gift_price":0,"goaway_time":180,"goods_id":-99998,"id":3092106,"is_broadcast":1,"join_type":0,"lot_status":0,"max_time":600,"require_text":"关注主播","require_type":1,"require_value":0,"room_id":23409672,"send_gift_ensure":0,"show_panel":1,"start_dont_popup":0,"status":1,"time":599,"url":"https://live.bilibili.com/p/html/live-lottery/anchor-join.html?is_live_half_webview=1\u0026hybrid_biz=live-lottery-anchor\u0026hybrid_half_ui=1,5,100p,100p,000000,0,30,0,0,1;2,5,100p,100p,000000,0,30,0,0,1;3,5,100p,100p,000000,0,30,0,0,1;4,5,100p,100p,000000,0,30,0,0,1;5,5,100p,100p,000000,0,30,0,0,1;6,5,100p,100p,000000,0,30,0,0,1;7,5,100p,100p,000000,0,30,0,0,1;8,5,100p,100p,000000,0,30,0,0,1","web_url":"https://live.bilibili.com/p/html/live-lottery/anchor-join.html"}}
						case "ANCHOR_LOT_CHECKSTATUS":
							// {"cmd":"ANCHOR_LOT_CHECKSTATUS","data":{"id":3092169,"status":4,"uid":436238604}}
						case "VOICE_JOIN_ROOM_COUNT_INFO":
							// {"cmd":"VOICE_JOIN_ROOM_COUNT_INFO","data":{"cmd":"","room_id":869833,"root_status":1,"room_status":1,"apply_count":1,"notify_count":0,"red_point":1},"room_id":869833}
						case "VOICE_JOIN_LIST":
							// {"cmd":"VOICE_JOIN_LIST","data":{"cmd":"","room_id":869833,"category":1,"apply_count":1,"red_point":1,"refresh":1},"room_id":869833}
						case "POPULARITY_RED_POCKET_START":
							// {"cmd":"POPULARITY_RED_POCKET_START","data":{"lot_id":5458200,"sender_uid":1746083,"sender_name":"人鱼A梦","sender_face":"http://i2.hdslb.com/bfs/face/9d5ea62a51fd8254bf52b071e832e635bf842ce7.jpg","join_requirement":1,"danmu":"老板大气！点点红包抽礼物","current_time":1661501281,"start_time":1661501281,"end_time":1661501461,"last_time":180,"remove_time":1661501476,"replace_time":1661501471,"lot_status":1,"h5_url":"https://live.bilibili.com/p/html/live-app-red-envelope/popularity.html?is_live_half_webview=1\u0026hybrid_half_ui=1,5,100p,100p,000000,0,50,0,0,1;2,5,100p,100p,000000,0,50,0,0,1;3,5,100p,100p,000000,0,50,0,0,1;4,5,100p,100p,000000,0,50,0,0,1;5,5,100p,100p,000000,0,50,0,0,1;6,5,100p,100p,000000,0,50,0,0,1;7,5,100p,100p,000000,0,50,0,0,1;8,5,100p,100p,000000,0,50,0,0,1\u0026hybrid_rotate_d=1\u0026hybrid_biz=popularityRedPacket\u0026lotteryId=5458200","user_status":2,"awards":[{"gift_id":31212,"gift_name":"打call","gift_pic":"https://s1.hdslb.com/bfs/live/f75291a0e267425c41e1ce31b5ffd6bfedc6f0b6.png","num":2},{"gift_id":31214,"gift_name":"牛哇","gift_pic":"https://s1.hdslb.com/bfs/live/b8a38b4bd3be120becddfb92650786f00dffad48.png","num":3},{"gift_id":31216,"gift_name":"i了i了","gift_pic":"https://s1.hdslb.com/bfs/live/1157a445487b39c0b7368d91b22290c60fa665b2.png","num":3}],"lot_config_id":3,"total_price":1600,"wait_num":25}}
						case "TRADING_SCORE":
							// {"cmd":"TRADING_SCORE","data":{"bubble_show_time":3,"num":5,"score_id":3,"uid":20066851,"update_time":1661501291,"update_type":1}}
						case "PK_BATTLE_PRE_NEW":
							// {"cmd":"PK_BATTLE_PRE_NEW","pk_status":101,"pk_id":305002745,"timestamp":1661501302,"data":{"battle_type":1,"match_type":1,"uname":"\u4e8c\u516b__8\u670826\u53f7\u6ee1\u6708\u54e6","face":"http:\/\/i0.hdslb.com\/bfs\/face\/0b71965e95963270e6456bf4e27ff7cb06e553fa.jpg","uid":3461563847543178,"room_id":25570949,"season_id":52,"pre_timer":10,"pk_votes_name":"\u4e71\u6597\u503c","end_win_task":null},"roomid":1604540}
						case "PK_BATTLE_PRE":
							// {"cmd":"PK_BATTLE_PRE","pk_status":101,"pk_id":305002745,"timestamp":1661501302,"data":{"battle_type":1,"match_type":1,"uname":"\u4e8c\u516b__8\u670826\u53f7\u6ee1\u6708\u54e6","face":"http:\/\/i0.hdslb.com\/bfs\/face\/0b71965e95963270e6456bf4e27ff7cb06e553fa.jpg","uid":3461563847543178,"room_id":25570949,"season_id":52,"pre_timer":10,"pk_votes_name":"\u4e71\u6597\u503c","end_win_task":null},"roomid":1604540}
						case "PK_BATTLE_START_NEW":
							// {"cmd":"PK_BATTLE_START_NEW","pk_id":305002745,"pk_status":201,"timestamp":1661501312,"data":{"battle_type":1,"final_hit_votes":0,"pk_start_time":1661501312,"pk_frozen_time":1661501612,"pk_end_time":1661501622,"pk_votes_type":0,"pk_votes_add":0,"pk_votes_name":"\u4e71\u6597\u503c","star_light_msg":"","pk_countdown":1661501552,"final_conf":{"switch":1,"start_time":1661501432,"end_time":1661501492},"init_info":{"room_id":25570949,"date_streak":0},"match_info":{"room_id":1604540,"date_streak":0}},"roomid":"1604540"}
						case "HOT_BUY_NUM":
							// {"cmd":"HOT_BUY_NUM","data":{"goods_id":"1499719178894123008","num":397}}
						case "GOTO_BUY_FLOW":
							// {"cmd":"GOTO_BUY_FLOW","data":{"text":"塞**正在去买"}}
						case "room_admin_entrance":
							// {"cmd":"room_admin_entrance","dmscore":45,"level":1,"msg":"系统提示：你已被主播设为房管","uid":1743919882}
						case "ROOM_ADMINS":
							// {"cmd":"ROOM_ADMINS","uids":[283751299,5724746,230091229,3243360,19704588,37996142,22959012,207534777,24004453,1743919882]}
						case "SHOPPING_CART_SHOW":
							// {"cmd":"SHOPPING_CART_SHOW","data":{"status":1}}
						case "SELECTED_GOODS_INFO":
							// {"cmd":"SELECTED_GOODS_INFO","data":{"change_type":3,"item":[{"goods_id":"1529022926925815814","goods_name":"搞机所 台式电脑主机 酷睿 i5 12400F/RX6500 XT 电竞 高配 游戏","source":1,"goods_icon":"http://i0.hdslb.com/bfs/e-commerce-goods/93b2fa7163a594e00c14555d828a325a815ba901.jpg","is_pre_sale":0,"activity_info":null,"pre_sale_info":null,"early_bird_info":null,"coupon_discount_price":"","selected_text":"","is_gift_buy":0,"goods_price":"3650","goods_max_price":"","reward_info":null},{"goods_id":"1529022926925815813","goods_name":"搞机所 台式电脑主机 酷睿 i5 12400F/RX6650 XT 电竞 高配 游戏","source":1,"goods_icon":"http://i0.hdslb.com/bfs/e-commerce-goods/8055d0fd05fee1676d063d9c03889fd355e1e5b4.jpg","is_pre_sale":0,"activity_info":null,"pre_sale_info":null,"early_bird_info":null,"coupon_discount_price":"","selected_text":"","is_gift_buy":0,"goods_price":"5199","goods_max_price":"","reward_info":null},{"goods_id":"1529022926925815808","goods_name":"搞机所 台式电脑主机 酷睿i5 12400F/3070Ti 电竞 高配 办公 游戏","source":1,"goods_icon":"http://i0.hdslb.com/bfs/e-commerce-goods/5bf950b11edee9da8d52646b89465298abc0b7ac.jpg","is_pre_sale":0,"activity_info":null,"pre_sale_info":null,"early_bird_info":null,"coupon_discount_price":"","selected_text":"","is_gift_buy":0,"goods_price":"7399","goods_max_price":"","reward_info":null},{"goods_id":"1529022926925815809","goods_name":"搞机所 台式电脑主机 酷睿 i5 12400F/3060 电竞 高配 办公 游戏","source":1,"goods_icon":"http://i0.hdslb.com/bfs/e-commerce-goods/1d3fa00402632828e29241dc95822b7649d53f70.jpg","is_pre_sale":0,"activity_info":null,"pre_sale_info":null,"early_bird_info":null,"coupon_discount_price":"","selected_text":"","is_gift_buy":0,"goods_price":"4950","goods_max_price":"","reward_info":null}],"title":"主播精选"}}
						case "ROOM_MODULE_DISPLAY":
							// {"cmd":"ROOM_MODULE_DISPLAY","data":{"timestamp":1661503652,"modules":{"bottom_banner":1,"top_banner":1,"widget_banner":1}}}
						case "POPULARITY_RED_POCKET_NEW":
							// {"cmd":"POPULARITY_RED_POCKET_NEW","data":{"lot_id":5461312,"start_time":1661509185,"current_time":1661509185,"wait_num":0,"uname":"直播小电视","uid":1407831746,"action":"送出","num":1,"gift_name":"红包","gift_id":13000,"price":70,"name_color":"","medal_info":{"target_id":0,"special":"","icon_id":0,"anchor_uname":"","anchor_roomid":0,"medal_level":0,"medal_name":"","medal_color":0,"medal_color_start":0,"medal_color_end":0,"medal_color_border":0,"is_lighted":0,"guard_level":0}}}
						case "RING_STATUS_CHANGE":
							// {"cmd":"RING_STATUS_CHANGE","data":{"status":0}}
						case "SUPER_CHAT_MESSAGE_DELETE":
							// {"cmd":"SUPER_CHAT_MESSAGE_DELETE","data":{"ids":[4892379]},"roomid":22880700}
						case "SUPER_CHAT_ENTRANCE":
							// {"cmd":"SUPER_CHAT_ENTRANCE","data":{"status":1,"jump_url":"https:\/\/live.bilibili.com\/p\/html\/live-app-superchat2\/index.html?is_live_half_webview=1&hybrid_half_ui=1,3,100p,70p,ffffff,0,30,100;2,2,375,100p,ffffff,0,30,100;3,3,100p,70p,ffffff,0,30,100;4,2,375,100p,ffffff,0,30,100;5,3,100p,60p,ffffff,0,30,100;6,3,100p,60p,ffffff,0,30,100;7,3,100p,60p,ffffff,0,30,100","icon":"https:\/\/i0.hdslb.com\/bfs\/live\/0a9ebd72c76e9cbede9547386dd453475d4af6fe.png","broadcast_type":0},"roomid":"22330922"}
						case "PANEL_INTERACTIVE_NOTIFY_CHANGE":
							// {"cmd":"PANEL_INTERACTIVE_NOTIFY_CHANGE","data":{"biz_id":4,"end_time":300,"icon":"https://i0.hdslb.com/bfs/live/164a37487431ce065981d76afe6c2fb2083facee.png","last_time":5,"level":1,"text":"主播开启预言"}}
						case "INTERACTIVE_USER":
							// {"cmd":"INTERACTIVE_USER","data":{"type":1,"value":{"delay":5,"dm_msg":"主播已开启预言玩法，点击直播间底部互动按钮参与","prophet_status":1,"send_msg":1}}}
						case "SHOPPING_BUBBLES_STYLE":
							// {"cmd":"SHOPPING_BUBBLES_STYLE","data":{"interval_between_bubbles":10,"interval_between_queues":10,"cycle_time":180,"goods_count":23,"checksum":"4c995f4f70b112575e290bcd69736067","bubbles_list":[{"tag":"giftbuy","name":"福哩购","priority":1,"show_banner":1,"goods_list":["1544207553222549504"]},{"tag":"coupon","name":"亿点券","priority":2,"show_banner":1,"goods_list":["1549576033690148864","1531173479947223040","1537659287350104064","1547771752473309184","1524926140823838720","1524926519791783936","1524926284793348096","1563022395630247936"]},{"tag":"goodsnum","name":"N个宝","priority":6,"show_banner":0,"goods_list":[]},{"tag":"onlyone","name":"快抢啊","priority":7,"show_banner":0,"goods_list":[]}]}}
						case "SHOPPING_EXPLAIN_CARD":
							// {"cmd":"SHOPPING_EXPLAIN_CARD","data":{"goods_id":"1531173988422647808","goods_name":"i5 12400F/RTX3060Ti/3070Ti/16G/500G游戏台式电脑主机diy组装机","goods_price":"5599","goods_max_price":"","sale_status":0,"coupon_name":"立减400元","goods_icon":"http://i0.hdslb.com/bfs/e-commerce-goods/2ad6ed6a8effc4a82bdaaf9b0a662956fbb0daac.jpg","status":3,"h5_url":"https://live.bilibili.com/p/html/live-app-ecommerce/index.html?is_live_half_webview=1\u0026hybrid_rotate_d=0\u0026hybrid_half_ui=1,3,100p,70p,0,0,30,100,12,0;2,2,375,100p,0,0,30,100,0,0;3,3,100p,70p,0,0,30,100,12,0;4,2,375,100p,0,0,30,100,0,0;5,3,100p,70p,0,0,30,100,12,0;6,3,100p,70p,0,0,30,100,12,0;7,3,100p,70p,0,0,30,100,12,0\u0026web_type=1\u0026source=1\u0026goods_id=1531173988422647808#/taobao","source":1,"timestamp":1661512808,"is_pre_sale":0,"activity_info":null,"pre_sale_info":null,"early_bird_info":null,"unique_id":"1563124129732079616","uid":297991412,"selling_point":"","coupon_discount_price":"5199.00","sei_status":0,"gift_buy_info":null,"reward_info":null,"is_exclusive":false,"coupon_id":""}}
						case "ACTIVITY_BANNER_CHANGE":
							// {"cmd":"ACTIVITY_BANNER_CHANGE","data":{"list":[{"id":2169,"timestamp":1661514300,"position":"bottom","activity_title":"第五人格新监管者隐士活动","cover":"https://i0.hdslb.com/bfs/live/e7870123c939a3b4b4c0665166fae07380d71e84.png","jump_url":"https://www.bilibili.com/blackboard/dynamic/309491?-Abrowser=live\u0026is_live_half_webview=1\u0026hybrid_rotate_d=1\u0026is_cling_player=1\u0026hybrid_half_ui=1,3,100p,70p,0,1,30,100;2,2,375,100p,0,1,30,100;3,3,100p,70p,0,1,30,100;4,2,375,100p,0,1,30,100;5,3,100p,70p,0,1,30,100;6,3,100p,70p,0,1,30,100;7,3,100p,70p,0,1,30,100;8,3,100p,70p,0,1,30,100","is_close":1,"action":"update"}]}}
						case "GUARD_ACHIEVEMENT_ROOM":
							// {"cmd":"GUARD_ACHIEVEMENT_ROOM","data":{"anchor_basemap_url":"https://i0.hdslb.com/bfs/live/f873a04b1544d8f8bcc37fb2924ac9a2c2554031.png","anchor_guard_achieve_level":100,"anchor_modal":{"first_line_content":"恭喜当前舰队规模突破\u003c%100%\u003e","highlight_color":"#00DCFF","second_line_content":"至直播中心 - 获奖记录填写收货信息可获得实物勋章奖励哦～","show_time":5},"app_basemap_url":"https://i0.hdslb.com/bfs/live/83008812e86cae42049414e965d6ab6002f061cb.png","current_achievement_level":2,"dmscore":8,"event_type":1,"face":"http://i1.hdslb.com/bfs/face/6e5235459bfb8e0cbdb0e6357524abbad7f7f0bc.jpg","first_line_content":"恭喜主播\u003c%希侑Kiyuu%\u003e","first_line_highlight_color":"#F2AE09","first_line_normal_color":"#FFFFFF","headmap_url":"https://i0.hdslb.com/bfs/vc/071eb10548fe9bc482ff69331983d94192ce9507.png","is_first":true,"is_first_new":false,"room_id":23805066,"second_line_content":"舰队规模突破\u003c%100%\u003e","second_line_highlight_color":"#06DDFF","second_line_normal_color":"#FFFFFF","show_time":3,"web_basemap_url":"https://i0.hdslb.com/bfs/live/83008812e86cae42049414e965d6ab6002f061cb.png"}}
						case "ROOM_SKIN_MSG":
							// {"cmd":"ROOM_SKIN_MSG","skin_id":65,"status":1,"end_time":2145888000,"current_time":1661515440,"only_local":false,"scatter":{"min":1,"max":200},"skin_config":{"android":{"1":{"zip":"https:\/\/i0.hdslb.com\/bfs\/live\/roomSkin\/d50490b2fb05cc32fe69a9ea40839fd68c575738.zip","md5":"1E111556D18406698350C007828EA0F8"}},"ios":{"1":{"zip":"https:\/\/i0.hdslb.com\/bfs\/live\/roomSkin\/8cc833f1e5e9caac5afd0f0d494dbb79e505b06e.zip","md5":"0781BED8AC8F09D18EB1F2091F211A82"}},"ipad":{"1":{"zip":"https:\/\/i0.hdslb.com\/bfs\/live\/roomSkin\/c814b4bc96e0ceae0be7c70300a49e2cdc9ccecc.zip","md5":"ED14E47E9D32FABB86AE857E8CBD1D1D"}},"web":{"1":{"zip":"https:\/\/i0.hdslb.com\/bfs\/live\/roomSkin\/bfac22ed069c4d41a6b9e2a305c9efd58bc07137.zip","md5":"9AE7E63C79466165ED03131CA6C62D1C","platform":"web","version":"1","headInfoBgPic":"https:\/\/i0.hdslb.com\/bfs\/live\/roomSkin\/7ba5a32cda0f985aa02bb05f453eac1f03cb976d.png","giftControlBgPic":"https:\/\/i0.hdslb.com\/bfs\/live\/roomSkin\/f864809e6be7b3fe834de6d37b5b7f42b2cdff2a.png","rankListBgPic":"https:\/\/i0.hdslb.com\/bfs\/live\/roomSkin\/00d1718591af1b5117c588f7bac1efb6c1e97fde.png","mainText":"#FFFFD432","normalText":"#FF999999","highlightContent":"#FFFFD432","border":"#33999999"}}}}
						case "PLAY_TAG":
							// {"cmd":"PLAY_TAG","data":{"tag_id":59095,"pic":"https://i0.hdslb.com/bfs/live/3c26626a30fdb70e44e16fd4313fa02785486e30.png","timestamp":1661517234,"type":"ADD"}}
						case "SPECIAL_GIFT":
							// {"cmd":"SPECIAL_GIFT","data":{"39":{"action":"start","content":"前方高能预警，注意这不是演习","hadJoin":0,"id":"3352122929875","num":1,"storm_gif":"http://static.hdslb.com/live-static/live-room/images/gift-section/mobilegift/2/jiezou.gif?2017011901","time":90}}}
						case "VOICE_JOIN_STATUS":
							// {"cmd":"VOICE_JOIN_STATUS","data":{"room_id":6535302,"status":1,"channel":"919003","channel_type":"voice","uid":399963039,"user_name":"糖八ks","head_pic":"http://i0.hdslb.com/bfs/face/67d0fa7c9ce194a3d106ed4f82b13df9d86363c1.jpg","guard":0,"start_at":1661518050,"current_time":1661518050,"web_share_link":"https://live.bilibili.com/h5/6535302"},"room_id":6535302}
						case "VIDEO_CONNECTION_JOIN_END":
							//  {"cmd":"VIDEO_CONNECTION_JOIN_END","data":{"channel_id":"72057594038846994","start_at":1661520034,"toast":"主播结束了与澈屿Don的连线.","current_time":1661520034},"roomid":23144336}
						case "VIDEO_CONNECTION_MSG":
							// {"cmd":"VIDEO_CONNECTION_MSG","data":{"channel_id":"72057594038846994","current_time":1661520034,"dmscore":4,"toast":"主播结束了视频连线"}}
						case "WIDGET_WISH_LIST":
							// {"cmd":"WIDGET_WISH_LIST","data":{"wish":[{"type":3,"gift_id":10003,"gift_name":"舰长","gift_img":"https://i0.hdslb.com/bfs/live/f1be2a2d5b227ce72641de1ad64bcc7f9e4111c3.png","gift_price":198000,"target_num":2,"current_num":0},{"type":2,"gift_id":31164,"gift_name":"粉丝团灯牌","gift_img":"https://s1.hdslb.com/bfs/live/cbed3bb0a894369b49ceaf0b5337b4491b75ac42.png","gift_price":1000,"target_num":88,"current_num":22},{"type":2,"gift_id":31075,"gift_name":"守护之翼","gift_img":"https://s1.hdslb.com/bfs/live/1d7d973972e70cad7e97478b3c8d20b0faafd0dc.png","gift_price":200000,"target_num":3,"current_num":3}],"wish_status":1,"sid":929,"wish_status_info":[{"wish_status_msg":"设定心愿","wish_status_img":"https://i0.hdslb.com/bfs/live/38f82bac32794e79776f7371269453652bd58a87.png","wish_status":0},{"wish_status_msg":"达成","wish_status_img":"https://i0.hdslb.com/bfs/live/1dae635924437239fc69e561a1a9467508521249.png","wish_status":2},{"wish_status_msg":"收集失败","wish_status_img":"https://i0.hdslb.com/bfs/live/3bbd30fdd32d085cc90e9ccd98c65a886dca9a8f.png","wish_status":3}],"wish_name":"心愿"}}
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

func getCMD(curBody []byte) (string, error) {
	var msg Msg
	err := json.Unmarshal(curBody, &msg)
	if err != nil {
		return "", err
	}
	return msg.Cmd, nil
}

func (b *Bot) getDanmakuInfo() (*DanmakuInfoResp, error) {
	b.DEBUG("弹幕池情报请求")
	resp, err := GetResponse(fmt.Sprintf(b.infoURL, b.RoomID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var danmakuInfoResp DanmakuInfoResp
	decoder := json.NewDecoder(resp.Body)
	err2 := decoder.Decode(&danmakuInfoResp)
	if err2 != nil {
		return nil, err
	}
	b.DEBUG("连接情报已确保")
	return &danmakuInfoResp, nil
}
