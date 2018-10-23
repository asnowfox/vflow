package mirror

import (
	"net"
	"../netflow/v9"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"strings"
	"fmt"
	"encoding/binary"
	"bytes"
	"log"
)

type SourcedNetflowMessage struct {
	SourceAddr string
	Msg        netflow9.Message
}

var (
	netflowChannel = make(chan SourcedNetflowMessage, 1000)
)

const INPUT_ID = 10
const OUTPUT_ID = 14

type UdpMirrorExchanger struct {
	mirrorConfigs []ExchangerConfig
	mirrorMaps    map[string]ExchangerConfig
	udpCliets     map[string]*UdpMirror
	Logger *log.Logger
}

type ExchangerConfig struct {
	Source string         `yaml:"source"`
	Rules  []ExchangeRule `yaml:"rules"`
}
type ExchangeRule struct {
	InPort      int32  `yaml:"inport"`
	OutPort     int32  `yaml:"outport"`
	DistAddress string `yaml:"distAddress"`
}

func (ume *UdpMirrorExchanger) Status() *MirrorStatus{
	status := new(MirrorStatus)
	status.QeueSize = int32(len(netflowChannel))
	return status
}

func (ume *UdpMirrorExchanger) ExchangeMessage(sourceAddr string, msg netflow9.Message) {
	netflowChannel <- SourcedNetflowMessage{msg.AgentID, msg}
}

func (ume *UdpMirrorExchanger) LoadCfgAndRun(mirrorCfg string,logger *log.Logger) error {
	b, err := ioutil.ReadFile(mirrorCfg)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(b, &ume.mirrorConfigs)
	if err != nil {
		return err
	}
	ume.Logger = logger
	ume.initMap()
	ume.initUdpClients()
	ume.run()
	return nil
}

func (ume *UdpMirrorExchanger) initMap() {
	ume.mirrorMaps = make(map[string]ExchangerConfig)
	for _, ec := range ume.mirrorConfigs {
		ume.mirrorMaps[ec.Source] = ec
	}
}
func (ume *UdpMirrorExchanger) initUdpClients() {
	//	logger.Println("this is a source i will init udp Client")
	ume.udpCliets = make(map[string]*UdpMirror)
	for _, ec := range ume.mirrorConfigs {
		for _, ecr := range ec.Rules {
			addrPort := strings.Split(ecr.DistAddress, ":")
			if _, ok := ume.udpCliets[ec.Source]; !ok {
				fmt.Printf("this is a source %s", ec.Source)
				ume.udpCliets[ec.Source] = NewMirror(addrPort[0], addrPort[1])
			}
		}
	}
}

func (ume *UdpMirrorExchanger) run() {
	ume.Logger.Printf("start send packet client.")
	go func() {
		for {
			sMsg := <-netflowChannel
			ume.Logger.Printf("depeckage %s", sMsg.SourceAddr)
			ec := ume.mirrorMaps[sMsg.SourceAddr]

			for _, exchageRule := range ec.Rules {
				//sMsg.Msg.DataSets 很多记录[[]DecodedField,[]DecodedField,[]DecodedField] --> 转化为
				var datas [][]netflow9.DecodedField
				for _, nfData := range sMsg.Msg.DataSets { //[]DecodedField
					inputMatch, outputMatch := false, false

					for _, decodedData := range nfData {
						id := decodedData.ID
						if id == INPUT_ID {
							if decodedData.Value == exchageRule.InPort || exchageRule.InPort == -1 {
								inputMatch = true
							}
						} else if id == OUTPUT_ID {
							if decodedData.Value == exchageRule.OutPort || exchageRule.OutPort == -1 {
								outputMatch = true
							}
						}
					}
					//出入都匹配
					if inputMatch && outputMatch {
						datas = append(datas, nfData)
					}
				}
				if len(datas) > 0 {
					//生成header 生成bytes
					ume.udpCliets[exchageRule.DistAddress].Send(toBytes(sMsg.Msg, 0, datas))
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
func NewMirror(remoteAddress string, port string) *UdpMirror {
	um := new(UdpMirror)
	um.port = port
	um.remoteAddress = remoteAddress
	um.conn = nil
	return um
}

type UdpMirror struct {
	remoteAddress string
	port          string
	conn          net.Conn
}

func (c *UdpMirror) Send(b []byte) error {
	if c.conn == nil {
		c.openConn()
	}
	_, e := c.conn.Write(b)
	if e != nil {
		c.openConn()
	}
	return nil
}
func (c *UdpMirror) openConn() error {
	conn, err := net.Dial("udp", c.remoteAddress+":"+c.port)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}
