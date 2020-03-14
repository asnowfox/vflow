package producer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/VerizonDigital/vflow/utils"
	"github.com/deckarep/golang-set"
	"strconv"

	"github.com/VerizonDigital/vflow/vlogger"
	"io/ioutil"
	"net"
	"os"
	"sync"
)

var (
	mirrorMaps         map[string][]QueueRule
	queuePolicyConfigs []QueuePolicy
	cfgMutex           sync.RWMutex
	queueTopics        map[string]string
)

type QueuePolicy struct {
	PolicyId     string      `json:"policyId"`
	TargetQueues []string    `json:"targetQueue"`
	Enable       int         `json:"enable"`
	Rules        []QueueRule `json:"rules"`
}

type QueueRule struct {
	Source       string   `json:"source"`
	Port         int32    `json:"port"`
	Direction    int      `json:"direction"`
	TargetQueues []string `json:"targetQueue"`
}

var (
	cachedQueueNames = make(map[string][]string)
)

func ParseTopic(agentId string, inPort int32, outPort int32, direction int) []string {
	//加锁，防止修改policy的时候并发修改的问题
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	key := agentId + "_" + strconv.Itoa(int(inPort)) + "_" + strconv.Itoa(int(outPort)) + "_" + strconv.Itoa(direction)
	topics := cachedQueueNames[key]

	if topics == nil {
		topics = innerParse(agentId, inPort, outPort, direction)
		if topics != nil {
			cachedQueueNames[key] = topics
		}
	}
	return topics

}

func innerParse(agentId string, inport int32, outport int32, direction int) []string {
	queueNames := mapset.NewSet()
	for _, rule := range mirrorMaps {
		for _, r := range rule {
			if r.isParsed(agentId, inport, outport, direction) {
				for _, e := range r.TargetQueues {
					queueNames.Add(e)
				}
			}
		}
	}
	rtn := make([]string, 0)
	for _, e := range queueNames.ToSlice() {
		rtn = append(rtn, e.(string))
	}
	return rtn
}

func (r *QueueRule) isParsed(agentId string, inPort int32, outPort int32, direction int) bool {
	if r.Source == agentId {
		if r.Direction == 0 { //入方向
			if r.Port == inPort || r.Port == -1 {
				return true
			} else {
				return false
			}
		} else if r.Direction == 1 { //出方向
			if r.Port == outPort || r.Port == -1 {
				return true
			} else {
				return false
			}
		} else if r.Direction == -1 { //双向
			if r.Port == inPort && direction == 0 { //入方向
				return true
			} else if r.Port == outPort && direction == 1 {
				return true
			} else if r.Port == -1 {
				return true
			}
			return false
		} else {
			return false
		}
	} else {
		return false
	}
}

func Init(queueForwardCfg string) error {
	err := loadPolicy(queueForwardCfg)
	if err != nil {
		vlogger.Logger.Printf("Mirror config file is wrong, exit! \n")
		fmt.Printf("Mirror config file is wrong,exit! \n")
		os.Exit(-1)
		return err
	}
	mirrorMaps = make(map[string][]QueueRule)
	queueTopics = make(map[string]string)
	buildMap()
	return nil
}

func loadPolicy(queueForwardCfg string) error {
	b, err := ioutil.ReadFile(queueForwardCfg)
	if err != nil {
		vlogger.Logger.Printf("No Mirror config file is defined. \n")
		fmt.Printf("No Mirror config file is defined. \n")
		return err
	}
	err = json.Unmarshal(b, &queuePolicyConfigs)
	if err != nil {
		vlogger.Logger.Printf("Mirror config file is worong, exit! \n")
		fmt.Printf("Mirror config file is worong,exit! \n")
		os.Exit(-1)
		return err
	}
	return nil
}

func GetPolicyById(policyId string) *QueuePolicy {
	for _, policy := range queuePolicyConfigs {
		if policy.PolicyId == policyId {
			return &policy
		}
	}
	return nil
}

