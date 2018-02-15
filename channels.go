package main

import "fmt"

// Вывод zabbix
func WriteOut(msgList *messageList) {
	for key := range exportChan {
		msg := msgList.Load(key)
		if msg.From == "payment@mail.youdo.com" {
			for s := 0; s < len(msg.rawString); s++ {
				logwriter.Info(key + ": " + msg.rawString[s])
			}
		}
		avgcountCh <- domainDelay{msg.Relay, msg.Delay}
		cleanChan <- key
	}
}

// Основная очередь обработки сообщений
func ParseQueue(msgList *messageList) {
	for msg := range parseChan {
		msgList.Save(msg.sessionId, msg.payload)
		msgList.mtx.Lock()
		msgList.msgProccesed++
		msgList.mtx.Unlock()
	}
}

// Очередь сообщений для удаления
func cleanQueue(msgList *messageList) {
	for key := range cleanChan {
		msgList.Delete(key)
	}
}

func proccessLogChannel() {
	for logParts := range channel {
		var logMessage = logMsg{}
		msg := fmt.Sprint(logParts["message"])
		logMessage.sessionId, logMessage.payload = parseMessage(msg)
		parseChan <- &logMessage
	}
}

func countAverageDelay(domainDelays *delays) {
	for delays := range avgcountCh {
		domainDelays.mtx.RLock()
		_, ok := domainDelays.dTable[delays.domain]
		domainDelays.mtx.RUnlock()
		if ok {
			domainDelays.mtx.Lock()
			qLen := len(domainDelays.dTable[delays.domain])
			if qLen > 200000 {
				newDelays := domainDelays.dTable[delays.domain][qLen/2:]
				domainDelays.dTable[delays.domain] = newDelays
			}
			domainDelays.dTable[delays.domain] = append(domainDelays.dTable[delays.domain], delays.delay)
			domainDelays.mtx.Unlock()
		} else {
			domainDelays.mtx.Lock()
			domainDelays.dTable[delays.domain] = []float32{delays.delay}
			domainDelays.mtx.Unlock()
		}
	}
}
