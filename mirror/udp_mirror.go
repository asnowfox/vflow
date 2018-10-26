package mirror

import (
	"../netflow/v9"
	"strings"
	"fmt"
	"encoding/binary"
	"bytes"
	"log"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"github.com/VerizonDigital/vflow/mirror"
	"net"
	"strconv"
)

var (
	netflowChannel    = make(chan netflow9.Message, 1000)
	//= NewRawConn(net.ParseIP("127.0.0.1"))
)

type Netflowv9Mirror struct {
	mirrorCfgFile string
	mirrorConfigs []Config
	mirrorMaps    map[string]Config
	udpClients    map[string]*UdpClient
	Logger        *log.Logger
	rawSocket Conn
}


func (nfv9Mirror *Netflowv9Mirror) Status() *Status {
	status := new(Status)
	status.QeueSize = int32(len(netflowChannel))
	return status
}

func (nfv9Mirror *Netflowv9Mirror) ReceiveMessage(msg *netflow9.Message) {
	netflowChannel <- *msg
}

func (nfv9Mirror *Netflowv9Mirror) initMap() {
	nfv9Mirror.mirrorMaps = make(map[string]Config)
	for _, ec := range nfv9Mirror.mirrorConfigs {
		nfv9Mirror.mirrorMaps[ec.Source] = ec
	}
}
func (nfv9Mirror *Netflowv9Mirror) initUdpClients() {
	nfv9Mirror.udpClients = make(map[string]*UdpClient)
	for _, ec := range nfv9Mirror.mirrorConfigs {
		for _, ecr := range ec.Rules {
			addrPort := strings.Split(ecr.DistAddress, ":")
			if _, ok := nfv9Mirror.udpClients[ecr.DistAddress]; !ok {
				fmt.Printf("this is a source %s to distination %s ", ec.Source, ecr.DistAddress)
				nfv9Mirror.udpClients[ecr.DistAddress] = NewUdpMirrorClient(addrPort[0], addrPort[1])

				nfv9Mirror.udpClients[ecr.DistAddress] = NewUdpMirrorClient(addrPort[0], addrPort[1])
			}
		}
	}
}

func (nfv9Mirror *Netflowv9Mirror) shutdown() {
	for _,e := range nfv9Mirror.udpClients {
		e.Close()
	}
}
func (nfv9Mirror *Netflowv9Mirror) GetConfig() ([]Config) {
	return nfv9Mirror.mirrorConfigs
}

func (nfv9Mirror *Netflowv9Mirror) AddConfig(mirrorConfig Config) (int) {
	if _, ok := nfv9Mirror.mirrorMaps[mirrorConfig.Source]; ok {
		return -1
	}

	nfv9Mirror.mirrorConfigs = append(nfv9Mirror.mirrorConfigs, mirrorConfig)
	nfv9Mirror.initMap()
	//添加dst 的client
	for _, ecr := range mirrorConfig.Rules {
		addrPort := strings.Split(ecr.DistAddress, ":")
		if _, ok := nfv9Mirror.udpClients[ecr.DistAddress]; !ok {
			fmt.Printf("this is a source %s", mirrorConfig.Source)
			nfv9Mirror.udpClients[ecr.DistAddress] = NewUdpMirrorClient(addrPort[0], addrPort[1])
		}
	}
	nfv9Mirror.saveConfigsTofile()
	return 0
}

func (nfv9Mirror *Netflowv9Mirror) AddRule(sourceId string, rule Rule) (int) {
	if _, ok := nfv9Mirror.mirrorMaps[sourceId]; !ok {
		nfv9Mirror.Logger.Printf("can not find source of id %s\n", sourceId)
		return -1
	}
	rules := append(nfv9Mirror.mirrorMaps[sourceId].Rules, rule)
	nfv9Mirror.Logger.Printf("current rule size is %d\n", len(rules))
	mc	:=  nfv9Mirror.mirrorMaps[sourceId]
	mc.Rules = rules

	nfv9Mirror.Logger.Printf("current rule size is %d\n", len(nfv9Mirror.mirrorMaps[sourceId].Rules))

	nfv9Mirror.initMap()
	nfv9Mirror.saveConfigsTofile()
	return len(nfv9Mirror.mirrorMaps[sourceId].Rules)
}

func (nfv9Mirror *Netflowv9Mirror) DeleteRule(sourceId string, rule Rule) (int) {
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
	return index
}

func (nfv9Mirror *Netflowv9Mirror) DeleteConfig(sourceId string) (int) {
	var index = -1
	for i, e := range nfv9Mirror.mirrorConfigs {
		if e.Source == sourceId {
			index = i
			break
		}
	}
	nfv9Mirror.Logger.Printf("delete %s find index %d ", sourceId, index)
	if index != -1 {
		nfv9Mirror.mirrorConfigs = append(nfv9Mirror.mirrorConfigs[:index],
			nfv9Mirror.mirrorConfigs[index+1:]...)
		nfv9Mirror.initMap()
	}
	nfv9Mirror.recycleClients()
	nfv9Mirror.saveConfigsTofile()
	return index
}
func (nfv9Mirror *Netflowv9Mirror) saveConfigsTofile() {
	b, err := yaml.Marshal(nfv9Mirror.mirrorConfigs)
	//var str string = string(b)
	if err == nil {
		ioutil.WriteFile(nfv9Mirror.mirrorCfgFile, b, 0x777)
	}
}

