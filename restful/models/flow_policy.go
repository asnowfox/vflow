package models

import (
	"github.com/VerizonDigital/vflow/snmp"
	"github.com/VerizonDigital/vflow/utils"
	"strconv"
)

type RPolicy struct {
	PolicyId      string  `json:"policyId"`
	TargetAddress string  `json:"targetAddress"`
	Enable        int     `json:"enable"`
	Rules         []RRule `json:"rules"`
}

type RRule struct {
	utils.Rule
	RuleId       string `json:"ruleId"`
	PortName     string `json:"portName"`
	PortDes      string `json:"portDes"`
	DirectionDes string `json:"directionDes"`
	Direction    int    `json:"direction"`
}

func TransPolicy(p utils.Policy) RPolicy {
	policy := new(RPolicy)
	policy.Enable = p.Enable
	policy.PolicyId = p.PolicyId
	for _, target := range p.TargetAddress {
		policy.TargetAddress += target + ","
	}
	a := len(policy.TargetAddress)
	if a > 0 {
		policy.TargetAddress = policy.TargetAddress[0 : a-1]
	}
	for _, r := range p.Rules {
		policy.Rules = append(policy.Rules, TransRule(r.Source, r))
	}
	return *policy
}

func TransRule(deviceIp string, r utils.Rule) RRule {
	rule := new(RRule)
	rule.Rule = r
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
