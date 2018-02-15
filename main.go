package main

import (
	"gopkg.in/mcuadros/go-syslog.v2"
	"log"
	nativesyslog "log/syslog"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	channel    = make(syslog.LogPartsChannel)
	exportChan = make(chan string, 100)
	parseChan  = make(chan *logMsg, 100)
	cleanChan  = make(chan string, 100)
	avgcountCh = make(chan domainDelay, 100)
	logwriter  *nativesyslog.Writer
)

// структура для отправки в канал - домен/задержка
type domainDelay struct {
	domain string
	delay  float32
}

// общая структура со статистикой
type delays struct {
	dTable map[string][]float32
	mtx    sync.RWMutex
}

// общая структура с информацией по письмам
type messageList struct {
	Messages     map[string]*emailMessage
	msgProccesed int64
	mtx          sync.RWMutex
}

// структура для отправки в канал id сессии/payload
type logMsg struct {
	sessionId string
	payload   string
}

// инициализация структуры очереди
func QueueInit() *messageList {
	return &messageList{
		Messages: make(map[string]*emailMessage),
	}
}

// инициализация структуры задержек
func DelayInit() *delays {
	return &delays{
		dTable: make(map[string][]float32),
	}
}

// парсинг первоначально приходящей записи
func parseMessage(msg string) (ok bool, header, payload string) {
	split := strings.SplitN(msg, ":", 2)
	if len(split) < 2 {
		return false, "", ""
	}
	header = split[0]
	if len(header) != 12 {
		return false, "", ""
	}
	payload = split[1]
	if payload == " message-id=<>" || payload == " removed" {
		return false, "", ""
	}
	return true, strings.TrimSpace(header), strings.TrimSpace(payload)
}

func cleanStuckedMessages(ticker time.Ticker, msgList *messageList) {
	for _ = range ticker.C {
		Now := int32(time.Now().Unix())
		msgList.mtx.RLock()
		for key, _ := range msgList.Messages {
			if (Now - msgList.Messages[key].UpdateTime) > 600 {
				cleanChan <- key
			}
		}
		msgList.mtx.RUnlock()
	}
}

func init() {
	var err error
	runtime.GOMAXPROCS(16)
	// категория, в которую пишутся payment логи (/etc/rsyslog.d/50-default.conf)
	logwriter, err = nativesyslog.New(nativesyslog.LOG_LOCAL4, "syslog-go")
	if err == nil {
		log.SetOutput(logwriter)
	}
	logwriter.Info("Starting syslog-go server")
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
}

func main() {
	msgList := QueueInit()
	handler := syslog.NewChannelHandler(channel)
	domainDelays := DelayInit()
	ticker := time.NewTicker(5 * time.Minute)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	server.ListenUDP("127.0.0.1:5141")
	server.Boot()

	http.HandleFunc("/stat", msgList.webStat)
	http.HandleFunc("/delays/", domainDelays.avgDelay)
	http.HandleFunc("/domains", domainDelays.listDomains)
	http.HandleFunc("/debug", msgList.webDebug)

	go http.ListenAndServe("127.0.0.1:8081", nil)

	go proccessLogChannel()
	go ParseQueue(msgList)
	go WriteOut(msgList)
	go cleanStuckedMessages(*ticker, msgList)
	go countAverageDelay(domainDelays)
	go cleanQueue(msgList)

	server.Wait()
	ticker.Stop()
}
