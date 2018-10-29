package mirror

import (
	"log"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"fmt"
	"os"
	"strings"
	"net"
	"sync"
	"../netflow/v9"
	"../ipfix"
	"encoding/binary"
)

var (
	Netflowv9MirrorInstance *Netflowv9Mirror
	IPFixMirrorInstance *IPFixMirror
	mirrorConfigs []Config
	logger *log.Logger
	mirrorMaps map[string]Config
	rawSockets map[string]Conn
	mirrorCfgFile string
	seqMutex       sync.Mutex
	cfgMutex       sync.Mutex

	seqMap          = make(map[string]uint32)

	ipfixChannel  = make(chan ipfix.Message, 1000)
	netflowChannel  = make(chan netflow9.Message, 1000)
)

const InputId = 10
const OutputId = 14

type FlowMirrorStatus struct {
	QueueSize            int
	MessageReceivedCount uint64
	RawSentCount         uint64
	RawErrorCount        uint64
}


type Config struct {
	Source string `yaml:"source"`
	Rules  []Rule `yaml:"rules"`
}

type Rule struct {
	InPort      int32 `yaml:"inport"`
	OutPort     int32 `yaml:"outport"`
	DistAddress string `yaml:"distAddress"`
}

func Init(mirrorCfg string,log *log.Logger) error{
	logger = log
	b, err := ioutil.ReadFile(mirrorCfg)
	if err != nil {
		logger.Printf("No Mirror config file is defined. \n")
		fmt.Printf("No Mirror config file is defined. \n")
		//os.Exit(-1)
		return  err
	}
	err = yaml.Unmarshal(b, &mirrorConfigs)
	if err != nil {
		logger.Printf("Mirror config file is worong, exit! \n")
		fmt.Printf("Mirror config file is worong,exit! \n")
		os.Exit(-1)
		return  err
	}
	mirrorCfgFile = mirrorCfg
	mirrorMaps = make(map[string]Config)
	rawSockets = make(map[string]Conn)
	buildMap()
	return nil
}

func parsePort(value interface{}) uint32{
	switch value.(type){
		case []byte:
			bytes := value.([]byte)
			if len(bytes) == 2 {
				return uint32(binary.BigEndian.Uint16(value.([]byte)))
			}else if len(bytes) == 4 {
				return uint32(binary.BigEndian.Uint32(value.([]byte)))
			}
		case uint32:
			return value.(uint32)
		case uint16:
			return uint32(value.(uint16))
		default:
			return 0
	}
	return 0
}


func  buildMap() {
	for _, ec := range mirrorConfigs {
		fmt.Printf("Router %10s add config rules count is %d\n",ec.Source, len(ec.Rules))
		for _,r := range ec.Rules {
			fmt.Printf("   rule: input port %6d, dst port %6d ->  %s \n",r.InPort,r.OutPort,r.DistAddress)
			remoteAddr := strings.Split(r.DistAddress,":")[0]
			if _, ok :=rawSockets[remoteAddr]; !ok {
				connect,err := NewRawConn(net.ParseIP(remoteAddr))
				if err != nil {
					logger.Printf("Mirror interface ip %s is wrong\n",remoteAddr)
					fmt.Printf("Mirror interface ip %s is wrong\n",remoteAddr)

				}else{
					rawSockets[remoteAddr] = connect
				}
			}
		}
		mirrorMaps[ec.Source] = ec
	}
}

func createRawPacket(srcAddress string, srcPort int,
	dstAddress string, dstPort int, data []byte) []byte {
	ipHLen := IPv4HLen
	udp := UDP{srcPort, dstPort, 0, 0}
	udpHdr := udp.Marshal()

	ip := NewIPv4HeaderTpl(UDPProto)
	ipHdr := ip.Marshal()
	payload := make([]byte, 1500)
	udp.SetLen(udpHdr, len(data))

	ip.SetAddrs(ipHdr, net.ParseIP(srcAddress), net.ParseIP(dstAddress))

	copy(payload[0:ipHLen], ipHdr)
	copy(payload[ipHLen:ipHLen+8], udpHdr)
	copy(payload[ipHLen+8:], data)
	return payload[:ipHLen+8+len(data)]
}


