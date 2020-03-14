package mirror

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/VerizonDigital/vflow/ipfix"
	. "github.com/VerizonDigital/vflow/utils"
	"github.com/VerizonDigital/vflow/vlogger"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
)

var (
	Netflowv9MirrorInstance *Netflowv9Mirror
	IPFixMirrorInstance     *IPFixMirror
	mirrorMaps              map[string][]Rule
	rawSockets              map[string]Conn
	policyConfigs           []Policy
	mirrorCfgFile           string
	seqMutex                sync.Mutex
	cfgMutex                sync.RWMutex
	seqMap                  = make(map[string]uint32)
	ipfixChannel            = make(chan ipfix.Message, 1000)
)

const InputId = 10
const OutputId = 14

type FlowMirrorStatus struct {
	QueueSize            int
	MessageReceivedCount uint64
	RawSentCount         uint64
	RawErrorCount        uint64
}

func Init(mirrorCfg string) error {
	vlogger.Logger.Println("Load flow forward file " + mirrorCfg)
	err := LoadPolicy(mirrorCfg)
	if err != nil {
		vlogger.Logger.Printf("Mirror config file is wrong, exit! \n")
		fmt.Printf("Mirror config file is wrong,exit! \n")
		os.Exit(-1)
		return err
	}
	mirrorCfgFile = mirrorCfg
	mirrorMaps = make(map[string][]Rule)
	rawSockets = make(map[string]Conn)
	buildMap()
	return nil
}

