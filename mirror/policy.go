package mirror

import (
	"io/ioutil"
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
)

type Policy struct {
	PolicyId string `yaml:"policyId"`
	Rules  []Rule `yaml:"rules"`
}


type Rule struct {
	Source      string `yaml:"source""`
	InPort      int32 `yaml:"inport"`
	OutPort     int32 `yaml:"outport"`
	DistAddress string `yaml:"distAddress"`
}

func LoadPolicy(mirrorCfg string) error{
	b, err := ioutil.ReadFile(mirrorCfg)
	if err != nil {
		//log.Printf("No Mirror config file is defined. \n")
		fmt.Printf("No Mirror config file is defined. \n")
		return  err
	}
	err = yaml.Unmarshal(b, &policyConfigs)
	if err != nil {
		//log.Printf("Mirror config file is worong, exit! \n")
		fmt.Printf("Mirror config file is worong,exit! \n")
		os.Exit(-1)
		return  err
	}
	return nil
}
