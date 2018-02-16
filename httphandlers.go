package main

import (
	"fmt"
	"github.com/montanaflynn/stats"
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

		a, _ := stats.Percentile(domainDelay.dTable[domain[1]], 90)
		fmt.Fprintf(w, "%.2f\n", a)
		// обнуляем метрики
		domainDelay.dTable[domain[1]] = []float64{}
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
