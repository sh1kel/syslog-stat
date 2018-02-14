package main

import (
	"strconv"
	"strings"
	"sync"
)

type emailMessage struct {
	From       string
	To         string
	Relay      string
	Delay      float32
	StatusCode string
	StatusMsg  string
	rawString  []string
	mtx        sync.RWMutex
}

// загрузка информации о сообщении
func (m *messageList) Load(key string) *emailMessage {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	return m.Messages[key]
}

// сохранение/апдейт записи, отправка в каналы вывода
func (m *messageList) Save(key, val string) {
	if val != "removed" {
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
		}
	}
}

// удаление записи-сообщения из очереди
func (m *messageList) Delete(key string) {
	m.mtx.Lock()
	delete(m.Messages, key)
	m.mtx.Unlock()
}

// проверка заполненности записи о сообщении
func (m *messageList) CheckComplete(key string) bool {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	if m.Messages[key].To != "" {
		return true
	} else {
		return false
	}
}

// апдейт записи
func (msg *emailMessage) UpdateMessage(logRecord string) {
	// Добавляем отрпавителя
	if strings.HasPrefix(logRecord, "from=") {
		stringParts := strings.SplitN(logRecord, ",", 2)
		from := strings.TrimPrefix(stringParts[0], "from=<")
		msg.mtx.Lock()
		msg.From = strings.TrimSuffix(from, ">")
		msg.rawString = append(msg.rawString, logRecord)
		msg.mtx.Unlock()
		return
	}
	// Добавляем получателя, релей, задержку, статус отправки
	if strings.HasPrefix(logRecord, "to=") {
		var domain string
		stringParts := strings.Split(logRecord, ",")
		to := strings.TrimPrefix(stringParts[0], "to=<")
		relay := strings.TrimPrefix(stringParts[1], " relay=")
		domainList := strings.Split(strings.Split(relay, "[")[0], ".")
		switch domainLen := len(domainList); domainLen {
		case 2:
			domain = domainList[0] + "." + domainList[1]
		case 1:
			domain = domainList[0]
		default:
			domain = domainList[len(domainList)-2] + "." + domainList[len(domainList)-1]
		}
		RawStatus := stringParts[5]
		fullStatus := strings.SplitN(RawStatus, "=", 2)[1]
		splitStatuses := strings.SplitN(fullStatus, " ", 2)
		delay, _ := strconv.ParseFloat(strings.Split(stringParts[2], "=")[1], 32)

		msg.mtx.Lock()
		msg.To = strings.TrimSuffix(to, ">")
		//msg.Relay = strings.Split(relay, "[")[0]
		msg.Relay = domain
		msg.Delay = float32(delay)
		msg.StatusCode = splitStatuses[0]
		msg.StatusMsg = splitStatuses[1]
		msg.rawString = append(msg.rawString, logRecord)
		msg.mtx.Unlock()
		return
	}
	msg.mtx.Lock()
	msg.rawString = append(msg.rawString, logRecord)
	msg.mtx.Unlock()

}
