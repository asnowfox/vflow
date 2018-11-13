package models

import (
	"../../mirror"
	"../../snmp"
)

type RPolicy struct {
	PolicyId      string `json:"policyId"`
	TargetAddress string `json:"targetAddress"`
	Enable        int    `json:"enable"`
	Rules         []RRule `json:"rules"`
}

type RRule struct {
	mirror.Rule
	InportName string `json:"inportName"`
	InportDes string `json:"inportDes"`
	OutportName string `json:"outportName"`
	OutportDes string `json:"outportDes"`
}

func TransPolicy(p mirror.Policy) RPolicy  {
	policy := new(RPolicy)
	policy.Enable = p.Enable
	policy.PolicyId = p.PolicyId
	policy.TargetAddress = p.TargetAddress
	for _,r := range p.Rules {
		policy.Rules = append(policy.Rules, TransRule(r.Source,r))
	}
	return *policy
}

func TransRule(deviceIp string,r mirror.Rule)RRule{
	rule := new(RRule)
	rule.Rule = r
	iPortInfo,err := snmp.SnmpTaskInstance.PortInfo(deviceIp,int(r.InPort))
	if err== nil {
		rule.InportName = iPortInfo.IfName
		rule.InportDes = iPortInfo.IfDes
	}
	oPortInfo,err := snmp.SnmpTaskInstance.PortInfo(deviceIp,int(r.OutPort))
	if err== nil {
		rule.OutportName = oPortInfo.IfName
		rule.OutportDes = oPortInfo.IfDes
	}
	return *rule
}