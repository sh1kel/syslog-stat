package main

import (
	"fmt"
	"gopkg.in/mcuadros/go-syslog.v2"
	"strings"
)

type emailMessage struct {
	From   string
	To     string
	Relay  string
	Delay  string
	Status string
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

func main() {
	// sgList := make(map[string]emailMessage)
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	server.ListenUDP("0.0.0.0:5140")
	server.Boot()

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			//fmt.Println(logParts["message"])
			msg := fmt.Sprint(logParts["message"])
			h, p := parseMessage(msg)
			fmt.Printf("Message:\n%s\n%s\n", h, p)
		}
	}(channel)

	server.Wait()

}
