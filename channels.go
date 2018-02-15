package main

import "fmt"

// Вывод zabbix и в канал подсчета средней задержки
func WriteOut(msgList *messageList) {
	for key := range exportChan {
		msg := msgList.Load(key)
		// выводим в сислог только сообщения от payment@mail.youdo.com
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

// парсинг полученной от сислога строки, разбиение на id сессии и полезную нагрузку
func proccessLogChannel() {
	for logParts := range channel {
		var logMessage = logMsg{}
		msg := fmt.Sprint(logParts["message"])
		logMessage.sessionId, logMessage.payload = parseMessage(msg)
		parseChan <- &logMessage
	}
}

// подсчет средней задержки
func countAverageDelay(domainDelays *delays) {
	for delays := range avgcountCh {
		domainDelays.mtx.RLock()
		_, ok := domainDelays.dTable[delays.domain]
		domainDelays.mtx.RUnlock()
		if ok {
			domainDelays.mtx.Lock()
			qLen := len(domainDelays.dTable[delays.domain])
			// если количество метрик больше 200к, удаляем более старые 100к
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
