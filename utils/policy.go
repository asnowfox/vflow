package utils

type Policy struct {
	PolicyId      string   `json:"policyId"`
	TargetAddress []string `json:"targetAddress"`
	Enable        int      `json:"enable"`
	Rules         []Rule   `json:"rules"`
}

type Rule struct {
	Source      string   `json:"source"`
	Port        int32    `json:"port"`
	Direction   int      `json:"direction"`
	DistAddress []string `json:"distAddress"`
}