func NewNetFlowv9Mirror() (*Netflowv9Mirror, error) {
	mirrorInstance := new(Netflowv9Mirror)
	mirrorInstance.Logger = logger
	Netflowv9MirrorInstance = mirrorInstance
	return mirrorInstance, nil
}

func NewIPFixMirror() (*IPFixMirror, error) {
	mirrorInstance := new(IPFixMirror)
	mirrorInstance.Logger = logger
	IPFixMirrorInstance = mirrorInstance
	return mirrorInstance, nil
}



func GetConfig() ([]Config) {
	return mirrorConfigs
}

func AddConfig(mirrorConfig Config) (int,string) {
	cfgMutex.Lock()
	logger.Printf("add config sourceId %s, configs %d",mirrorConfig.Source, len(mirrorConfig.Rules))
	if _, ok := mirrorMaps[mirrorConfig.Source]; ok {
		return -1,"Source existed!"
	}
	mirrorConfigs = append(mirrorConfigs, mirrorConfig)
	buildMap()
	defer cfgMutex.Unlock()
	saveConfigsTofile()
	return 0,"Add succeed!"
}

func AddRule(agentIP string, rule Rule) (int,string) {
	cfgMutex.Lock()
	logger.Printf("add rule sourceId %s, rule dist %s.",agentIP, rule.DistAddress)

	if _, ok := mirrorMaps[agentIP]; !ok {
		logger.Printf("can not find source of id %s.\n", agentIP)
		return -1,"no resource of "+agentIP
	}

	for index,config := range mirrorConfigs {
		if config.Source == agentIP {
			mirrorConfigs[index].Rules = append(config.Rules, rule)
		}
	}

	buildMap()
	logger.Printf("current rule size is %d.\n", len(mirrorMaps[agentIP].Rules))
	defer cfgMutex.Unlock()

	saveConfigsTofile()
	return len(mirrorMaps[agentIP].Rules),"add rule succeed."
}

func DeleteRule(sourceId string, rule Rule) (int) {
	cfgMutex.Lock()
	if _, ok := mirrorMaps[sourceId]; !ok {
		return -1
	}
	var index = -1
	for i, r := range mirrorMaps[sourceId].Rules {
		if r.OutPort == rule.OutPort &&
			r.InPort == rule.InPort &&
			r.DistAddress == rule.DistAddress {
			index = i
		}
	}
	if index != -1 {
		copy(mirrorMaps[sourceId].Rules, append(mirrorMaps[sourceId].Rules[:index],
			mirrorMaps[sourceId].Rules[index+1:]...))
		buildMap()
	}
	recycleClients()
	defer cfgMutex.Unlock()

	saveConfigsTofile()
	return index
}

func DeleteConfig(agentIp string) (int) {
	cfgMutex.Lock()
	var index = -1
	for i, e := range mirrorConfigs {
		if e.Source == agentIp {
			index = i
			break
		}
	}
	logger.Printf("delete %s find index %d ", agentIp, index)
	if index != -1 {
		mirrorConfigs = append(mirrorConfigs[:index],
			mirrorConfigs[index+1:]...)
		buildMap()
	}
	recycleClients()
	defer cfgMutex.Unlock()
	saveConfigsTofile()
	return index
}
func saveConfigsTofile() {
	b, err := yaml.Marshal(mirrorConfigs)
	if err == nil {
		ioutil.WriteFile(mirrorCfgFile, b, 0x777)
	}
}

func  recycleClients() {
	usedClient := make(map[string]string)
	for _, mirrorConfig := range mirrorConfigs {
		for _, ecr := range mirrorConfig.Rules {
			//找到在用的
			if _, ok := rawSockets[ecr.DistAddress]; ok {
				dstAddrs := strings.Split(ecr.DistAddress, ":")
				dstAddr := dstAddrs[0]
				usedClient[dstAddr] = ecr.DistAddress
			}
		}
	}

	for _, mirrorConfig := range mirrorConfigs {
		for _, ecr := range mirrorConfig.Rules {
			//在用的不存在了
			dstAddrs := strings.Split(ecr.DistAddress, ":")
			dstAddr := dstAddrs[0]
			if _, ok := usedClient[dstAddr]; !ok {
				raw := rawSockets[dstAddr]
				raw.Close()
				delete(rawSockets, dstAddr)
			}
		}
	}
}