func GetPolicies() []QueuePolicy {
	return queuePolicyConfigs
}
func DeleteQueuePolicy(policyId string) (int, string) {
	var index = -1
	for i, e := range queuePolicyConfigs {
		if e.PolicyId == policyId {
			index = i
			break
		}
	}
	vlogger.Logger.Printf("delete %s find index %d ", policyId, index)
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	if index != -1 {
		queuePolicyConfigs = append(queuePolicyConfigs[:index],
			queuePolicyConfigs[index+1:]...)
		buildMap()
		recycleClients()
		saveConfigsTofile()
		cachedQueueNames = make(map[string][]string)
		return index, "delete success"
	} else {
		return index, "can not find policy"
	}
}

func AddQueuePolicy(policy QueuePolicy) (int, string) {
	vlogger.Logger.Printf("add config sourceId %s, target is %s, configs %d",
		policy.PolicyId, policy.TargetQueues, len(policy.Rules))

	for _, target := range policy.TargetQueues {
		result, e := utils.MQNameCheck(target)
		if !result || e != nil {
			return -1, e.Error()
		}
	}
	if policy.PolicyId == "" {
		return -1, "Policy id is blank"
	}
	if policy.Rules == nil {
		policy.Rules = make([]QueueRule, 0)
	}
	for _, config := range queuePolicyConfigs {
		if config.PolicyId == policy.PolicyId {
			return -1, "already have this policy " + policy.PolicyId
		}
	}

	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	queuePolicyConfigs = append(queuePolicyConfigs, policy)
	buildMap()
	saveConfigsTofile()
	return len(queuePolicyConfigs), "add succeed."
}

func buildMap() {
	mirrorMaps = make(map[string][]QueueRule)
	for _, policy := range queuePolicyConfigs {
		//if policy.Enable == 0 {
		//	continue
		//}
		targetAddress := policy.TargetQueues
		for i := 0; i < len(policy.Rules); i++ {
			policy.Rules[i].TargetQueues = targetAddress
		}
	}
	for _, policy := range queuePolicyConfigs {
		vlogger.Logger.Printf("Policy %20s, enable %5d, target is %10s,rules count is %d\n",
			policy.PolicyId, policy.Enable, policy.TargetQueues, len(policy.Rules))
		if policy.Enable == 0 {
			continue
		}
		for _, r := range policy.Rules {
			if _, ok := mirrorMaps[r.Source]; !ok {
				mirrorMaps[r.Source] = make([]QueueRule, 0)
			}
			mirrorMaps[r.Source] = append(mirrorMaps[r.Source], r)
			vlogger.Logger.Printf("   (Source:%15s, Port %5d, Direction %5d) ->  %s \n", r.Source, r.Port, r.Direction, r.TargetQueues)

			for _, queueName := range r.TargetQueues {
				topic := queueName
				if _, ok := queueTopics[topic]; !ok {
					queueTopics[topic] = topic
				}
			}
		}
	}
}

func UpdateQueuePolicy(policyId string, nPolicy QueuePolicy) (int, string) {
	index := 0
	found := false
	for _, config := range queuePolicyConfigs {
		if config.PolicyId == policyId {
			found = true
			break
		}
		index++
	}
	if found {
		cfgMutex.Lock()
		defer cfgMutex.Unlock()
		cachedQueueNames = make(map[string][]string)
		queuePolicyConfigs[index].PolicyId = nPolicy.PolicyId
		queuePolicyConfigs[index].TargetQueues = nPolicy.TargetQueues
		queuePolicyConfigs[index].Enable = nPolicy.Enable
		//for _, r := range queuePolicyConfigs[index].Rules {
		//	r.TargetQueues = nPolicy.TargetQueues
		//	vlogger.Logger.Printf("update rule %s target queue is %s", r.Source, r.TargetQueues)
		//}
		buildMap()
		saveConfigsTofile()
		recycleClients()
		return 1, "update success"
	} else {
		return -1, "can not find policy " + policyId
	}
}

