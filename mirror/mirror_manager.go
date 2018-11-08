package mirror

import (
	"log"
	"io/ioutil"
	"fmt"
	"os"
	"strings"
	"net"
	"sync"
	"../netflow/v9"
	"../ipfix"
	"encoding/binary"
	"encoding/json"
)

var (
	Netflowv9MirrorInstance *Netflowv9Mirror
	IPFixMirrorInstance *IPFixMirror
	logger *log.Logger
	mirrorMaps map[string][]Rule
	rawSockets map[string]Conn
	policyConfigs []Policy
	mirrorCfgFile string
	seqMutex       sync.Mutex
	cfgMutex       sync.RWMutex
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

func Init(mirrorCfg string,log *log.Logger) error{
	logger = log

	err := LoadPolicy(mirrorCfg)
	if err != nil {
		logger.Printf("Mirror config file is worong, exit! \n")
		fmt.Printf("Mirror config file is worong,exit! \n")
		os.Exit(-1)
		return  err
	}
	mirrorCfgFile = mirrorCfg
	mirrorMaps = make(map[string][]Rule)
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
	mirrorMaps = make(map[string][]Rule)
	for _, policy := range policyConfigs {
		fmt.Printf("Policy %10s, rules count is %d\n", policy.PolicyId, len(policy.Rules))
		for _,r := range policy.Rules {
			if _, ok :=mirrorMaps[r.Source]; !ok {
				mirrorMaps[r.Source] = make([]Rule,0)
			}
			mirrorMaps[r.Source] = append(mirrorMaps[r.Source], r)
			fmt.Printf("   (source:%15s, inputPort %5d, outputPort %5d) ->  %s \n",r.Source,r.InPort,r.OutPort,r.DistAddress)
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
	}
}

func createRawPacket(srcAddress string, srcPort int,
	dstAddress string, dstPort int, data []byte) []byte{
	ipHLen := IPv4HLen
	udp := UDP{srcPort, dstPort, 0, 0}
	udpHdr := udp.Marshal()

	ip := NewIPv4HeaderTpl(UDPProto)
	ipHdr := ip.Marshal()
	payload := make([]byte, ipHLen+8+len(data))
	udp.SetLen(udpHdr, len(data))

	ip.SetAddrs(ipHdr, net.ParseIP(srcAddress), net.ParseIP(dstAddress))

	copy(payload[0:ipHLen], ipHdr)
	copy(payload[ipHLen:ipHLen+8], udpHdr)
	copy(payload[ipHLen+8:], data)
	return payload
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



func GetPolicies() ([]Policy) {
	return policyConfigs
}
func GetPolicyById(policyId string) (*Policy){
	for _,policy := range policyConfigs {
		if policy.PolicyId == policyId {
			return &policy
		}
	}
	return nil
}

func AddPolicy(policy Policy) (int,string) {
	logger.Printf("add config sourceId %s, configs %d",policy.PolicyId, len(policy.Rules))
	for _,config := range policyConfigs {
		if config.PolicyId == policy.PolicyId {
			return -1,"already have this policy "+policy.PolicyId
		}
	}
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	policyConfigs = append(policyConfigs, policy)
	buildMap()
	saveConfigsTofile()
	return len(policyConfigs),"add succeed."
}

func AddRule(policyId string, rule Rule) (int,string) {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	logger.Printf("add rule policyId %s, rule dist %s.",policyId, rule.DistAddress)
	curLen := 0
	for index,config := range policyConfigs {
		if config.PolicyId == policyId {
			for _,r := range config.Rules {
				if isSameRule(rule,r){
					return -1,"already has same rule."
				}
			}
			policyConfigs[index].Rules = append(config.Rules, rule)
			logger.Printf("current rule size is %d.\n", len(policyConfigs[index].Rules))
			curLen = len(policyConfigs[index].Rules)
			break
		}
	}
	if curLen == 0{
		return -1,"no policy id "+policyId
	}
	buildMap()
	saveConfigsTofile()
	return curLen,"add rule succeed."
}

func isSameRule(r1 Rule,r2 Rule) bool{
	if r1.Source == r2.Source &&
		r1.DistAddress == r2.DistAddress &&
		r1.InPort == r2.InPort &&
		r1.OutPort == r2.OutPort{
			return true
	}
	return false
}

func DeleteRule(policyId string, rule Rule) (int,string) {
	var pid = -1
	for i, e := range policyConfigs {
		if e.PolicyId == policyId {
			pid = i
			break
		}
	}
	if pid == -1 {
		return -1,"no policy "+policyId
	}
	var index = -1
	for i, r := range policyConfigs[pid].Rules {
		if r.OutPort == rule.OutPort &&
			r.InPort == rule.InPort &&
			r.DistAddress == rule.DistAddress {
			index = i
			break
		}
	}
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	if index != -1 {
		policyConfigs[pid].Rules = append(policyConfigs[pid].Rules[:index],
			policyConfigs[pid].Rules[index+1:]...)
		buildMap()
		recycleClients()
		saveConfigsTofile()
		return index,"rule is deleted"
	}else{
		return -1,"can not find matched rule for policy "+policyId
	}
}

func DeletePolicy(policyId string) (int,string) {
	var index = -1
	for i, e := range policyConfigs {
		if e.PolicyId == policyId {
			index = i
			break
		}
	}
	logger.Printf("delete %s find index %d ", policyId ,index)
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	if index != -1 {
		policyConfigs = append(policyConfigs[:index],
			policyConfigs[index+1:]...)
		buildMap()
		recycleClients()
		saveConfigsTofile()
		return index,"delete success"
	} else {
		return index,"can not find policy"
	}
}
func saveConfigsTofile() {
	b, err := json.Marshal(policyConfigs)
	if err == nil {
		ioutil.WriteFile(mirrorCfgFile, b, 0x777)
	}
}

func recycleClients() {
	usedClient := make(map[string]string)
	for _, policy := range policyConfigs {
		for _, ecr := range policy.Rules {
			//找到在用的
			dstAddresses := strings.Split(ecr.DistAddress, ":")
			dstAddr := dstAddresses[0]
			if _, ok := rawSockets[dstAddr]; ok {
				usedClient[dstAddr] = ecr.DistAddress
			}
		}
	}

	for _, mirrorConfig := range policyConfigs {
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
