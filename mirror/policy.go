package mirror

import (
	"io/ioutil"
	"fmt"
	"os"
	"encoding/json"
)

type Policy struct {
	PolicyId string `json:"policyId"`
	Rules  []Rule `json:"rules"`
}

type Rule struct {
	Source      string `json:"source"`
	InPort      int32 `json:"inport"`
	OutPort     int32 `json:"outport"`
	DistAddress string `json:"distAddress"`
}

func LoadPolicy(mirrorCfg string) error{
	b, err := ioutil.ReadFile(mirrorCfg)
	if err != nil {
		//log.Printf("No Mirror config file is defined. \n")
		fmt.Printf("No Mirror config file is defined. \n")
		return  err
	}
	err = json.Unmarshal(b, &policyConfigs)
	if err != nil {
		//log.Printf("Mirror config file is worong, exit! \n")
		fmt.Printf("Mirror config file is worong,exit! \n")
		os.Exit(-1)
		return  err
	}
	return nil
}
