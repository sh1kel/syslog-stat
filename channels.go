package main

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Вывод zabbix и в канал подсчета средней задержки
func writeOut(cleanChan chan string, exportChan chan *emailMessage, avgcountCh chan *domainDelay, wg *sync.WaitGroup) {
	defer wg.Done()

	for msg := range exportChan {
		msg.mtx.RLock()
		// выводим в сислог только сообщения от payment@mail.youdo.com
		if msg.From == "user@mail.tld" {
			for s := 0; s < len(msg.rawString); s++ {
				logwriter.Info(msg.SessionId + ": " + msg.rawString[s])
			}
		}
		avgcountCh <- &domainDelay{msg.Relay, msg.Delay}
		cleanChan <- msg.SessionId
		msg.mtx.RUnlock()
	}
	logwriter.Info("Stopping writer.")
	close(avgcountCh)
}

// парсинг полученной от сислога строки
func proccessLogChannel(cleanChan chan string, exportChan chan *emailMessage, ticker *time.Ticker, ctlCh chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	m := make(map[string]*emailMessage)

	var pool = sync.Pool{
		New: func() interface{} {
			return &emailMessage{}
		},
	}

	for {
		select {
		case logParts := <-syslogChannel:
			var logMessage = logMsg{}
			var ok = false
			msg := fmt.Sprint(logParts["message"])
			ok, logMessage.sessionId, logMessage.payload = parseMessage(msg)

			if ok {
				_, ok = m[logMessage.sessionId]
				if !ok {
					messagePtr := pool.Get().(*emailMessage)
					m[logMessage.sessionId] = messagePtr
					//m[logMessage.sessionId] = &emailMessage{}
				}
				m[logMessage.sessionId].UpdateMessage(logMessage.sessionId, logMessage.payload)
				if m[logMessage.sessionId].To != "" {
					gMsgCounter++
					exportChan <- m[logMessage.sessionId]
				}
			}
			atomic.StoreUint32(&gQueueLen, uint32(len(m)))
		case key := <-cleanChan:
			val, ok := m[key]
			if ok {
				m[key].Clean()
				delete(m, key)
				pool.Put(val)
			}
			atomic.StoreUint32(&gQueueLen, uint32(len(m)))
		case <-ticker.C:
			i := 0
			Now := int32(time.Now().Unix())
			for key, val := range m {
				if (Now - val.UpdateTime) > 300 {
					m[key].Clean()
					delete(m, key)
					pool.Put(val)
					i++
				}
			}
			logwriter.Info("Deleted old objects: " + strconv.Itoa(i))
			atomic.StoreUint32(&gQueueLen, uint32(len(m)))
		case <-ctlCh:
			logwriter.Info("Stopping parser.")
			close(exportChan)
			return
		}
	}
}

// подсчет средней задержки
func countAverageDelay(domainDelays *delaysMap, avgCountChan chan *domainDelay, wg *sync.WaitGroup) {
	defer wg.Done()
	for delays := range avgCountChan {
		domainDelays.mtx.RLock()
		_, ok := domainDelays.dTable[delays.domain]
		domainDelays.mtx.RUnlock()
		if ok {
			domainDelays.mtx.Lock()
			qLen := len(domainDelays.dTable[delays.domain])
			// если количество метрик больше 20к, удаляем более старые 10к
			if qLen >= 20000 {
				newDelays := domainDelays.dTable[delays.domain][qLen/2:]
				domainDelays.dTable[delays.domain] = newDelays
			}
			domainDelays.dTable[delays.domain] = append(domainDelays.dTable[delays.domain], delays.delay)
			domainDelays.mtx.Unlock()
		} else {
			domainDelays.mtx.Lock()
			domainDelays.dTable[delays.domain] = make([]float64, 1, 20000)
			domainDelays.dTable[delays.domain] = append(domainDelays.dTable[delays.domain], delays.delay)
			domainDelays.mtx.Unlock()
		}
	}
	logwriter.Info("Stopping stat.")
}
