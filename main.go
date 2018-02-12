package main

import (
	"fmt"
	"gopkg.in/mcuadros/go-syslog.v2"
	"strings"
	"sync"
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

func (m *messageList) Load(key string) *emailMessage {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	return m.Messages[key]
}

func (m *messageList) Save(key, val string) {
	m.mtx.Lock()
	_, ok := m.Messages[key]
	if ok {
		m.Messages[key].UpdateMessage(val)
	} else {
		m.Messages[key] = &emailMessage{}
		m.Messages[key].UpdateMessage(val)
	}
	m.mtx.Unlock()
}

func (m *messageList) Delete(key string) {
	m.mtx.Lock()
	delete(m.Messages, key)
	m.mtx.Unlock()
}

func (m *messageList) Range() {

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

func WriteOut(ch chan *emailMessage) {
	for msg := range ch {
		if msg != nil {
			fmt.Printf("MSG: %v\nChannel len: %d\n", *msg, len(ch))
		}
	}
}

func main() {
	msgList := Init()
	channel := make(syslog.LogPartsChannel)
	outChan := make(chan *emailMessage, 10)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	server.ListenUDP("0.0.0.0:5140")
	server.Boot()

	// go func(channel syslog.LogPartsChannel) {
	for logParts := range channel {
		msg := fmt.Sprint(logParts["message"])
		h, p := parseMessage(msg)
		go msgList.Save(h, p)
		outChan <- msgList.Load(h)
		go WriteOut(outChan)
	}
	// }(channel)

	server.Wait()

}
