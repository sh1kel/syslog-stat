package main

import (
	"fmt"
	"github.com/montanaflynn/stats"
	"net/http"
	"strings"
	"sync/atomic"
)

// урл со статистикой
// /stat
func webStat(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Queue length: %d\nProccessed emails: %d\n", atomic.LoadUint32(&gQueueLen), gMsgCounter)
}

// урл с доменом отдает среднюю задержку
// /delays/domain.tld
func (domainDelay *delaysMap) avgDelay(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:]
	domain := strings.Split(path, "/")
	domainDelay.mtx.Lock()
	_, ok := domainDelay.dTable[domain[1]]
	if ok {
		if len(domainDelay.dTable[domain[1]]) == 0 {
			fmt.Fprintf(w, "%.2f\n", 0.00)
		} else {
			a, _ := stats.Percentile(domainDelay.dTable[domain[1]], 90)
			fmt.Fprintf(w, "%.2f\n", a)
			// обнуляем метрики
			domainDelay.dTable[domain[1]] = []float64{}
		}
	}
	domainDelay.mtx.Unlock()
}

// урл отдает список доменов, для которых есть статистика
// /domains
func (domainDelay *delaysMap) listDomains(w http.ResponseWriter, r *http.Request) {
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
func (domainDelay *delaysMap) webDebug(w http.ResponseWriter, r *http.Request) {
	/*
		for key, val := range debugMap {
			fmt.Fprintf(w, "ID: %s\n", key)
			for _, line := range val.rawString {
				fmt.Fprintf(w, "%s\n", line)
			}
		}
	*/
	fmt.Fprintln(w, "dump")
}
