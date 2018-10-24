package mirror

import (
	"net"
	"../netflow/v9"
	"strings"
	"fmt"
	"encoding/binary"
	"bytes"
	"log"
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

var (
	netflowChannel = make(chan netflow9.Message, 1000)
)

const INPUT_ID = 10
const OUTPUT_ID = 14

type NFMirrorStatus struct {
	StartTime int64
	QeueSize int32
}

func NewNetflowv9Mirror(mirrorCfg string,logger *log.Logger) (*Netflowv9Mirror,error) {
	ume := new(Netflowv9Mirror)
	ume.Logger = logger
	ume.mirrorCfgFile = mirrorCfg
	b, err := ioutil.ReadFile(mirrorCfg)
	if err != nil {
		return nil,err
	}
	err = yaml.Unmarshal(b, &ume.mirrorConfigs)
	if err != nil {
		return ume,err
	}

	ume.initMap()
	ume.initUdpClients()
	return ume,nil
}

type Netflowv9Mirror struct {
	mirrorCfgFile string
	mirrorConfigs []Nfv9MirrorConfig
	mirrorMaps    map[string]Nfv9MirrorConfig
	udpClients    map[string]*UdpMirrorClient
	Logger        *log.Logger
}

type Nfv9MirrorConfig struct {
	Source string           `yaml:"source"`
	Rules  []nfv9MirrorRule `yaml:"rules"`
}
type nfv9MirrorRule struct {
	InPort      int32  `yaml:"inport"`
	OutPort     int32  `yaml:"outport"`
	DistAddress string `yaml:"distAddress"`
}

func (nfv9Mirror *Netflowv9Mirror) Status() *NFMirrorStatus{
	status := new(NFMirrorStatus)
	status.QeueSize = int32(len(netflowChannel))
	return status
}

func (nfv9Mirror *Netflowv9Mirror) ReceiveMessage(msg *netflow9.Message) {
	netflowChannel <- *msg
}

func (nfv9Mirror *Netflowv9Mirror) initMap() {
	nfv9Mirror.mirrorMaps = make(map[string]Nfv9MirrorConfig)
	for _, ec := range nfv9Mirror.mirrorConfigs {
		nfv9Mirror.mirrorMaps[ec.Source] = ec
	}
}
func (nfv9Mirror *Netflowv9Mirror) initUdpClients() {
	nfv9Mirror.udpClients = make(map[string]*UdpMirrorClient)
	for _, ec := range nfv9Mirror.mirrorConfigs {
		for _, ecr := range ec.Rules {
			addrPort := strings.Split(ecr.DistAddress, ":")
			if _, ok := nfv9Mirror.udpClients[ecr.DistAddress]; !ok {
				fmt.Printf("this is a source %s to distination %s ", ec.Source,ecr.DistAddress)
				nfv9Mirror.udpClients[ecr.DistAddress] = NewUdpMirrorClient(addrPort[0], addrPort[1])
			}
		}
	}
}

func (nfv9Mirror *Netflowv9Mirror) shutdown() {

}
func (nfv9Mirror *Netflowv9Mirror) GetConfig() ([]Nfv9MirrorConfig){
	return nfv9Mirror.mirrorConfigs
}

func (nfv9Mirror *Netflowv9Mirror) AddConfig(mirrorConfig Nfv9MirrorConfig) (int){
	if _,ok := nfv9Mirror.mirrorMaps[mirrorConfig.Source]; ok{
		return -1
	}

	nfv9Mirror.mirrorConfigs = append(nfv9Mirror.mirrorConfigs,mirrorConfig)
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

func (nfv9Mirror *Netflowv9Mirror) AddRule(sourceId string,rule nfv9MirrorRule) (int){
	if _, ok := nfv9Mirror.mirrorMaps[sourceId]; !ok {
		nfv9Mirror.Logger.Printf("can not find source of id %s\n",sourceId)
		return -1
	}
	rules := append(nfv9Mirror.mirrorMaps[sourceId].Rules, rule)
	nfv9Mirror.Logger.Printf("current rule size is %d\n", len(rules))

	//nfv9Mirror.mirrorMaps[sourceId].Rules = rules

	copy(nfv9Mirror.mirrorMaps[sourceId].Rules,rules)
	nfv9Mirror.Logger.Printf("current rule size is %d\n", len(nfv9Mirror.mirrorMaps[sourceId].Rules))

	nfv9Mirror.initMap()
	nfv9Mirror.saveConfigsTofile()
	return len(nfv9Mirror.mirrorMaps[sourceId].Rules)
}

func (nfv9Mirror *Netflowv9Mirror) DeleteRule(sourceId string,rule nfv9MirrorRule) (int){
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
		copy(nfv9Mirror.mirrorMaps[sourceId].Rules,append(nfv9Mirror.mirrorMaps[sourceId].Rules[:index],
			nfv9Mirror.mirrorMaps[sourceId].Rules[index+1:]...))
		nfv9Mirror.initMap()
	}
	nfv9Mirror.recycleClients()
	return index
}

func (nfv9Mirror *Netflowv9Mirror) DeleteConfig(sourceId string) (int) {
	var index= -1
	for i, e := range nfv9Mirror.mirrorConfigs {
		if e.Source == sourceId {
			index = i
			break
		}
	}
	nfv9Mirror.Logger.Printf("delete %s find index %d ",sourceId,index)
	if index != -1 {
		nfv9Mirror.mirrorConfigs = append(nfv9Mirror.mirrorConfigs[:index],
			nfv9Mirror.mirrorConfigs[index+1:]...)
		nfv9Mirror.initMap()
	}
	nfv9Mirror.recycleClients()
	nfv9Mirror.saveConfigsTofile()
	return index
}
func (nfv9Mirror *Netflowv9Mirror) saveConfigsTofile(){
	b,err := yaml.Marshal(nfv9Mirror.mirrorConfigs)
	//var str string = string(b)
	if err == nil{
		ioutil.WriteFile(nfv9Mirror.mirrorCfgFile,b,0x777)
	}
}

func (nfv9Mirror *Netflowv9Mirror) recycleClients(){
	//回收不用的client
	usedClient := make(map[string]string)
	for _,mirrorConfig := range nfv9Mirror.mirrorConfigs{
		for _, ecr := range mirrorConfig.Rules {
			//找到在用的
			if _, ok := nfv9Mirror.udpClients[ecr.DistAddress]; ok {
				usedClient[ecr.DistAddress] = ecr.DistAddress
			}
		}
	}

	for _,mirrorConfig := range nfv9Mirror.mirrorConfigs{
		for _, ecr := range mirrorConfig.Rules {
			//在用的不存在了
			if _, ok := usedClient[ecr.DistAddress]; !ok {
				nfv9Mirror.udpClients[ecr.DistAddress].conn.Close()
				delete(nfv9Mirror.udpClients,ecr.DistAddress)
			}
		}
	}

}




func (nfv9Mirror *Netflowv9Mirror) Run() {
	nfv9Mirror.Logger.Printf("start send packet client.")
	go func() {
		for {
			sMsg := <-netflowChannel
			nfv9Mirror.Logger.Printf("depeckage %s", sMsg.AgentID)
			ec := nfv9Mirror.mirrorMaps[sMsg.AgentID]

			for _, mRule := range ec.Rules {
				//sMsg.Msg.DataSets 很多记录[[]DecodedField,[]DecodedField,[]DecodedField] --> 转化为
				var datas [][]netflow9.DecodedField
				for _, nfData := range sMsg.DataSets { //[]DecodedField
					inputMatch, outputMatch := false, false
					inputFound, outputFound := false, false
					for _, decodedData := range nfData {
						id := decodedData.ID
						if id == INPUT_ID {
							inputFound = true
							if decodedData.Value == mRule.InPort || mRule.InPort == -1 {
								inputMatch = true
							}
						} else if id == OUTPUT_ID {
							outputFound = true
							if decodedData.Value == mRule.OutPort || mRule.OutPort == -1 {
								outputMatch = true
							}
						}
					}
					if !outputFound{
						outputMatch = true
					}
					if !inputFound{
						inputMatch = true
					}
					if inputMatch && outputMatch {// input and output matched
						datas = append(datas, nfData)
					}
				}
				if len(datas) > 0 {
					//生成header 生成bytes
					nfv9Mirror.udpClients[mRule.DistAddress].Send(toBytes(sMsg, 0, datas))
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

func toBytes(originalMsg netflow9.Message, seq uint32, fields [][]netflow9.DecodedField) []byte {
	buf := new(bytes.Buffer)

	var count uint16 = 0
	count = count + originalMsg.TemplaRecord.FieldCount
	count = count + uint16(len(originalMsg.SetHeaders))
	//orginal flow header
	binary.Write(buf, binary.BigEndian, originalMsg.Header.Version)
	binary.Write(buf, binary.BigEndian, uint16(count))
	binary.Write(buf, binary.BigEndian, originalMsg.Header.SysUpTime)
	binary.Write(buf, binary.BigEndian, originalMsg.Header.UNIXSecs)
	binary.Write(buf, binary.BigEndian, seq)
	binary.Write(buf, binary.BigEndian, originalMsg.Header.SrcID)

	//original flow templateRecord
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

	//flow的template 头部

	//flow的data
	for _, record := range fields {
		for _, item := range record {
			binary.Write(buf, binary.BigEndian, item.ID)
			binary.Write(buf, binary.BigEndian, item.Value)
		}
	}

	for i, setHeader := range originalMsg.SetHeaders {
		binary.Write(buf, binary.BigEndian, setHeader.FlowSetID)
		binary.Write(buf, binary.BigEndian, uint16(len(fields)))
		for _, field := range fields[i] {
			binary.Write(buf, binary.BigEndian, field.ID)
			binary.Write(buf, binary.BigEndian, field.Value)
		}
	}
	// 在写data记录

	return buf.Bytes()
}

func NewUdpMirrorClient(remoteAddress string, port string) *UdpMirrorClient {
	um := new(UdpMirrorClient)
	um.port = port
	um.remoteAddress = remoteAddress
	um.conn = nil
	return um
}

type UdpMirrorClient struct {
	remoteAddress string
	port          string
	conn          net.Conn
}

func (c *UdpMirrorClient) Send(b []byte) error {
	if c.conn == nil {
		c.openConn()
	}
	_, e := c.conn.Write(b)
	if e != nil {
		c.openConn()
	}
	return nil
}
func (c *UdpMirrorClient) openConn() error {
	conn, err := net.Dial("udp", c.remoteAddress+":"+c.port)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}