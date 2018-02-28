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
	SessionId  string
	mtx        sync.RWMutex
}

// апдейт записи
func (msg *emailMessage) UpdateMessage(sessionID, logRecord string) {
	if strings.HasPrefix(logRecord, "from=") {
		stringParts := strings.SplitN(logRecord, ",", 2)
		from := strings.TrimPrefix(stringParts[0], "from=<")
		msg.mtx.Lock()
		msg.From = strings.TrimSuffix(from, ">")
		msg.mtx.Unlock()
		return
	} else if strings.HasPrefix(logRecord, "to=") {
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
				log.Printf("Error log part: %s\n", logRecord)
			}
			splitStatuses := strings.SplitN(strStatus, " ", 2)
			if len(splitStatuses) > 1 {
				StatusMsg = splitStatuses[1]
			} else {
				StatusMsg = "Unknown"
				log.Printf("Error log part: %s\n", logRecord)
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
		msg.mtx.Unlock()
		return

	}
	msg.mtx.Lock()
	if msg.SessionId == "" {
		msg.SessionId = sessionID
	}
	msg.UpdateTime = int32(time.Now().Unix())
	msg.rawString = append(msg.rawString, logRecord)
	msg.mtx.Unlock()

}
