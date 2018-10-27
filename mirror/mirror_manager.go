package mirror

import (
	"log"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"fmt"
	"os"
)

var(
	Netflowv9Instance *Netflowv9Mirror
)

const InputId = 10
const OutputId = 14

type Status struct {
	StartTime int64
	QeueSize  int32
}

type Config struct {
	Source string `yaml:"source"`
	Rules  []Rule `yaml:"rules"`
}
type Rule struct {
	InPort      uint16  `yaml:"inport"`
	OutPort     uint16  `yaml:"outport"`
	DistAddress string `yaml:"distAddress"`
}


func NewNetflowv9Mirror(mirrorCfg string, logger *log.Logger, mirrorInfIp string) (*Netflowv9Mirror, error) {
	ume := new(Netflowv9Mirror)
	ume.Logger = logger
	ume.mirrorCfgFile = mirrorCfg
	b, err := ioutil.ReadFile(mirrorCfg)
	if err != nil {
		logger.Printf("Mirror config file is worong, exit! \n",mirrorInfIp)
		fmt.Printf("Mirror config file is worong,exit! \n",mirrorInfIp)
		os.Exit(-1)
		return nil, err
	}
	err = yaml.Unmarshal(b, &ume.mirrorConfigs)
	if err != nil {
		logger.Printf("Mirror config file is worong, exit! \n",mirrorInfIp)
		fmt.Printf("Mirror config file is worong,exit! \n",mirrorInfIp)
		os.Exit(-1)
		return ume, err
	}
	ume.initMap()
	fmt.Printf("Starting raw socket on interface %s....\n",mirrorInfIp)


	Netflowv9Instance = ume
	return ume, nil
}