func parsePort(value interface{}) uint32 {
	switch value.(type) {
	case []byte:
		bytes := value.([]byte)
		if len(bytes) == 2 {
			return uint32(binary.BigEndian.Uint16(value.([]byte)))
		} else if len(bytes) == 4 {
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

func buildMap() {
	mirrorMaps = make(map[string][]Rule)
	for _, policy := range policyConfigs {
		//if policy.Enable == 0 {
		//	continue
		//}
		targetAddress := policy.TargetAddress
		for i := 0; i < len(policy.Rules); i++ {
			policy.Rules[i].DistAddress = targetAddress
		}
	}
	for _, policy := range policyConfigs {
		vlogger.Logger.Printf("Policy %10s, enable %10d, target is %10s,rules count is %d\n",
			policy.PolicyId, policy.Enable, policy.TargetAddress, len(policy.Rules))
		if policy.Enable == 0 {
			continue
		}
		for _, r := range policy.Rules {
			if _, ok := mirrorMaps[r.Source]; !ok {
				mirrorMaps[r.Source] = make([]Rule, 0)
			}
			mirrorMaps[r.Source] = append(mirrorMaps[r.Source], r)
			vlogger.Logger.Printf("   (Source:%15s, Port %5d, Direction %5d) ->  %s \n", r.Source, r.Port, r.Direction, r.DistAddress)

			for _, rule := range r.DistAddress {
				remoteAddr := strings.Split(rule, ":")[0]
				if _, ok := rawSockets[remoteAddr]; !ok {
					connect, err := NewRawConn(net.ParseIP(remoteAddr))
					if err != nil {
						vlogger.Logger.Printf("Mirror interface ip %s is wrong\n", remoteAddr)
					} else {
						rawSockets[remoteAddr] = connect
					}
				}
			}
		}
	}
}

func createRawPacket(srcAddress string, srcPort int,
	dstAddress string, dstPort int, data []byte) []byte {
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

func NewNetflowv9Mirror() (*Netflowv9Mirror, error) {
	mirrorInstance := new(Netflowv9Mirror)
	Netflowv9MirrorInstance = mirrorInstance
	return mirrorInstance, nil
}

func NewIPFixMirror() (*IPFixMirror, error) {
	mirrorInstance := new(IPFixMirror)
	IPFixMirrorInstance = mirrorInstance
	return mirrorInstance, nil
}

func GetPolicies() []Policy {
	return policyConfigs
}
func GetPolicyById(policyId string) *Policy {
	for _, policy := range policyConfigs {
		if policy.PolicyId == policyId {
			return &policy
		}
	}
	return nil
}

func UpdatePolicy(policyId string, nPolicy Policy) (int, string) {
	index := 0
	found := false
	for _, config := range policyConfigs {
		if config.PolicyId == policyId {
			found = true
			break
		}
		index++
	}
	if found {
		policyConfigs[index].PolicyId = nPolicy.PolicyId
		policyConfigs[index].TargetAddress = nPolicy.TargetAddress
		policyConfigs[index].Enable = nPolicy.Enable
		//for _,rule := range policyConfigs[index].Rules {
		//	rule.DistAddress = nPolicy.TargetAddress
		//	vlogger.Logger.Printf("update rule %s target queue is %s", rule.Source, rule.DistAddress)
		//}
		buildMap()
		saveConfigsTofile()
		recycleClients()
		return 1, "update success"
	} else {
		return -1, "can not find policy " + policyId
	}
}

func AddPolicy(policy Policy) (int, string) {
	vlogger.Logger.Printf("add config sourceId %s, target is %s, configs %d",
		policy.PolicyId, policy.TargetAddress, len(policy.Rules))
	for _, target := range policy.TargetAddress {
		result, e := HostAddrCheck(target)
		if !result {
			return -1, e.Error()
		}
		if e != nil {
			return -1, e.Error()
		}
	}

	if policy.PolicyId == "" {
		return -1, "Policy id is blank"
	}
	if policy.Rules == nil {
		policy.Rules = make([]Rule, 0)
	}
	for _, config := range policyConfigs {
		if config.PolicyId == policy.PolicyId {
			return -1, "already have this policy " + policy.PolicyId
		}
	}

	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	policyConfigs = append(policyConfigs, policy)
	buildMap()
	saveConfigsTofile()
	return len(policyConfigs), "add succeed."
}

func AddRule(policyId string, rule Rule) (int, string) {
	result, e := RuleCheck(rule)
	if !result {
		return -1, e.Error()
	}
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	vlogger.Logger.Printf("add rule policyId %s, rule dist %s.", policyId, rule.DistAddress)
	curLen := 0
	for index, policy := range policyConfigs {
		if policy.PolicyId == policyId {
			for _, r := range policy.Rules {
				if isSameRule(rule, r) {
					return -1, "already has same rule."
				}
			}
			rule.DistAddress = policy.TargetAddress
			policyConfigs[index].Rules = append(policy.Rules, rule)
			vlogger.Logger.Printf("current rule size is %d.\n", len(policyConfigs[index].Rules))
			curLen = len(policyConfigs[index].Rules)
			break
		}
	}
	if curLen == 0 {
		return -1, "no policy id " + policyId
	}
	buildMap()
	saveConfigsTofile()
	return curLen, "add rule succeed."
}

func isSameRule(r1 Rule, r2 Rule) bool {
	if r1.Source == r2.Source &&
		r1.Port == r2.Port &&
		r1.Direction == r2.Direction {
		return true
	}
	return false
}

func DeleteRule(policyId string, rule Rule) (int, string) {
	var pid = -1
	for i, e := range policyConfigs {
		if e.PolicyId == policyId {
			pid = i
			break
		}
	}
	if pid == -1 {
		return -1, "no policy " + policyId
	}
	var index = -1
	for i, r := range policyConfigs[pid].Rules {
		if r.Port == rule.Port &&
			r.Direction == rule.Direction {
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
		saveConfigsTofile()
		recycleClients()
		return len(policyConfigs[pid].Rules), "delete success."
	} else {
		return -1, "can not find matched rule for policy " + policyId
	}
}

func DeletePolicy(policyId string) (int, string) {
	var index = -1
	for i, e := range policyConfigs {
		if e.PolicyId == policyId {
			index = i
			break
		}
	}
	vlogger.Logger.Printf("delete %s find index %d ", policyId, index)
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	if index != -1 {
		policyConfigs = append(policyConfigs[:index],
			policyConfigs[index+1:]...)
		buildMap()
		recycleClients()
		saveConfigsTofile()
		return index, "delete success"
	} else {
		return index, "can not find policy"
	}
}
func saveConfigsTofile() {
	b, err := json.MarshalIndent(policyConfigs, "", "    ")

	if err == nil {
		_ = ioutil.WriteFile(mirrorCfgFile, b, 0x777)
	}
}

func recycleClients() {
	go func() {
		usedClient := make(map[string]string)
		for _, policy := range policyConfigs {
			vlogger.Logger.Printf("check rule for policy %s, rules length is %d.\r\n", policy.PolicyId, len(policy.Rules))
			fmt.Printf("check rule for policy %s, rules length is %d.\r\n", policy.PolicyId, len(policy.Rules))
			for _, ecr := range policy.Rules {
				//找到在用的
				for _, dist := range ecr.DistAddress {
					dstAddresses := strings.Split(dist, ":")
					dstAddr := dstAddresses[0]
					if _, ok := rawSockets[dstAddr]; ok {
						vlogger.Logger.Printf("used address add %s .\r\n", dstAddr)
						fmt.Printf("used address add %s .\r\n", dstAddr)
						usedClient[dstAddr] = dist
					}
				}
			}
		}

		for _, mirrorConfig := range policyConfigs {
			for _, ecr := range mirrorConfig.Rules {
				//在用的不存在了
				for _, dist := range ecr.DistAddress {
					dstAddrs := strings.Split(dist, ":")
					dstAddr := dstAddrs[0]
					if _, ok := usedClient[dstAddr]; !ok {
						vlogger.Logger.Printf("recycle dstAddress %s .\r\n", dstAddr)
						fmt.Printf("recycle dstAddress %s .\r\n", dstAddr)
						raw := rawSockets[dstAddr]
						_ = raw.Close()
						delete(rawSockets, dstAddr)
					}
				}
			}
		}
	}()
}
