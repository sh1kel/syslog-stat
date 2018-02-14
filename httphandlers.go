package main

import (
	"fmt"
	"net/http"
	"strings"
)

// Сервер со статистикоц
func (msgList *messageList) webStat(w http.ResponseWriter, r *http.Request) {
	msgList.mtx.RLock()
	fmt.Fprintf(w, "Queue length: %d\nProccessed messages: %d\n", len(msgList.Messages), msgList.msgProccesed)
	msgList.mtx.RUnlock()
}

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
		domainDelay.dTable[domain[1]] = []float32{}
	}

	domainDelay.mtx.Unlock()
}

func (domainDelay *delays) listDomains(w http.ResponseWriter, r *http.Request) {
	domainDelay.mtx.RLock()
	for key, _ := range domainDelay.dTable {
		if key != "" {
			fmt.Fprintln(w, key)
		}
	}
	domainDelay.mtx.RUnlock()
}
