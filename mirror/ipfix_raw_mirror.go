package mirror

import (
	"log"
	"sync/atomic"
	"encoding/binary"
	"strconv"
	"strings"
	"../ipfix"
)

type IPFixMirror struct {
	Logger        *log.Logger
	stats FlowMirrorStatus
}

func (t *IPFixMirror) Status() *FlowMirrorStatus {
	return &FlowMirrorStatus{
		QueueSize:            len(netflowChannel),
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
			cfgMutex.Lock()
			if _, ok := mirrorMaps[sMsg.AgentID]; !ok {
				cfgMutex.Unlock()
				continue
			}

			ec := mirrorMaps[sMsg.AgentID]
			var recordHeader ipfix.SetHeader
			recordHeader.SetID = sMsg.SetHeader.SetID
			recordHeader.Length = 0
			for _, mRule := range ec.Rules {
				//sMsg.Msg.DataSets 很多记录[[]DecodedField,[]DecodedField,[]DecodedField] --> 转化为
				var datas [][]ipfix.DecodedField
				for _, nfData := range sMsg.DataSets { //[]DecodedField
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
							if port == uint32(mRule.OutPort) || mRule.OutPort == -1 {
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
						recordHeader.Length += dataLen
					}
				}
				if len(datas) > 0  {
					recordHeader.Length += 4

					var seq uint32 = 0
					key := sMsg.AgentID+"_"+strconv.FormatUint(uint64(sMsg.Header.DomainID),10)
					// add a lock support
					seqMutex.Lock()
					if _, ok := seqMap[key]; ok {
						seq = seqMap[key]
					}else{
						seqMap[key] = 0
					}
					rBytes := ipfix.Encode(sMsg, seq,  datas)
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


