package main

import (
	"fmt"
	"gopkg.in/mcuadros/go-syslog.v2"
	"runtime"
	"strings"
	"sync"
)

var (
	channel    = make(syslog.LogPartsChannel)
	exportChan = make(chan string, 100)
	parseChan  = make(chan *logMsg, 100)
	cleanChan  = make(chan string, 100)
)

type emailMessage struct {
	From       string
	To         string
	Relay      string
	Delay      string
	StatusCode string
	StatusMsg  string
	RawRecord  []string
	mtx        sync.RWMutex
}

type messageList struct {
	Messages map[string]*emailMessage
	mtx      sync.RWMutex
}

type logMsg struct {
	sessionId string
	payload   string
}

func (m *messageList) Load(key string) *emailMessage {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	return m.Messages[key]
}

func (m *messageList) Save(key, val string) {
	m.mtx.RLock()
	_, ok := m.Messages[key]
	m.mtx.RUnlock()

	if ok {
		m.Messages[key].UpdateMessage(val)
	} else {
		m.mtx.Lock()
		m.Messages[key] = &emailMessage{}
		m.mtx.Unlock()
		m.Messages[key].UpdateMessage(val)
	}
	if m.CheckComplete(key) {
		exportChan <- key
		go fmt.Printf("Q len: %d\n", len(m.Messages))
	}
}

func (m *messageList) Delete(key string) {
	m.mtx.Lock()
	delete(m.Messages, key)
	m.mtx.Unlock()
}

func (m *messageList) CheckComplete(key string) bool {
	// m.Messages[key].mtx.RLock()
	// defer m.Messages[key].mtx.RUnlock()
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	if m.Messages[key].From != "" {
		if m.Messages[key].To != "" {
			if m.Messages[key].Relay != "" {
				if m.Messages[key].Delay != "" {
					if m.Messages[key].StatusCode != "" {
						if m.Messages[key].StatusMsg != "" {
							// return true

							if len(m.Messages[key].RawRecord) == 6 {
								return true
							} else {
								return false
							}

						} else {
							return false
						}
					} else {
						return false
					}
				} else {
					return false
				}
			} else {
				return false
			}
		} else {
			return false
		}
	} else {
		return false
	}
}

func Init() *messageList {
	return &messageList{
		Messages: make(map[string]*emailMessage),
	}
}

func parseMessage(msg string) (header, payload string) {
	split := strings.Split(msg, ":")
	if len(split) < 2 {
		return "", strings.TrimSpace(msg)
	}
	if len(split) > 2 {
		header = split[0]
		payload = strings.Join(split[1:], " ")
		return strings.TrimSpace(header), strings.TrimSpace(payload)
	}
	header = split[0]
	payload = split[1]
	return strings.TrimSpace(header), strings.TrimSpace(payload)
}

func (msg *emailMessage) UpdateMessage(logRecord string) {
	// Добавляем отрпавителя
	if strings.HasPrefix(logRecord, "from=") {
		stringParts := strings.SplitN(logRecord, ",", 2)
		from := strings.TrimPrefix(stringParts[0], "from=<")
		msg.mtx.Lock()
		msg.From = strings.TrimSuffix(from, ">")
		msg.RawRecord = append(msg.RawRecord, logRecord)
		msg.mtx.Unlock()
		return
	}
	// Добавляем получателя, релей, задержку, статус отправки
	if strings.HasPrefix(logRecord, "to=") {
		stringParts := strings.Split(logRecord, ",")
		to := strings.TrimPrefix(stringParts[0], "to=<")
		relay := strings.TrimPrefix(stringParts[1], " relay=")
		RawStatus := stringParts[5]
		fullStatus := strings.SplitN(RawStatus, "=", 2)[1]
		splitStatuses := strings.SplitN(fullStatus, " ", 2)

		msg.mtx.Lock()
		msg.To = strings.TrimSuffix(to, ">")
		msg.Relay = strings.Split(relay, "[")[0]
		msg.Delay = strings.Split(stringParts[2], "=")[1]
		msg.StatusCode = splitStatuses[0]
		msg.StatusMsg = splitStatuses[1]
		msg.RawRecord = append(msg.RawRecord, logRecord)
		msg.mtx.Unlock()
		return
	}
	// Добавляем строку в любом случае
	msg.mtx.Lock()
	msg.RawRecord = append(msg.RawRecord, logRecord)
	msg.mtx.Unlock()
}

// Очередь на вывод (syslog и zabbix)
func WriteOut(msgList *messageList) {
	for key := range exportChan {
		msg := msgList.Load(key)
		fmt.Printf("From: %s To: %s Relay: %s Delay: %s Status: %s\n", msg.From, msg.To, msg.Relay, msg.Delay, msg.StatusCode) //, msg.RawRecord)
		cleanChan <- key
	}
}

// Основная очередь обработки сообщений
func proccessParseQueue(msgList *messageList) {
	for msg := range parseChan {
		msgList.Save(msg.sessionId, msg.payload)
	}
}

// Очередь сообщений для удаления
func cleanQueue(msgList *messageList) {
	for key := range cleanChan {
		msgList.Delete(key)
	}
}

func main() {
	msgList := Init()
	runtime.GOMAXPROCS(16)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	server.ListenUDP("0.0.0.0:5140")
	server.Boot()

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			var logMessage = logMsg{}
			msg := fmt.Sprint(logParts["message"])
			logMessage.sessionId, logMessage.payload = parseMessage(msg)
			parseChan <- &logMessage
			go proccessParseQueue(msgList)
			go WriteOut(msgList)
			go cleanQueue(msgList)
		}
	}(channel)

	server.Wait()

}
