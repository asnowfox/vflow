package vflow

import (
	"fmt"
	"sync"
	"strings"
)

type PacketStatistics struct {
	IdCurrentSeqMap  map[string]uint32
	IdCurrentLostMap map[string]uint32
	lock  sync.Mutex
}

func NewPacketStatistics() (*PacketStatistics) {
	return &PacketStatistics{
		IdCurrentSeqMap : make(map[string]uint32),
		IdCurrentLostMap : make(map[string]uint32),
	}
}

func (i *PacketStatistics) getLost(agentId string) uint32 {
	var cnt uint32 = 0
	for key,value := range i.IdCurrentLostMap {
		if strings.HasPrefix(key,agentId+"_") {
			cnt += value
		}
	}
	return cnt
}

func (i *PacketStatistics) recordSeq(agentId string, source uint32, seq uint32) {
	i.lock.Lock()
	defer i.lock.Unlock()
	key := agentId + "_" + fmt.Sprint(source)
	if _, ok := i.IdCurrentLostMap[key]; !ok {
		i.IdCurrentLostMap[key] = 0
		i.IdCurrentSeqMap[key] = seq
		return
	}
	i.IdCurrentLostMap[key] = i.IdCurrentLostMap[key] + (seq - i.IdCurrentSeqMap[key] - 1)
	i.IdCurrentSeqMap[key] = seq
}

