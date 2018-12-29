package mirror

import (
	"io/ioutil"
	"fmt"
	"os"
	"encoding/json"
	"../vlogger"
)

type Policy struct {
	PolicyId      string `json:"policyId"`
	TargetAddress []string `json:"targetAddress"`
	Enable        int    `json:"enable"`
	Rules         []Rule `json:"rules"`
}

type Rule struct {
	Source      string `json:"source"`
	Port      int32  `json:"port"`
	Direction     int32  `json:"direction"`
	DistAddress []string `json:"distAddress"`
}

func LoadPolicy(mirrorCfg string) error {
	b, err := ioutil.ReadFile(mirrorCfg)
	if err != nil {
		vlogger.Logger.Printf("No Mirror config file is defined. \n")
		fmt.Printf("No Mirror config file is defined. \n")
		return err
	}
	err = json.Unmarshal(b, &policyConfigs)
	if err != nil {
		vlogger.Logger.Printf("Mirror config file is worong, exit! \n")
		fmt.Printf("Mirror config file is worong,exit! \n")
		os.Exit(-1)
		return err
	}
	return nil
}
