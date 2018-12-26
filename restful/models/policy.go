package models

import (
	"../../mirror"
	"../../snmp"
	"strconv"
)

type RPolicy struct {
	PolicyId      string `json:"policyId"`
	TargetAddress string `json:"targetAddress"`
	Enable        int    `json:"enable"`
	Rules         []RRule `json:"rules"`
}

type RRule struct {
	mirror.Rule
	RuleId string `json:"ruleId"`
	InportName string `json:"inportName"`
	InportDes string `json:"inportDes"`
	OutportName string `json:"outportName"`
	OutportDes string `json:"outportDes"`
}

func TransPolicy(p mirror.Policy) RPolicy  {
	policy := new(RPolicy)
	policy.Enable = p.Enable
	policy.PolicyId = p.PolicyId
	for _,target := range p.TargetAddress {
		policy.TargetAddress += target+","
	}
	a := len(policy.TargetAddress)
	if a>0{
		policy.TargetAddress = policy.TargetAddress[0:a-1]
	}
	for _,r := range p.Rules {
		policy.Rules = append(policy.Rules, TransRule(r.Source,r))
	}
	return *policy
}

func TransRule(deviceIp string,r mirror.Rule)RRule{
	rule := new(RRule)
	rule.Rule = r
	rule.RuleId = strconv.Itoa(int(r.InPort))+"_"+strconv.Itoa(int(r.OutPort))+"_"+r.Source
	//nfindex inport is nfindex
	iPortInfo,err := snmp.ManageInstance.PortInfo(deviceIp,int(r.InPort))
	if err== nil {
		rule.InportName = iPortInfo.IfName
		rule.InportDes = iPortInfo.IfDes
	}
	oPortInfo,err := snmp.ManageInstance.PortInfo(deviceIp,int(r.OutPort))
	if err== nil {
		rule.OutportName = oPortInfo.IfName
		rule.OutportDes = oPortInfo.IfDes
	}
	return *rule
}