func AddQueueRule(policyId string, rule QueueRule) (int, string) {
	result, e := RuleCheck(rule)
	if !result {
		return -1, e.Error()
	}
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	vlogger.Logger.Printf("add rule policyId %s, rule dist %s.", policyId, rule.TargetQueues)
	curLen := 0
	for index, policy := range queuePolicyConfigs {
		if policy.PolicyId == policyId {
			for _, r := range policy.Rules {
				if isSameRule(rule, r) {
					return -1, "already has same rule."
				}
			}
			rule.TargetQueues = policy.TargetQueues
			queuePolicyConfigs[index].Rules = append(policy.Rules, rule)
			vlogger.Logger.Printf("current rule size is %d.\n", len(queuePolicyConfigs[index].Rules))
			curLen = len(queuePolicyConfigs[index].Rules)
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

func isSameRule(r1 QueueRule, r2 QueueRule) bool {
	if r1.Source == r2.Source &&
		r1.Port == r2.Port &&
		r1.Direction == r2.Direction {
		return true
	}
	return false
}

func RuleCheck(rule QueueRule) (bool, error) {
	a := net.ParseIP(rule.Source)
	if a == nil {
		return false, errors.New("can not parse ip " + rule.Source)
	}
	if rule.Port < int32(-1) || rule.Port > int32(65535) {
		return false, errors.New("port is illegal, too big or to small")
	}

	if rule.Direction != 0 && rule.Direction != 1 && rule.Direction != -1 {
		return false, errors.New("direction is error must be 0 1 or -l")
	}

	return true, nil
}

func DeleteQueueRule(policyId string, rule QueueRule) (int, string) {
	var pid = -1
	for i, e := range queuePolicyConfigs {
		if e.PolicyId == policyId {
			pid = i
			break
		}
	}
	if pid == -1 {
		return -1, "no policy " + policyId
	}
	var index = -1
	for i, r := range queuePolicyConfigs[pid].Rules {
		if r.Port == rule.Port &&
			r.Direction == rule.Direction {
			index = i
			break
		}
	}
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	if index != -1 {
		queuePolicyConfigs[pid].Rules = append(queuePolicyConfigs[pid].Rules[:index],
			queuePolicyConfigs[pid].Rules[index+1:]...)
		buildMap()
		saveConfigsTofile()
		recycleClients()
		return len(queuePolicyConfigs[pid].Rules), "delete success."
	} else {
		return -1, "can not find matched rule for policy " + policyId
	}
}

func saveConfigsTofile() {
	b, err := json.MarshalIndent(queuePolicyConfigs, "", "    ")
	for _, p := range queuePolicyConfigs {
		vlogger.Logger.Printf("Save policy %s -> %s", p.PolicyId, p.TargetQueues)
		for _, r := range p.Rules {
			r.TargetQueues = p.TargetQueues
			vlogger.Logger.Printf("r is %s -> %s", r.Source, r.TargetQueues)
		}
	}
	if err == nil {
		_ = ioutil.WriteFile(utils.Opts.QueueForwardFile, b, 0x777)
	}
}

func recycleClients() {
	go func() {
		usedClient := make(map[string]string)
		for _, policy := range queuePolicyConfigs {
			vlogger.Logger.Printf("check rule for policy %s, rules length is %d.\r\n", policy.PolicyId, len(policy.Rules))
			for _, ecr := range policy.Rules {
				//找到在用的
				for _, dist := range ecr.TargetQueues {
					dstAddr := dist
					if _, ok := queueTopics[dstAddr]; ok {
						vlogger.Logger.Printf("used address add %s .\r\n", dstAddr)
						usedClient[dstAddr] = dist
					}
				}
			}
		}
		for _, mirrorConfig := range queuePolicyConfigs {
			for _, ecr := range mirrorConfig.Rules {
				//在用的不存在了
				for _, dstTopic := range ecr.TargetQueues {
					if _, ok := usedClient[dstTopic]; !ok {
						vlogger.Logger.Printf("recycle dst queue %s .\r\n", dstTopic)
						delete(queueTopics, dstTopic)
					}
				}
			}
		}
	}()
}
