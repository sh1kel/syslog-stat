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
	avgcountCh    = make(chan domainDelay, 100)
	syslogChannel = make(syslog.LogPartsChannel)
	logwriter     *nativesyslog.Writer
	gMsgCounter   int64
)

// структура для отправки в канал - домен/задержка
type domainDelay struct {
	domain string
	delay  float64
}

// общая структура со статистикой
type delays struct {
	dTable map[string][]float64
	mtx    sync.RWMutex
}

// структура для отправки в канал id сессии/payload
type logMsg struct {
	sessionId string
	payload   string
}

// инициализация структуры задержек
func DelayInit() *delays {
	return &delays{
		dTable: make(map[string][]float64),
	}
}

// парсинг первоначально приходящей записи
func parseMessage(msg string) (ok bool, header, payload string) {
	split := strings.SplitN(msg, ":", 2)
	if len(split) < 2 {
		return false, "", ""
	}
	header = split[0]
	if len(header) > 12 {
		return false, "", ""
	}
	payload = split[1]
	if payload == " message-id=<>" || payload == " removed" {
		return false, "", ""
	}
	return true, strings.TrimSpace(header), strings.TrimSpace(payload)
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
	cleanChan := make(chan string, 100)
	exportChan := make(chan *emailMessage, 100)

	ticker := time.NewTicker(90 * time.Second)

	handler := syslog.NewChannelHandler(syslogChannel)
	domainDelays := DelayInit()

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	server.ListenUDP("127.0.0.1:5141")
	server.Boot()

	http.HandleFunc("/stat", webStat)
	http.HandleFunc("/delays/", domainDelays.avgDelay)
	http.HandleFunc("/domains", domainDelays.listDomains)
	//http.HandleFunc("/debug", msgList.webDebug)

	go http.ListenAndServe("127.0.0.1:8081", nil)

	go proccessLogChannel(cleanChan, exportChan, ticker)
	go WriteOut(cleanChan, exportChan)
	go countAverageDelay(domainDelays)

	server.Wait()
	ticker.Stop()
}
