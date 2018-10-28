package mirror

import (
	"../netflow/v9"
	"strings"
	"encoding/binary"
	"bytes"
	"log"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"github.com/VerizonDigital/vflow/mirror"
	"net"
	"strconv"
	"fmt"
	"sync/atomic"
	"sync"
)

var (
	netflowChannel  = make(chan netflow9.Message, 1000)
	seqMap          = make(map[string]uint32)
	seqMutex       sync.Mutex
	cfgMutex       sync.Mutex
)

type Netflowv9Mirror struct {
	mirrorCfgFile string
	mirrorConfigs []Config
	mirrorMaps    map[string]Config
	rawSockets    map[string]Conn
	Logger        *log.Logger
	stats Netflowv9MirrorStatus
}

func (nfv9Mirror *Netflowv9Mirror) Status() *Netflowv9MirrorStatus {
	return &Netflowv9MirrorStatus{
		QueueSize:            len(netflowChannel),
		//MessageErrorCount:    atomic.LoadUint64(&nfv9Mirror.stats.MessageErrorCount),
		MessageReceivedCount: atomic.LoadUint64(&nfv9Mirror.stats.MessageReceivedCount),
		RawSentCount:         atomic.LoadUint64(&nfv9Mirror.stats.RawSentCount),
		RawErrorCount:        atomic.LoadUint64(&nfv9Mirror.stats.RawErrorCount),
	}
}

func (nfv9Mirror *Netflowv9Mirror) ReceiveMessage(msg *netflow9.Message) {
	netflowChannel <- *msg
}

func (nfv9Mirror *Netflowv9Mirror) initMap() {
	nfv9Mirror.mirrorMaps = make(map[string]Config)
	nfv9Mirror.rawSockets = make(map[string]Conn)
	for _, ec := range nfv9Mirror.mirrorConfigs {
		fmt.Printf("Router %10s add config rules count is %d\n",ec.Source, len(ec.Rules))
		for _,r := range ec.Rules {
			fmt.Printf("   rule: input port %6d, dst port %6d ->  %s \n",r.InPort,r.OutPort,r.DistAddress)
			remoteAddr := strings.Split(r.DistAddress,":")[0]
			if _, ok := nfv9Mirror.rawSockets[remoteAddr]; !ok {
				connect,err := NewRawConn(net.ParseIP(remoteAddr))
				if err != nil {
					nfv9Mirror.Logger.Printf("Mirror interface ip %s is wrong\n",remoteAddr)
					fmt.Printf("Mirror interface ip %s is wrong\n",remoteAddr)

				}else{
					nfv9Mirror.rawSockets[remoteAddr] = connect
				}
			}
		}
		nfv9Mirror.mirrorMaps[ec.Source] = ec
	}
}

func (nfv9Mirror *Netflowv9Mirror) shutdown() {

}
func (nfv9Mirror *Netflowv9Mirror) GetConfig() ([]Config) {
	return nfv9Mirror.mirrorConfigs
}

func (nfv9Mirror *Netflowv9Mirror) AddConfig(mirrorConfig Config) (int,string) {
	cfgMutex.Lock()
	nfv9Mirror.Logger.Printf("add config sourceId %s, configs %d",mirrorConfig.Source, len(mirrorConfig.Rules))
	if _, ok := nfv9Mirror.mirrorMaps[mirrorConfig.Source]; ok {
		return -1,"Source existed!"
	}
	nfv9Mirror.mirrorConfigs = append(nfv9Mirror.mirrorConfigs, mirrorConfig)
	nfv9Mirror.initMap()
	defer cfgMutex.Unlock()

	nfv9Mirror.saveConfigsTofile()
	return 0,"Add succeed!"
}

func (nfv9Mirror *Netflowv9Mirror) AddRule(agentIP string, rule Rule) (int,string) {
	cfgMutex.Lock()
	nfv9Mirror.Logger.Printf("add rule sourceId %s, rule dist %s.",agentIP, rule.DistAddress)

	if _, ok := nfv9Mirror.mirrorMaps[agentIP]; !ok {
		nfv9Mirror.Logger.Printf("can not find source of id %s.\n", agentIP)
		return -1,"no resource of "+agentIP
	}
	rules := append(nfv9Mirror.mirrorMaps[agentIP].Rules, rule)
	nfv9Mirror.Logger.Printf("current rule size is %d.\n", len(rules))
	mc := nfv9Mirror.mirrorMaps[agentIP]
	mc.Rules = rules

	nfv9Mirror.initMap()
	nfv9Mirror.Logger.Printf("current rule size is %d.\n", len(nfv9Mirror.mirrorMaps[agentIP].Rules))
	defer cfgMutex.Unlock()

	nfv9Mirror.saveConfigsTofile()
	return len(nfv9Mirror.mirrorMaps[agentIP].Rules),"add rule succeed."

}

