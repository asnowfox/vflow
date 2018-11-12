package snmp

import (
	"time"
	"github.com/alouca/gosnmp"
	"log"
	"fmt"
	"os"
	"io/ioutil"
	"encoding/json"
	"../vlogger"
)

var ifNameOid = ".1.3.6.1.2.1.31.1.1.1.1"
var ifIndexOid = ".1.3.6.1.2.1.2.2.1.1"
var ifDesOid = ".1.3.6.1.2.1.31.1.1.1.18"

type WalkTask struct {
	snmpConfigs DeviceSnmpConfig
}
/***

{
    "interval":30,
    "devices":[

        {
            "deviceAddress": "159.226.8.131",
            "community":"cst*net"
        },
        {
            "deviceAddress": "10.0.0.2",
            "community":"public"
        },
        {
            "deviceAddress": "10.0.0.3",
            "community":"public"
        }
    ]
}



 */

type DeviceSnmpConfig struct {
	delay     int32             `json:"interval"`
	deviceCfg []CommunityConfig `json:"devices"`
}

type CommunityConfig struct {
	deviceAddress string `json:"deviceAddress"`
	community     string `json:"community"`
}

var snmpTaskInstance *WalkTask
var snmpCfgFile string
var cfg DeviceSnmpConfig

func Init(cfgFile string) (*WalkTask,error) {
	snmpCfgFile = cfgFile
	snmpTaskInstance = new(WalkTask)

	b, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		vlogger.Logger.Printf("No SNMP config file is defined. \n")
		fmt.Printf("No SNMP config file is defined. \n")
		return nil,err
	}

	fmt.Printf("file is %s",string(b))
	err = json.Unmarshal(b, &cfg)
	if err != nil {
		vlogger.Logger.Printf("SNMP config file is worong, exit! \n")
		fmt.Printf("SNMP config file is worong,exit! \n")
		os.Exit(-1)
		return  nil,err
	}
	fmt.Printf("delay is %d. device length is %d\n",cfg.delay, len(cfg.deviceCfg))
	snmpTaskInstance.snmpConfigs = cfg
	return snmpTaskInstance,nil
}

func (task *WalkTask) Run() {
	go func() {
		duration := time.Duration(time.Duration(task.snmpConfigs.delay) * time.Second)
		timer1 := time.NewTicker(duration)
		for {
			select {
			case <-timer1.C:
				task.task()
			}
		}
	}()
}

func (task *WalkTask) task() {
	for _, dev := range task.snmpConfigs.deviceCfg {
		task.walkIndex(dev.deviceAddress, dev.community)
	}
}

type NameIndex struct {
	IfName  string
	IfIndex string
}

func (task *WalkTask) walkIndex(deviceAddress string, community string) {
	s, err := gosnmp.NewGoSNMP(deviceAddress, community, gosnmp.Version2c, 5)
	if err != nil {
		log.Fatal(err)
	}
	indexResp, err := s.Walk(ifIndexOid)

	if err == nil {
		for _, v := range indexResp {
			switch v.Type {
			case gosnmp.OctetString:
				log.Printf("Response: %s : %s : %s \n", v.Name, v.Value.(string), v.Type.String())
			}
		}
	} else {
		log.Printf("snmp walk err %e", err)
	}

	nameResp, err := s.Walk(ifNameOid)
	if err == nil {
		for _, v := range nameResp {
			switch v.Type {
			case gosnmp.OctetString:
				log.Printf("Response: %s : %s : %s \n", v.Name, v.Value.(string), v.Type.String())
			}
		}
	} else {
		log.Printf("snmp walk err %e", err)
	}

	desResp, err := s.Walk(ifNameOid)
	if err == nil {
		for _, v := range desResp {
			switch v.Type {
			case gosnmp.OctetString:
				log.Printf("Response: %s : %s : %s \n", v.Name, v.Value.(string), v.Type.String())
			}
		}
	} else {
		log.Printf("snmp walk err %e", err)
	}
}


func (task *WalkTask) AddConfig(deviceCfg CommunityConfig) (int, string) {
	for _, addr := range task.snmpConfigs.deviceCfg {
		if addr.deviceAddress == deviceCfg.deviceAddress {
			return -1, "config exist!"
		}
	}
	task.snmpConfigs.deviceCfg = append(task.snmpConfigs.deviceCfg, deviceCfg)
	return len(task.snmpConfigs.deviceCfg), "add success!"
}

func (task *WalkTask) DeleteConfig(deviceAddr string) (int, string) {
	index := -1
	for i, addr := range task.snmpConfigs.deviceCfg {
		if addr.deviceAddress == deviceAddr {
			index = i
			break
		}
	}
	if index == -1 {
		return -1, "can not find address " + deviceAddr
	}
	task.snmpConfigs.deviceCfg = append(task.snmpConfigs.deviceCfg[:index],
		task.snmpConfigs.deviceCfg[index+1:]...)
	err := saveConfigToFile()
	if err != nil {
		return -1, "save config to file error"
	}
	return len(task.snmpConfigs.deviceCfg), "delete success!"
}

func saveConfigToFile() error {
	b, err := json.MarshalIndent(snmpTaskInstance.snmpConfigs, "", "    ")
	if err == nil {
		return ioutil.WriteFile(snmpCfgFile, b, 0x777)
	} else {
		return err
	}
}