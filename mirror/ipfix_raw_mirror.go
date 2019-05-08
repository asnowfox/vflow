package mirror

import (
	"encoding/binary"
	"github.com/VerizonDigital/vflow/ipfix"
	"github.com/VerizonDigital/vflow/vlogger"
	"strconv"
	"strings"
	"sync/atomic"
)

type IPFixMirror struct {
	stats FlowMirrorStatus
}

func (t *IPFixMirror) Status() *FlowMirrorStatus {
	return &FlowMirrorStatus{
		//QueueSize:            len(netflowChannel),
		MessageReceivedCount: atomic.LoadUint64(&t.stats.MessageReceivedCount),
		RawSentCount:         atomic.LoadUint64(&t.stats.RawSentCount),
		RawErrorCount:        atomic.LoadUint64(&t.stats.RawErrorCount),
	}
}

func (t *IPFixMirror) ReceiveMessage(msg *ipfix.Message) {
	ipfixChannel <- *msg
}

func (t *IPFixMirror) shutdown() {

}

func (t *IPFixMirror) Run() {
	go func() {
		for {
			sMsg := <-ipfixChannel
			atomic.AddUint64(&t.stats.MessageReceivedCount, 1)
			//cfgMutex.Lock()
			if _, ok := mirrorMaps[sMsg.AgentID]; !ok {
			//	cfgMutex.Unlock()
				vlogger.Logger.Printf("Can not find agent cache, %s. ",sMsg.AgentID)
				continue
			}
			ec := mirrorMaps[sMsg.AgentID]
			for _, mRule := range ec{
				var msgFlowSets []ipfix.DataFlowSet
				for _,flowSet := range sMsg.DataFlowSets {
					flowDataSet := t.filterFlowDataSet(mRule,flowSet)
					//该flowSet中有存在的记录
					if len(flowDataSet.DataSets) > 0  {
						msgFlowSets = append(msgFlowSets, flowDataSet)
					}
				}
				//no data and no template records continue
				if len(msgFlowSets) == 0 && len(sMsg.TemplateRecords) == 0{
					continue
				}
				var seq uint32 = 0
				key := sMsg.AgentID+"_"+strconv.FormatUint(uint64(sMsg.Header.DomainID),10)
				// add a lock support
				seqMutex.Lock()
				if a, ok := seqMap[key]; ok {
					seq = a
				}else{
					seqMap[key] = 0
				}
				seqMap[key] = seqMap[key] + 1
				seqMutex.Unlock()

				rBytes := ipfix.Encode(sMsg, seq, msgFlowSets)

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
			}//end rule for
			//cfgMutex.Unlock()
		}// end loop
	}()
}

func (t *IPFixMirror) filterFlowDataSet(mRule Rule,flowSet ipfix.DataFlowSet)ipfix.DataFlowSet{
	rtnFlowSet := new(ipfix.DataFlowSet)
	rtnFlowSet.SetHeader.SetID = flowSet.SetHeader.SetID
	var datas [][]ipfix.DecodedField
	// 从data里面进行匹配，过滤出这个flowSet中满足条件的的flowData,放入 datas数据结构
	for _, nfData := range flowSet.DataSets { //[]DecodedField
		inputMatch, outputMatch := false, false
		inputFound, outputFound := false, false
		var dataLen uint16 = 0
		for _, decodedData := range nfData {
			id := decodedData.ID
			dataLen = dataLen + uint16(binary.Size(decodedData.Value))
			if id == InputId {
				inputFound = true
				port := parsePort(decodedData.Value)
				if port == uint32(mRule.Port) {
					inputMatch = true
				}
			} else if id == OutputId {
				outputFound = true
				port := parsePort(decodedData.Value)
				if port == uint32(mRule.Port)  {
					outputMatch = true
				}
			}
		}
		if !outputFound {
			outputMatch = true
		}
		if !inputFound {
			inputMatch = true
		}
		if inputMatch && outputMatch { // input and output matched
			datas = append(datas, nfData)
			rtnFlowSet.SetHeader.Length+= dataLen
			rtnFlowSet.DataSets = datas
		}
	}
	if rtnFlowSet.SetHeader.Length > 0 {
		rtnFlowSet.SetHeader.Length+=4
	}
	return *rtnFlowSet
}