func (nfv9Mirror *Netflowv9Mirror) DeleteRule(sourceId string, rule Rule) (int) {
	cfgMutex.Lock()
	if _, ok := nfv9Mirror.mirrorMaps[sourceId]; !ok {
		return -1
	}
	var index = -1
	for i, r := range nfv9Mirror.mirrorMaps[sourceId].Rules {
		if r.OutPort == rule.OutPort &&
			r.InPort == rule.InPort &&
			r.DistAddress == rule.DistAddress {
			index = i
		}
	}
	if index != -1 {
		copy(nfv9Mirror.mirrorMaps[sourceId].Rules, append(nfv9Mirror.mirrorMaps[sourceId].Rules[:index],
			nfv9Mirror.mirrorMaps[sourceId].Rules[index+1:]...))
		nfv9Mirror.initMap()
	}
	nfv9Mirror.recycleClients()
	defer cfgMutex.Unlock()

	nfv9Mirror.saveConfigsTofile()
	return index
}

func (nfv9Mirror *Netflowv9Mirror) DeleteConfig(agentIp string) (int) {
	cfgMutex.Lock()
	var index = -1
	for i, e := range nfv9Mirror.mirrorConfigs {
		if e.Source == agentIp {
			index = i
			break
		}
	}
	nfv9Mirror.Logger.Printf("delete %s find index %d ", agentIp, index)
	if index != -1 {
		nfv9Mirror.mirrorConfigs = append(nfv9Mirror.mirrorConfigs[:index],
			nfv9Mirror.mirrorConfigs[index+1:]...)
		nfv9Mirror.initMap()
	}
	nfv9Mirror.recycleClients()
	defer cfgMutex.Unlock()
	nfv9Mirror.saveConfigsTofile()
	return index
}
func (nfv9Mirror *Netflowv9Mirror) saveConfigsTofile() {
	b, err := yaml.Marshal(nfv9Mirror.mirrorConfigs)
	if err == nil {
		ioutil.WriteFile(nfv9Mirror.mirrorCfgFile, b, 0x777)
	}
}

func (nfv9Mirror *Netflowv9Mirror) recycleClients() {
	usedClient := make(map[string]string)
	for _, mirrorConfig := range nfv9Mirror.mirrorConfigs {
		for _, ecr := range mirrorConfig.Rules {
			//找到在用的
			if _, ok := nfv9Mirror.rawSockets[ecr.DistAddress]; ok {
				dstAddrs := strings.Split(ecr.DistAddress, ":")
				dstAddr := dstAddrs[0]
				usedClient[dstAddr] = ecr.DistAddress
			}
		}
	}

	for _, mirrorConfig := range nfv9Mirror.mirrorConfigs {
		for _, ecr := range mirrorConfig.Rules {
			//在用的不存在了
			dstAddrs := strings.Split(ecr.DistAddress, ":")
			dstAddr := dstAddrs[0]
			if _, ok := usedClient[dstAddr]; !ok {
				raw := nfv9Mirror.rawSockets[dstAddr]
				raw.Close()
				delete(nfv9Mirror.rawSockets, dstAddr)
			}
		}
	}
}

func (nfv9Mirror *Netflowv9Mirror) createRawPacket(srcAddress string, srcPort int,
	dstAddress string, dstPort int, data []byte) []byte {
	ipHLen := mirror.IPv4HLen
	udp := mirror.UDP{srcPort, dstPort, 0, 0}
	udpHdr := udp.Marshal()

	ip := mirror.NewIPv4HeaderTpl(mirror.UDPProto)
	ipHdr := ip.Marshal()
	payload := make([]byte, 1500)
	udp.SetLen(udpHdr, len(data))

	ip.SetAddrs(ipHdr, net.ParseIP(srcAddress), net.ParseIP(dstAddress))

	copy(payload[0:ipHLen], ipHdr)
	copy(payload[ipHLen:ipHLen+8], udpHdr)
	copy(payload[ipHLen+8:], data)
	return payload[:ipHLen+8+len(data)]
}

