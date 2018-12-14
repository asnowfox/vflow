package mirror

import (
	"../netflow/v9"
	"sync/atomic"
	"strconv"
	"strings"
	"../vlogger"
	"bytes"
)

type Netflowv9Mirror struct {
	stats FlowMirrorStatus
}

func (t *Netflowv9Mirror) ReceiveMessage(msg *netflow9.Message) {
	netflowChannel <- *msg
}

func (t *Netflowv9Mirror) Status() *FlowMirrorStatus {
	return &FlowMirrorStatus{
		QueueSize: len(netflowChannel),
		//MessageErrorCount:    atomic.LoadUint64(&t.stats.MessageErrorCount),
		MessageReceivedCount: atomic.LoadUint64(&t.stats.MessageReceivedCount),
		RawSentCount:         atomic.LoadUint64(&t.stats.RawSentCount),
		RawErrorCount:        atomic.LoadUint64(&t.stats.RawErrorCount),
	}
}

func (t *Netflowv9Mirror) shutdown() {

}

func (t *Netflowv9Mirror) Run() {
	go func() {
		for {
			sMsg := <-netflowChannel
			atomic.AddUint64(&t.stats.MessageReceivedCount, 1)
			cfgMutex.RLock()
			if _, ok := mirrorMaps[sMsg.AgentID]; !ok {
				cfgMutex.RUnlock()
				continue
			}
			ec := mirrorMaps[sMsg.AgentID]
			for _, mRule := range ec {
				var msgFlowSets = make([]netflow9.DataFlowSet,0)
				for _, flowSet := range sMsg.DataFlowSets {
					//TODO 这里可以缓存区查找该flowSet对应的Rule
					// agentId_inport_outport -> distAddress:port
					flowDataSet := t.filterFlowDataSet(mRule, flowSet)
					//该flowSet中有存在的记录
					if len(flowDataSet.DataFlowRecords) > 0 {
						msgFlowSets = append(msgFlowSets, flowDataSet)
					}
				}
				//no data and no template records continue
				if len(msgFlowSets) == 0 && len(sMsg.TemplateRecords) == 0 {
					continue
				}
				//这个是针对这个rule进行发送的过程

				var seq uint32 = 0
				key := sMsg.AgentID + "_" + strconv.FormatUint(uint64(sMsg.Header.SrcID), 10)
				// add a lock support
				seqMutex.Lock()
				if a, ok := seqMap[key]; ok {
					seq = a
				} else {
					seqMap[key] = 0
				}
				seqMap[key] = seqMap[key] + 1
				seqMutex.Unlock()
				for _,e := range msgFlowSets {
					buf := new(bytes.Buffer)
					b, _ := sMsg.JSONMarshal(buf, e.DataFlowRecords)
					vlogger.Logger.Printf("send msg %s\r\n.",string(b))
				}


				rBytes := netflow9.Encode(sMsg, seq, msgFlowSets)

				for _,r := range mRule.DistAddress {
					dstAddrs := strings.Split(r, ":")
					dstAddr := dstAddrs[0]
					dstPort, _ := strconv.Atoi(dstAddrs[1])
					rBytes = createRawPacket(sMsg.AgentID, 9999, dstAddr, dstPort, rBytes)
					if raw, ok := rawSockets[dstAddr]; ok {
						err := raw.Send(rBytes)
						if err != nil {
							atomic.AddUint64(&t.stats.RawErrorCount, 1)
							vlogger.Logger.Printf("raw socket send message error  bytes size %d, %s", len(rBytes), err)
						} else {
							atomic.AddUint64(&t.stats.RawSentCount, 1)
						}
					} else {
						vlogger.Logger.Printf("can not find raw socket for dist %s", dstAddr)
					}
				}

			} //end rule for
			cfgMutex.RUnlock()
		} // end loop
	}()
}

func (t *Netflowv9Mirror) filterFlowDataSet(mRule Rule, flowSet netflow9.DataFlowSet) netflow9.DataFlowSet {
	rtnFlowSet := new(netflow9.DataFlowSet)
	rtnFlowSet.SetHeader.FlowSetID = flowSet.SetHeader.FlowSetID
	var datas []netflow9.DataFlowRecord
	// 从data里面进行匹配，过滤出这个flowSet中满足条件的的flowData,放入 datas数据结构
	for _, nfData := range flowSet.DataFlowRecords { //[]DecodedField
		inputMatch, outputMatch := false, false
		if nfData.InPort == -1 || nfData.OutPort == -1 {
			inputMatch, outputMatch = true, true
		}
		if nfData.InPort == int(mRule.InPort) || mRule.InPort == -1 {
			inputMatch = true
		}
		if nfData.OutPort == int(mRule.OutPort) || mRule.OutPort == -1 {
			outputMatch = true
		}

		if inputMatch && outputMatch { // input and output matched
			datas = append(datas, nfData)
			//for _, data := range nfData.DataSets {
			rtnFlowSet.SetHeader.Length += nfData.Length
			//}
			//rtnFlowSet.SetHeader.Length += dataLen
			rtnFlowSet.DataFlowRecords = datas
		}
	}
	if rtnFlowSet.SetHeader.Length > 0 {
		rtnFlowSet.SetHeader.Length += 4
	}
	return *rtnFlowSet
}
