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
	"sync"
	"errors"
)

var ifNameOid = ".1.3.6.1.2.1.31.1.1.1.1"
var ifIndexOid = ".1.3.6.1.2.1.2.2.1.1"
var ifDesOid = ".1.3.6.1.2.1.31.1.1.1.18"
var devicePortMap = make(map[string][]PortInfo)
var rwLock = new(sync.RWMutex)

var SnmpTaskInstance *WalkTask
var snmpCfgFile string
var cfg DeviceSnmpConfig

type WalkTask struct {
	snmpConfigs DeviceSnmpConfig
}

type PortInfo struct {
	IfIndex int    `json:"ifIndex"`
	IfName  string `json:"ifName"`
	IfDes   string `json:"ifDes"`
}

type DeviceSnmpConfig struct {
	Interval  int32             `json:"interval"`
	DeviceCfg []CommunityConfig `json:"devices"`
}

type CommunityConfig struct {
	DeviceAddress string `json:"DeviceAddress"`
	Community     string `json:"Community"`
}

func Init(cfgFile string) (*WalkTask, error) {
	snmpCfgFile = cfgFile
	SnmpTaskInstance = new(WalkTask)

	b, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		vlogger.Logger.Printf("No SNMP config file is defined. \n")
		fmt.Printf("No SNMP config file is defined. \n")
		return nil, err
	}

	fmt.Printf("config is %s", string(b))
	err = json.Unmarshal(b, &cfg)
	if err != nil {
		vlogger.Logger.Printf("SNMP config file is worong, exit! \n")
		fmt.Printf("SNMP config file is worong,exit! \n")
		os.Exit(-1)
		return nil, err
	}
	fmt.Printf("delay is %d. device length is %d\n", cfg.Interval, len(cfg.DeviceCfg))
	SnmpTaskInstance.snmpConfigs = cfg
	return SnmpTaskInstance, nil
}

func (task *WalkTask) Run() {
	go func() {
		task.task()
	}()

	go func() {
		duration := time.Duration(time.Duration(task.snmpConfigs.Interval) * time.Second)
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
	for _, dev := range task.snmpConfigs.DeviceCfg {
		task.walkIndex(dev.DeviceAddress, dev.Community)
	}
}

type NameIndex struct {
	IfName  string
	IfIndex string
}

func (task *WalkTask) walkIndex(DeviceAddress string, Community string) error {
	s, err := gosnmp.NewGoSNMP(DeviceAddress, Community, gosnmp.Version2c, 5)
	if err != nil {
		log.Fatal(err)
	}
	indexResp, err := s.Walk(ifIndexOid)

	indexList := make([]int, 0)
	nameList := make([]string, 0)
	desList := make([]string, 0)

	if err == nil {
		for _, v := range indexResp {
			log.Printf("Response: %s : %d : %s \n",
				v.Name, v.Value, v.Type.String())
			indexList = append(indexList, v.Value.(int))
		}
	} else {
		log.Printf("snmp walk err %e", err)
		return err
	}

	nameResp, err := s.Walk(ifNameOid)
	if err == nil {
		for _, v := range nameResp {
			log.Printf("Response: %s : %s : %s \n",
				v.Name, v.Value, v.Type.String())
			nameList = append(nameList, v.Value.(string))
		}
	} else {
		log.Printf("snmp walk err %e", err)
		return err
	}

	desResp, err := s.Walk(ifDesOid)
	if err == nil {
		for _, v := range desResp {
			log.Printf("Response: %s : %s : %s \n",
				v.Name, v.Value, v.Type.String())
			desList = append(desList, v.Value.(string))
		}
	} else {
		log.Printf("snmp walk err %e", err)
		return err
	}

	rwLock.RLock()
	defer rwLock.RUnlock()

	if (len(indexList) == len(nameList)) && (len(indexList) == len(desList)) {
		devicePortMap[DeviceAddress] = make([]PortInfo, 0)
		for i, index := range indexList {
			info := PortInfo{index, nameList[i], desList[i]}
			devicePortMap[DeviceAddress] = append(devicePortMap[DeviceAddress], info)
		}
	} else {
		return errors.New("snmp walk err response is not equal")
		log.Printf("snmp walk err response is not equal")
	}
	return nil
}

func (task *WalkTask) RefreshConfig(deviceIp string) ([]PortInfo, error) {
	community := ""
	found := false
	for _, devCfg := range task.snmpConfigs.DeviceCfg {
		if devCfg.DeviceAddress == deviceIp {
			found = true
			community = devCfg.Community
			break
		}
	}
	if found {
		err := task.walkIndex(deviceIp, community)
		if err == nil {
			return devicePortMap[deviceIp], nil
		} else {
			return make([]PortInfo, 0), err
		}

	} else {
		return make([]PortInfo, 0), errors.New("no device " + deviceIp)
	}
}

func (task *WalkTask) AddConfig(DeviceCfg CommunityConfig) (int, string) {
	for _, addr := range task.snmpConfigs.DeviceCfg {
		if addr.DeviceAddress == DeviceCfg.DeviceAddress {
			return -1, "config exist!"
		}
	}
	task.snmpConfigs.DeviceCfg = append(task.snmpConfigs.DeviceCfg, DeviceCfg)
	return len(task.snmpConfigs.DeviceCfg), "add success!"
}

func (task *WalkTask) DeleteConfig(deviceAddr string) (int, string) {
	index := -1
	for i, addr := range task.snmpConfigs.DeviceCfg {
		if addr.DeviceAddress == deviceAddr {
			index = i
			break
		}
	}
	if index == -1 {
		return -1, "can not find address " + deviceAddr
	}
	task.snmpConfigs.DeviceCfg = append(task.snmpConfigs.DeviceCfg[:index],
		task.snmpConfigs.DeviceCfg[index+1:]...)
	err := saveConfigToFile()
	if err != nil {
		return -1, "save config to file error"
	}
	return len(task.snmpConfigs.DeviceCfg), "delete success!"
}

func (task *WalkTask) ListConfig() ([]CommunityConfig) {
	return task.snmpConfigs.DeviceCfg
}

func (task *WalkTask) ListPortInfo(devAddress string) ([]PortInfo) {
	rwLock.RLock()
	defer rwLock.RLock()
	return devicePortMap[devAddress]
}

func saveConfigToFile() error {
	b, err := json.MarshalIndent(SnmpTaskInstance.snmpConfigs, "", "    ")
	if err == nil {
		return ioutil.WriteFile(snmpCfgFile, b, 0x777)
	} else {
		return err
	}
}
