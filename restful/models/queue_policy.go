package models

import (
	"github.com/VerizonDigital/vflow/producer"
	"github.com/VerizonDigital/vflow/snmp"
	"strconv"
)

type QueuePolicyModel struct {
	PolicyId     string           `json:"policyId"`
	TargetQueues string           `json:"targetQueue"`
	Enable       int              `json:"enable"`
	Rules        []QueueRuleModel `json:"rules"`
}

type QueueRuleModel struct {
	producer.QueueRule
	RuleId       string `json:"ruleId"`
	PortName     string `json:"portName"`
	PortDes      string `json:"portDes"`
	DirectionDes string `json:"directionDes"`
	Direction    int    `json:"direction"`
}

func TransQueuePolicy(p producer.QueuePolicy) QueuePolicyModel {
	policy := new(QueuePolicyModel)
	policy.Enable = p.Enable
	policy.PolicyId = p.PolicyId
	for _, target := range p.TargetQueues {
		policy.TargetQueues += target + ","
	}
	a := len(policy.TargetQueues)
	if a > 0 {
		policy.TargetQueues = policy.TargetQueues[0 : a-1]
	}
	for _, r := range p.Rules {
		policy.Rules = append(policy.Rules, TransQueueRule(r.Source, r))
	}
	return *policy
}

func TransQueueRule(deviceIp string, r producer.QueueRule) QueueRuleModel {
	rule := new(QueueRuleModel)
	rule.QueueRule = r
	rule.RuleId = strconv.Itoa(int(r.Port)) + "_" + strconv.Itoa(int(r.Direction)) + "_" + r.Source
	//nfindex inport is nfindex
	iPortInfo, err := snmp.ManageInstance.PortInfo(deviceIp, int(r.Port))
	if err == nil {
		rule.PortName = iPortInfo.IfName
		rule.PortDes = iPortInfo.IfDes
		rule.Direction = r.Direction
		if rule.Direction == 0 {
			rule.DirectionDes = "入方向"
		} else if rule.Direction == 1 {
			rule.DirectionDes = "出方向"
		} else {
			rule.DirectionDes = "双方向"
		}
	}
	return *rule
}
