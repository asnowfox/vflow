package mirror

import (
	"log"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"fmt"
	"os"
)

var (
	Netflowv9Instance *Netflowv9Mirror
)

const InputId = 10
const OutputId = 14

//UDPQueue     int
//MessageQueue int
//UDPCount     uint64
//DecodedCount uint64
//MQErrorCount uint64
//Workers      int32
type Netflowv9MirrorStatus struct {
	QueueSize            int
	//MessageErrorCount    uint64
	MessageReceivedCount uint64
	RawSentCount         uint64
	RawErrorCount        uint64
}

type Config struct {
	Source string `yaml:"source"`
	Rules  []Rule `yaml:"rules"`
}
type Rule struct {
	InPort      uint16 `yaml:"inport"`
	OutPort     uint16 `yaml:"outport"`
	DistAddress string `yaml:"distAddress"`
}

func NewNetflowv9Mirror(mirrorCfg string, logger *log.Logger) (*Netflowv9Mirror, error) {
	ume := new(Netflowv9Mirror)
	ume.Logger = logger
	ume.mirrorCfgFile = mirrorCfg
	b, err := ioutil.ReadFile(mirrorCfg)
	if err != nil {
		logger.Printf("No Mirror config file is defined. \n")
		fmt.Printf("No Mirror config file is defined. \n")
		//os.Exit(-1)
		return nil, err
	}
	err = yaml.Unmarshal(b, &ume.mirrorConfigs)
	if err != nil {
		logger.Printf("Mirror config file is worong, exit! \n")
		fmt.Printf("Mirror config file is worong,exit! \n")
		os.Exit(-1)
		return ume, err
	}
	ume.initMap()

	Netflowv9Instance = ume
	return ume, nil
}
