package main

import (
	"fmt"
	"gopkg.in/mcuadros/go-syslog.v2"
	"strconv"
	"strings"
)

type emailMessage struct {
	From       string
	To         string
	Relay      string
	Delay      int
	StatusCode string
	StatusMsg  string
	RawRecord  []string
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
		msg.From = strings.TrimSuffix(from, ">")
		msg.RawRecord = append(msg.RawRecord, logRecord)
		return
	}
	// Добавляем получателя, релей, задержку, статус отправки
	if strings.HasPrefix(logRecord, "to=") {
		var err error
		stringParts := strings.Split(logRecord, ",")
		to := strings.TrimPrefix(stringParts[0], "to=<")
		msg.To = strings.TrimSuffix(to, ">")
		relay := strings.TrimPrefix(stringParts[1], "relay=")
		msg.Relay = strings.Split(relay, "[")[0]
		msg.Delay, err = strconv.Atoi(strings.Split(stringParts[2], "=")[1])
		if err != nil {
			msg.Delay = -1
		}
		RawStatus := stringParts[5]
		fullStatus := strings.SplitN(RawStatus, "=", 2)[1]
		splitStatuses := strings.SplitN(fullStatus, " ", 2)
		msg.StatusCode = splitStatuses[0]
		msg.StatusMsg = splitStatuses[1]
		msg.RawRecord = append(msg.RawRecord, logRecord)
		return
	}
	msg.RawRecord = append(msg.RawRecord, logRecord)
}

func main() {
	msgList := make(map[string]emailMessage)
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	server.ListenUDP("0.0.0.0:5140")
	server.Boot()

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			msg := fmt.Sprint(logParts["message"])
			h, p := parseMessage(msg)
			msgList[h].UpdateMessage(p)
		}
	}(channel)

	server.Wait()

}
