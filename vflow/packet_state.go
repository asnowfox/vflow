package vflow

import (
	"fmt"
	"sync"
	"strings"
	"errors"
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

func (i *PacketStatistics) getLost(agentId string) (uint32,error) {
	var cnt uint32 = 0
	found := false
	for key,value := range i.IdCurrentLostMap {
		if strings.HasPrefix(key,agentId+"_") {
			cnt += value
			found = true
		}
	}
	if !found{
		return 0,errors.New("can not find agent")
	}
	return cnt,nil
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