func (nfv9Mirror *Netflowv9Mirror) recycleClients() {
	//回收不用的client
	usedClient := make(map[string]string)
	for _, mirrorConfig := range nfv9Mirror.mirrorConfigs {
		for _, ecr := range mirrorConfig.Rules {
			//找到在用的
			if _, ok := nfv9Mirror.udpClients[ecr.DistAddress]; ok {
				usedClient[ecr.DistAddress] = ecr.DistAddress
			}
		}
	}

	for _, mirrorConfig := range nfv9Mirror.mirrorConfigs {
		for _, ecr := range mirrorConfig.Rules {
			//在用的不存在了
			if _, ok := usedClient[ecr.DistAddress]; !ok {
				(*nfv9Mirror.udpClients[ecr.DistAddress].conn).Close()
				delete(nfv9Mirror.udpClients, ecr.DistAddress)
			}
		}
	}

}

func (nfv9Mirror *Netflowv9Mirror) genRawPacket(srcAddress string,srcPort int,
	dstAddress string ,dstPort int,data []byte) []byte{

	ipHLen := mirror.IPv4HLen
	udp := mirror.UDP{srcPort, dstPort, 0, 0}
	udpHdr := udp.Marshal()

	ip := mirror.NewIPv4HeaderTpl(mirror.UDPProto)
	//ip.Length = int16(ipHLen+8+len(data))
	ipHdr := ip.Marshal()

	payload := make([]byte, 1500)

	udp.SetLen(udpHdr, len(data))

	//(b []byte, src, dst net.IP)
	ip.SetAddrs(ipHdr, net.ParseIP(srcAddress), net.ParseIP(dstAddress))

	copy(payload[0:ipHLen], ipHdr)
	copy(payload[ipHLen:ipHLen+8], udpHdr)
	copy(payload[ipHLen+8:], data)

	return payload[:ipHLen+8+len(data)]

}

func (nfv9Mirror *Netflowv9Mirror) Run() {
	nfv9Mirror.Logger.Printf("start send packet client.")
	go func() {
		for {
			sMsg := <-netflowChannel
			ec := nfv9Mirror.mirrorMaps[sMsg.AgentID]

			for _, mRule := range ec.Rules {
				//sMsg.Msg.DataSets 很多记录[[]DecodedField,[]DecodedField,[]DecodedField] --> 转化为
				var datas [][]netflow9.DecodedField
				var recordHeader netflow9.SetHeader
				recordHeader.FlowSetID = sMsg.SetHeader.FlowSetID
				recordHeader.Length = 4
				for _, nfData := range sMsg.DataSets { //[]DecodedField
					inputMatch, outputMatch := false, false
					inputFound, outputFound := false, false
					dataLen := 0
					for _, decodedData := range nfData {
						id := decodedData.ID
						dataLen += binary.Size(decodedData.Value)
						if id == InputId {
							inputFound = true
							if decodedData.Value == mRule.InPort || mRule.InPort == -1 {
								inputMatch = true
							}
						} else if id == OutputId {
							outputFound = true
							if decodedData.Value == mRule.OutPort || mRule.OutPort == -1 {
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
					}
				}

				if len(datas) > 0 || sMsg.TemplaRecord.FieldCount > 0{
					//生成header 生成bytes
					if sMsg.TemplaRecord.FieldCount > 0 {
						nfv9Mirror.Logger.Printf("temlate field count > 0")
					}
					bytes := nfv9Mirror.toBytes(sMsg, mRule.Req,recordHeader,datas)
					nfv9Mirror.Logger.Printf("send netflow v9 data to: %s, size: %d bytes",mRule.DistAddress, len(bytes))

					//genRawPacket(srcAddress string,srcPort int,
					//	dstAddress string ,dstPort int,data []byte) []byte
					dsts := strings.Split(mRule.DistAddress,":")
					dstport,_ :=strconv.Atoi(dsts[1])
					nfv9Mirror.genRawPacket(sMsg.AgentID, 9999,dsts[0],dstport,bytes)
					//nfv9Mirror.udpClients[mRule.DistAddress].Send(bytes)
					mRule.Req = mRule.Req+1
				}
			}
		}
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
	var count uint16 = 0

	count = count + uint16(len(fields))

	//orginal flow header
	binary.Write(buf, binary.BigEndian, originalMsg.Header.Version)
	binary.Write(buf, binary.BigEndian, uint16(count))
	binary.Write(buf, binary.BigEndian, originalMsg.Header.SysUpTime)
	binary.Write(buf, binary.BigEndian, originalMsg.Header.UNIXSecs)
	binary.Write(buf, binary.BigEndian, originalMsg.Header.SeqNum)
	binary.Write(buf, binary.BigEndian, originalMsg.Header.SrcID)

	if originalMsg.TemplaRecord.FieldCount > 0 {
		binary.Write(buf, binary.BigEndian, originalMsg.TemplaRecord.TemplateID)
		binary.Write(buf, binary.BigEndian, originalMsg.TemplaRecord.FieldCount)
		for _, spec := range originalMsg.TemplaRecord.FieldSpecifiers {
			binary.Write(buf, binary.BigEndian, spec.ElementID)
			binary.Write(buf, binary.BigEndian, spec.Length)
		}
		if originalMsg.TemplaRecord.ScopeFieldCount > 0 {
			for _, spec1 := range originalMsg.TemplaRecord.ScopeFieldSpecifiers {
				binary.Write(buf, binary.BigEndian, spec1.ElementID)
				binary.Write(buf, binary.BigEndian, spec1.Length)
			}
		}
	}

	nfv9Mirror.Logger.Printf("buffer before length is %d.",buf.Len())

	binary.Write(buf,binary.BigEndian,recordHeader.FlowSetID)
	binary.Write(buf,binary.BigEndian,recordHeader.Length)
	for _,field := range fields {
		for _, item := range field {
			binary.Write(buf, binary.BigEndian, item.Value)
		}
	}
	result := buf.Bytes()
	nfv9Mirror.Logger.Printf("buffer bytes is %d.",len(result))
	return result
}



