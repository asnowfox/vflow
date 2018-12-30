package mirror

import (
	"../netflow/v9"
	"sync/atomic"
	"strconv"
	"strings"
	"../vlogger"
	"fmt"
)

type Netflowv9Mirror struct {
	stats FlowMirrorStatus
}

func (t *Netflowv9Mirror) ReceiveMessage(msg netflow9.Message) {
	sMsg := msg
	atomic.AddUint64(&t.stats.MessageReceivedCount, 1)
	cfgMutex.RLock()
	if _, ok := mirrorMaps[sMsg.AgentID]; !ok {
		cfgMutex.RUnlock()
		return
	}

	ec := mirrorMaps[sMsg.AgentID]

	for _, mRule := range ec {
		var msgFlowSets = make([]netflow9.DataFlowSet, 0)
		if sMsg.DataFlowSets != nil {
			for _, flowSet := range sMsg.DataFlowSets {
				//TODO 这里可以缓存区查找该flowSet对应的Rule
				// agentId_inport_outport -> distAddress:port
				flowDataSet := t.filterFlowDataSet(sMsg, mRule, flowSet)
				//该flowSet中有存在的记录
				if len(flowDataSet.DataFlowRecords) > 0 {
					msgFlowSets = append(msgFlowSets, flowDataSet)
				}
			}
		}

		//no data and no template records continue
		if len(msgFlowSets) == 0 && len(sMsg.TemplateRecords) == 0 {
			continue
		}

		//buf := new(bytes.Buffer)
		//for _,e := range msgFlowSets {
		//	b, err := sMsg.JSONMarshal(buf, e.DataFlowRecords)
		//	if err == nil {
		//		if strings.Contains(string(b),"{\"i\":8,\"v\":\"0.0.0"){
		//			vlogger.Logger.Printf("msg is %s, length is %d.",string(b),len(sMsg.DataFlowSets))
		//		}
		//
		//	}
		//}

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
		rBytes := netflow9.Encode(sMsg.AgentID,sMsg, seq, msgFlowSets)

		for _, r := range mRule.DistAddress {
			dstAddrs := strings.Split(r, ":")
			dstAddr := dstAddrs[0]
			dstPort, _ := strconv.Atoi(dstAddrs[1])
			rBytes = createRawPacket(sMsg.AgentID, 9999, dstAddr, dstPort, rBytes)
			if raw, ok := rawSockets[dstAddr]; ok {
				err := raw.Send(rBytes)

				if len(sMsg.TemplateRecords) > 0{
					fmt.Printf("I will send template record to %s:%d\r\n",dstAddr,dstPort)
				}

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
}

func (t *Netflowv9Mirror) Status() *FlowMirrorStatus {
	return &FlowMirrorStatus{
		//QueueSize: len(netflowChannel),
		//MessageErrorCount:    atomic.LoadUint64(&t.stats.MessageErrorCount),
		MessageReceivedCount: atomic.LoadUint64(&t.stats.MessageReceivedCount),
		RawSentCount:         atomic.LoadUint64(&t.stats.RawSentCount),
		RawErrorCount:        atomic.LoadUint64(&t.stats.RawErrorCount),
	}
}

func (t *Netflowv9Mirror) shutdown() {

}

func (t *Netflowv9Mirror) filterFlowDataSet(msg netflow9.Message, mRule Rule, flowSet netflow9.DataFlowSet) netflow9.DataFlowSet {
	rtnFlowSet := new(netflow9.DataFlowSet)
	rtnFlowSet.SetHeader.FlowSetID = flowSet.SetHeader.FlowSetID
	var datas []netflow9.DataFlowRecord
	// 从data里面进行匹配，过滤出这个flowSet中满足条件的的flowData,放入 datas数据结构
	for _, nfData := range flowSet.DataFlowRecords { //[]DecodedField
		inputMatch, outputMatch := false, false

		if nfData.InPort == -1 || nfData.OutPort == -1 {
			inputMatch, outputMatch = true, true
		}
		if nfData.InPort == int(mRule.Port) {
			inputMatch = true
		}
		if nfData.OutPort == int(mRule.Port) {
			outputMatch = true
		}
		/*
		0x00: ingress flow
		0x01: egress flow
		*/
		if mRule.Direction == -1 { //双向
			if inputMatch || outputMatch { // input and output matched
				datas = append(datas, nfData)
				rtnFlowSet.SetHeader.Length += nfData.Length
				rtnFlowSet.DataFlowRecords = datas
			}
		} else if mRule.Direction == 0 { //入方向
			if inputMatch  { // input and output matched
				datas = append(datas, nfData)
				rtnFlowSet.SetHeader.Length += nfData.Length
				rtnFlowSet.DataFlowRecords = datas
			}
		} else if mRule.Direction == 1 { //出方向
			if outputMatch  { // input and output matched
				datas = append(datas, nfData)
				rtnFlowSet.SetHeader.Length += nfData.Length
				rtnFlowSet.DataFlowRecords = datas
			}
		}
	}
	if rtnFlowSet.SetHeader.Length > 0 {
		rtnFlowSet.SetHeader.Length += 4
	}
	return *rtnFlowSet
}