func (nfv9Mirror *Netflowv9Mirror) Run() {
	go func() {
		for {
			sMsg := <-netflowChannel
			atomic.AddUint64(&nfv9Mirror.stats.MessageReceivedCount, 1)
			cfgMutex.Lock()
			if _, ok := nfv9Mirror.mirrorMaps[sMsg.AgentID]; !ok {
				cfgMutex.Unlock()
				continue
			}


			ec := nfv9Mirror.mirrorMaps[sMsg.AgentID]
			var recordHeader netflow9.SetHeader
			recordHeader.FlowSetID = sMsg.SetHeader.FlowSetID
			recordHeader.Length = 0
			for _, mRule := range ec.Rules {
				//sMsg.Msg.DataSets 很多记录[[]DecodedField,[]DecodedField,[]DecodedField] --> 转化为
				var datas [][]netflow9.DecodedField
				for _, nfData := range sMsg.DataSets { //[]DecodedField
					inputMatch, outputMatch := false, false
					inputFound, outputFound := false, false
					var dataLen uint16 = 0
					for _, decodedData := range nfData {
						id := decodedData.ID
						dataLen = dataLen + uint16(binary.Size(decodedData.Value))
						if id == InputId {
							inputFound = true
							if binary.BigEndian.Uint16(decodedData.Value.([]byte)) == mRule.InPort || mRule.InPort == 255 {
								inputMatch = true
							}
						} else if id == OutputId {
							outputFound = true
							if binary.BigEndian.Uint16(decodedData.Value.([]byte)) == mRule.OutPort || mRule.OutPort == 255 {
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
					key := sMsg.AgentID+"_"+strconv.FormatUint(uint64(sMsg.Header.SrcID),10)
					// add a lock support
					seqMutex.Lock()
					if _, ok := seqMap[key]; ok {
						seq = seqMap[key]
					}else{
						seqMap[key] = 0
					}
					rBytes := nfv9Mirror.toBytes(sMsg, seq, recordHeader, datas)
					seqMap[key] = seqMap[key] + 1
					seqMutex.Unlock()

					dstAddrs := strings.Split(mRule.DistAddress, ":")
					dstAddr := dstAddrs[0]
					dstPort, _ := strconv.Atoi(dstAddrs[1])

					rBytes = nfv9Mirror.createRawPacket(sMsg.AgentID, 9999, dstAddr, dstPort, rBytes)
					raw := nfv9Mirror.rawSockets[dstAddr]
					err := raw.Send(rBytes)
					if err != nil {
						atomic.AddUint64(&nfv9Mirror.stats.RawErrorCount, 1)
						nfv9Mirror.Logger.Printf("raw socket send message error  bytes size %d, %s", len(rBytes),err)
					}else{
						atomic.AddUint64(&nfv9Mirror.stats.RawSentCount, 1)
					}
				}
			}//end rule fore
			cfgMutex.Unlock()
		}// end loop
	}()
}



//   The Packet Header format is specified as:
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |       Version Number          |            Count              |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                           sysUpTime                           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                           UNIX Secs                           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                       Sequence Number                         |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                        Source ID                              |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// template header 信息
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      FlowSet ID  = 0          |          Length               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// template 描述信息
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      Template ID 256          |         Field Count           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |        Field Type 1           |         Field Length 1        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |        Field Type 2           |         Field Length 2        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |             ...               |              ...              |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |        Field Type N           |         Field Length N        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// data 头部信息
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      FlowSet ID  = 256        |          Length               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// data具体信息
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |        Field Type             |         Field Length          |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

func (nfv9Mirror *Netflowv9Mirror) toBytes(originalMsg netflow9.Message, seq uint32,
	recordHeader netflow9.SetHeader, fields [][]netflow9.DecodedField) []byte {
	buf := new(bytes.Buffer)
	count := uint16(len(fields))
	count = count + uint16(len(originalMsg.TemplateRecords))

	//orginal flow header
	binary.Write(buf, binary.BigEndian, originalMsg.Header.Version)
	binary.Write(buf, binary.BigEndian, uint16(count))
	binary.Write(buf, binary.BigEndian, originalMsg.Header.SysUpTime)
	binary.Write(buf, binary.BigEndian, originalMsg.Header.UNIXSecs)
	binary.Write(buf, binary.BigEndian, seq)
	binary.Write(buf, binary.BigEndian, originalMsg.Header.SrcID)

	for _,template := range originalMsg.TemplateRecords {
		nfv9Mirror.writeTemplate(buf,template)
	}
	for _, field := range fields {
		for _, item := range field {
			binary.Write(buf, binary.BigEndian, item.Value)
		}
	}
	result := buf.Bytes()
	return result
}

func (nfv9Mirror *Netflowv9Mirror) writeTemplate(buf *bytes.Buffer,TemplaRecord netflow9.TemplateRecord){
	if TemplaRecord.FieldCount > 0 {
		binary.Write(buf, binary.BigEndian, 0)
		binary.Write(buf, binary.BigEndian, 4 + 4 + 4*TemplaRecord.FieldCount)
		nfv9Mirror.Logger.Printf("build a template templateId %d, fieldCount %d,header length is %d.",
			TemplaRecord.TemplateID,TemplaRecord.FieldCount,4 + 4 + 4*TemplaRecord.FieldCount)
		binary.Write(buf, binary.BigEndian, TemplaRecord.TemplateID)
		binary.Write(buf, binary.BigEndian,TemplaRecord.FieldCount)
		for _, spec := range TemplaRecord.FieldSpecifiers {
			binary.Write(buf, binary.BigEndian, spec.ElementID)
			binary.Write(buf, binary.BigEndian, spec.Length)
		}
		if TemplaRecord.ScopeFieldCount > 0 {
			for _, spec1 := range TemplaRecord.ScopeFieldSpecifiers {
				binary.Write(buf, binary.BigEndian, spec1.ElementID)
				binary.Write(buf, binary.BigEndian, spec1.Length)
			}
		}
	}
}