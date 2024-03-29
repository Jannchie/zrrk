package zrrk

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/net/proxy"
)

type BiliHeader struct {
	PackL int32
	HeadL int16
	BodyV int16
	OpeaT int32
	Seque int32
}

func WriteToFile(msg string) {
	log.Printf("进行文字的输出: %s", msg)
	ioutil.WriteFile("../message.txt", []byte(msg), 0644)
}
func CleanFile() {
	log.Printf("清理文件")
	ioutil.WriteFile("../message.txt", []byte(" "), 0644)
}

func getStringWidth(str string) int {
	var ans int
	for _, c := range str {
		if unicode.IsLower(c) || unicode.IsUpper(c) || unicode.IsDigit(c) || unicode.IsPunct(c) {
			ans += 1
		} else {
			ans += 2
		}
	}
	return ans
}

func IsSameDay(t time.Time) bool {
	yp, mp, dp := time.Unix(t.Unix(), 0).Date()
	y, m, d := time.Unix(time.Now().Unix(), 0).Date()
	isSameDay := yp == y && m == mp && d == dp
	return isSameDay
}

func NewClient() (*http.Client, error) {
	proxyURL := os.Getenv("PROXY_URL")
	if proxyURL == "" {
		return http.DefaultClient, nil
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	d, err := proxy.FromURL(u, proxy.Direct)
	if err != nil {
		return nil, err
	}
	t := &http.Transport{Dial: d.Dial}
	return &http.Client{Transport: t}, nil
}
func GetResponse(targetURL string) (*http.Response, error) {
	client, err := NewClient()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36")
	return client.Do(req)
}

func HeadGen(datalength, Opeation, Sequence int) []byte {
	var buffer bytes.Buffer
	buffer.Write(Itob32(int32(datalength + WS_PACKAGE_HEADER_TOTAL_LENGTH)))
	buffer.Write(Itob16(WS_PACKAGE_HEADER_TOTAL_LENGTH))
	buffer.Write(Itob16(WS_HEADER_DEFAULT_VERSION))
	buffer.Write(Itob32(int32(Opeation)))
	buffer.Write(Itob32(int32(Sequence)))
	return buffer.Bytes()
}

func GetHeader(rawHead []byte) *BiliHeader {
	if len(rawHead) != WS_PACKAGE_HEADER_TOTAL_LENGTH {
		log.Println(errors.New("invalid header length"))
	}
	PackL := Btoi32(rawHead, WS_PACKAGE_OFFSET)
	HeadL := Btoi16(rawHead, WS_HEADER_OFFSET)
	BodyV := Btoi16(rawHead, WS_VERSION_OFFSET)
	OpeaT := Btoi32(rawHead, WS_OPERATION_OFFSET)
	Seque := Btoi32(rawHead, WS_SEQUENCE_OFFSET)
	return &BiliHeader{
		PackL: PackL,
		HeadL: HeadL,
		BodyV: BodyV,
		OpeaT: OpeaT,
		Seque: Seque,
	}
}

func btoi32(b []byte) int32 {
	var buffer int32
	err := binary.Read(bytes.NewReader(b), binary.BigEndian, &buffer)
	if err != nil {
		log.Println(err)
	}
	return buffer
}

func btoi16(b []byte) int16 {
	var buffer int16
	err := binary.Read(bytes.NewReader(b), binary.BigEndian, &buffer)
	if err != nil {
		log.Println(err)
	}
	return buffer
}

func Btoi32(b []byte, offset int) int32 {
	return btoi32(b[offset : offset+4])
}

func Btoi16(b []byte, offset int) int16 {
	return btoi16(b[offset : offset+2])
}

func Itob32(num int32) []byte {
	var buffer bytes.Buffer
	err := binary.Write(&buffer, binary.BigEndian, num)
	if err != nil {
		log.Println(err)
	}
	return buffer.Bytes()
}

func Itob16(num int16) []byte {
	var buffer bytes.Buffer
	err := binary.Write(&buffer, binary.BigEndian, num)
	if err != nil {
		log.Println(err)
	}
	return buffer.Bytes()
}

var (
	zr io.ReadCloser
	r  *bytes.Reader
	m  sync.Mutex
)

func ZlibParse(rawBody []byte) []byte {
	m.Lock()
	defer m.Unlock()
	if zr == nil {
		r = bytes.NewReader(rawBody)
		zr, _ = zlib.NewReader(r)
	} else {
		r.Reset(rawBody)
		err := zr.(zlib.Resetter).Reset(r, nil)
		if err != nil {
			log.Println("解压错误: ", err)
		}
	}
	buf := bytes.NewBuffer(nil)
	if _, err := buf.ReadFrom(zr); err != nil {
		log.Println("解压错误: ", err)
	}
	body := buf.Bytes()
	return body
}

func ContainStrings(s ...string) bool {
	if len(s) < 2 {
		log.Println("参数不足")
	}
	for i := 1; i < len(s); i++ {
		if flag := strings.Contains(s[0], s[i]); flag {
			return true
		}
	}
	return false
}
