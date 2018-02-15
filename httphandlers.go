package main

import (
	"fmt"
	"net/http"
	"strings"
)

// урл со статистикой
// /stat
func (msgList *messageList) webStat(w http.ResponseWriter, r *http.Request) {
	msgList.mtx.RLock()
	fmt.Fprintf(w, "Queue length: %d\nProccessed emails: %d\n", len(msgList.Messages), msgList.msgProccesed)
	msgList.mtx.RUnlock()
}

// урл с доменом отдает среднюю задержку
// /delays/domain.tld
func (domainDelay *delays) avgDelay(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:]
	domain := strings.Split(path, "/")
	domainDelay.mtx.Lock()
	_, ok := domainDelay.dTable[domain[1]]
	if ok {
		var total float32 = 0
		var d float32
		for _, d = range domainDelay.dTable[domain[1]] {
			total += d
		}
		fmt.Fprintf(w, "%.2f\n", total/float32(len(domain[1])))
		// обнуляем метрики
		domainDelay.dTable[domain[1]] = []float32{}
	}

	domainDelay.mtx.Unlock()
}

// урл отдает список доменов, для которых есть статистика
// /domains
func (domainDelay *delays) listDomains(w http.ResponseWriter, r *http.Request) {
	domainDelay.mtx.RLock()
	for key, _ := range domainDelay.dTable {
		if key != "" {
			fmt.Fprintln(w, key)
		}
	}
	domainDelay.mtx.RUnlock()
}

// урл с дебагом
// /debug
func (msgList *messageList) webDebug(w http.ResponseWriter, r *http.Request) {
	msgList.mtx.RLock()
	for key, val := range msgList.Messages {
		for _, line := range val.rawString {
			fmt.Fprintf(w, "%s: %v\n", key, line)
		}
	}
	msgList.mtx.RUnlock()
}
