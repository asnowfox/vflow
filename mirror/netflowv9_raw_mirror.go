package mirror

import (
	"../netflow/v9"
	"strings"
	"encoding/binary"
	"log"
	"strconv"
	"sync/atomic"
)



type Netflowv9Mirror struct {
	Logger *log.Logger
	stats  FlowMirrorStatus
}


func (t *Netflowv9Mirror) ReceiveMessage(msg *netflow9.Message) {
	netflowChannel <- *msg
}


func (t *Netflowv9Mirror) Status() *FlowMirrorStatus {
	return &FlowMirrorStatus{
		QueueSize:            len(netflowChannel),
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
			cfgMutex.Lock()
			if _, ok := mirrorMaps[sMsg.AgentID]; !ok {
				cfgMutex.Unlock()
				continue
			}

			ec := mirrorMaps[sMsg.AgentID]

			for _, mRule := range ec.Rules {
				//sMsg.Msg.DataSets 很多记录[[]DecodedField,[]DecodedField,[]DecodedField] --> 转化为
				var datas [][]netflow9.DecodedField
				var headers []netflow9.SetHeader
				for i, nfData := range sMsg.DataSets { //[]DecodedField
					inputMatch, outputMatch := false, false
					inputFound, outputFound := false, false
					var dataLen uint16 = 0
					for _, decodedData := range nfData {
						id := decodedData.ID
						dataLen = dataLen + uint16(binary.Size(decodedData.Value))
						if id == InputId {
							inputFound = true
							port := parsePort(decodedData.Value)
							if port == uint32(mRule.InPort) || mRule.InPort == -1 {
								inputMatch = true
							}
						} else if id == OutputId {
							outputFound = true
							port := parsePort(decodedData.Value)
							if port == uint32(mRule.OutPort) ||  mRule.OutPort == -1 {
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
						var recordHeader netflow9.SetHeader
						recordHeader.FlowSetID = sMsg.SetHeader[i].FlowSetID
						recordHeader.Length = 4 // self length
						recordHeader.Length += dataLen
						headers = append(headers, recordHeader)
					}
				}
				if len(datas) > 0  {
					//recordHeader.Length += 4
					if len(datas) != len(headers) {
						t.Logger.Printf("Encode %s netflow message error!",sMsg.AgentID)
					}
					var seq uint32 = 0
					key := sMsg.AgentID+"_"+strconv.FormatUint(uint64(sMsg.Header.SrcID),10)
					// add a lock support
					seqMutex.Lock()
					if _, ok := seqMap[key]; ok {
						seq = seqMap[key]
					}else{
						seqMap[key] = 0
					}
					rBytes := netflow9.Encode(sMsg, seq, headers, datas)
					seqMap[key] = seqMap[key] + 1
					seqMutex.Unlock()

					dstAddrs := strings.Split(mRule.DistAddress, ":")
					dstAddr := dstAddrs[0]
					dstPort, _ := strconv.Atoi(dstAddrs[1])

					rBytes = createRawPacket(sMsg.AgentID, 9999, dstAddr, dstPort, rBytes)
					raw := rawSockets[dstAddr]
					err := raw.Send(rBytes)
					if err != nil {
						atomic.AddUint64(&t.stats.RawErrorCount, 1)
						t.Logger.Printf("raw socket send message error  bytes size %d, %s", len(rBytes),err)
					}else{
						atomic.AddUint64(&t.stats.RawSentCount, 1)
					}
				}
			}//end rule fore
			cfgMutex.Unlock()
		}// end loop
	}()
}

