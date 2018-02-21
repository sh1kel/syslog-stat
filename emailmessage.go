package main

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// структура с информацией о сообщении
type emailMessage struct {
	From       string
	To         string
	Relay      string
	Delay      float64
	StatusCode string
	StatusMsg  string
	UpdateTime int32
	rawString  []string
	mtx        sync.RWMutex
}

// метод - загрузка информации о сообщении
func (m *messageList) Load(key string) *emailMessage {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	return m.Messages[key]
}

// метод - сохранение/апдейт записи, отправка в каналы вывода
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
	}
}

// метод - удаление записи-сообщения из очереди
func (m *messageList) Delete(key string) {
	m.mtx.Lock()
	delete(m.Messages, key)
	m.mtx.Unlock()
}

// метод - проверка заполненности записи о сообщении
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
		var domain, RawStatus, strStatus, StatusMsg string
		var delay float64
		stringParts := strings.Split(logRecord, ",")
		partLen := len(stringParts)
		to := strings.TrimPrefix(stringParts[0], "to=<")
		if partLen > 4 {
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
			RawStatus = stringParts[5]
			fullStatus := strings.SplitN(RawStatus, "=", 2)
			if len(fullStatus) > 1 {
				strStatus = fullStatus[1]
			} else {
				log.Printf("Error log part: %v\n", fullStatus)
			}
			splitStatuses := strings.SplitN(strStatus, " ", 2)
			if len(splitStatuses) > 1 {
				StatusMsg = splitStatuses[1]
			} else {
				StatusMsg = "Unknown"
				log.Printf("Error log part: %v\n", splitStatuses)
			}
			delay, _ = strconv.ParseFloat(strings.Split(stringParts[2], "=")[1], 32)
		} else {
			log.Printf("Error log record: %s\n", logRecord)
		}
		msg.mtx.Lock()
		msg.To = strings.TrimSuffix(to, ">")
		msg.Relay = strings.ToLower(domain)
		msg.Delay = delay
		msg.StatusCode = StatusMsg
		msg.rawString = append(msg.rawString, logRecord)
		msg.mtx.Unlock()
		return

	}
	msg.mtx.Lock()
	msg.UpdateTime = int32(time.Now().Unix())
	msg.rawString = append(msg.rawString, logRecord)
	msg.mtx.Unlock()

}
