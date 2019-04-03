package snmp

import (
	"time"
	"github.com/alouca/gosnmp"
	"fmt"
	"os"
	"io/ioutil"
	"encoding/json"
	"../vlogger"
	"sync"
	"errors"
	"net"
	"strconv"
)

var ifNameOid = ".1.3.6.1.2.1.31.1.1.1.1"
var ifIndexOid   = ".1.3.6.1.2.1.2.2.1.1"
var ifOperStatus = ".1.3.6.1.2.1.2.2.1.8"
var ifDesOid = ".1.3.6.1.2.1.31.1.1.1.18"
var ifOutOct = ".1.3.6.1.2.1.31.1.1.1.10"
var ifInOct = ".1.3.6.1.2.1.31.1.1.1.6"
var nfIndexOid = ".1.3.6.1.4.1.2011.5.25.110.1.2.1.2"

var devicePortMap = make(map[string][]PortInfo)
var devicePortIndexMap = make(map[string]map[int]PortInfo)
var rwLock = new(sync.RWMutex)

var ManageInstance *DevicePortManager
var snmpCfgFile string
var cfg DeviceSnmpConfig

type DevicePortManager struct {
	snmpConfigs DeviceSnmpConfig
}

type PortInfo struct {
	IfIndex int    `json:"ifIndex"`
	IfName  string `json:"ifName"`
	IfDes   string `json:"ifDes"`
	NfIndex int    `json:"nfIndex"`
}

type DeviceSnmpConfig struct {
	Interval  int32             `json:"interval"`
	DeviceCfg []CommunityConfig `json:"devices"`
}

type CommunityConfig struct {
	DeviceAddress string `json:"deviceIp"`
	Community     string `json:"community"`
}

func NewDevicePortManager(cfgFile string) (*DevicePortManager, error) {
	snmpCfgFile = cfgFile
	ManageInstance = new(DevicePortManager)

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
	ManageInstance.snmpConfigs = cfg
	return ManageInstance, nil
}

func (task *DevicePortManager) Run() {
	go func() {
		task.taskOnce(time.Now(), false)
	}()

	go func() {
		sleepSecond := 120 - time.Now().Unix()%120
		vlogger.Logger.Printf("I will delay for %d", sleepSecond)
		time.Sleep(time.Duration(sleepSecond) * time.Second)
		duration := time.Duration(time.Duration(task.snmpConfigs.Interval) * time.Second)
		timer1 := time.NewTicker(duration)
		for {
			select {
			case <-timer1.C:
				curTime := time.Now()
				task.taskOnce(curTime, true)
			}
		}
	}()
}

func (task *DevicePortManager) taskOnce(curTime time.Time, isSave bool) {
	for _, dev := range task.snmpConfigs.DeviceCfg {
		addr := dev.DeviceAddress
		community := dev.Community
		go func() {
			task.walkIndex(curTime, addr, community, isSave)
		}()
	}
}

type NameIndex struct {
	IfName  string
	IfIndex string
}

func (task *DevicePortManager) walkIndex(curTime time.Time, DeviceAddress string, Community string, isSave bool) error {
	s, err := gosnmp.NewGoSNMP(DeviceAddress, Community, gosnmp.Version2c, 5)
	if err != nil {
		vlogger.Logger.Fatal(err)
	}

	indexList := make([]int, 0)
	nameList := make([]string, 0)
	desList := make([]string, 0)
	ifInOctList := make([]uint64, 0)
	ifOutOctList := make([]uint64, 0)
	//nfIndexList :=make([]int,0)
	statusList := make([]int, 0)
	ifToNfIndexMap := make(map[int]int)

	inResp, err := s.Walk(ifInOct)
	if err == nil {
		for _, v := range inResp {
			ifInOctList = append(ifInOctList, v.Value.(uint64))
		}
	} else {
		vlogger.Logger.Printf("snmp walk err1 %e", err)
		return err
	}

	outResp, err := s.Walk(ifOutOct)
	if err == nil {
		for _, v := range outResp {
			ifOutOctList = append(ifOutOctList, v.Value.(uint64))
		}
	} else {
		vlogger.Logger.Printf("snmp walk err2 %e", err)
		return err
	}

	indexResp, err := s.Walk(ifIndexOid)
	if err == nil {
		for _, v := range indexResp {
			indexList = append(indexList, v.Value.(int))
		}
	} else {
		vlogger.Logger.Printf("snmp walk err3 %e", err)
		return err
	}

	statusResp, err := s.Walk(ifOperStatus)
	if err == nil {
		for _, v := range statusResp {
			statusList = append(statusList, v.Value.(int))
		}
	} else {
		vlogger.Logger.Printf("snmp walk err3 %e", err)
		return err
	}
	nfIndexResp, err := s.Walk(nfIndexOid)
	if err == nil {
		for _, v := range nfIndexResp {
			ofIndexStr := v.Name[len(nfIndexOid)+1 : len(v.Name)]
			ofIndex, _ := strconv.Atoi(ofIndexStr)
			ifToNfIndexMap[v.Value.(int)] = ofIndex
		}
	}
	if len(ifToNfIndexMap) == 0 {
		for _, v := range indexList {
			ifToNfIndexMap[v] = v
		}
	}

	nameResp, err := s.Walk(ifNameOid)
	if err == nil {
		for _, v := range nameResp {
			nameList = append(nameList, v.Value.(string))
		}
	} else {
		vlogger.Logger.Printf("snmp walk err4 %e", err)
		return err
	}
	desResp, err := s.Walk(ifDesOid)
	if err == nil {
		for _, v := range desResp {
			desList = append(desList, v.Value.(string))
		}
	} else {
		vlogger.Logger.Printf("snmp walk err5 %e", err)
		return err
	}

	rwLock.Lock()
	defer rwLock.Unlock()

	if (len(indexList) == len(nameList)) && (len(indexList) == len(desList)) {
		devicePortMap[DeviceAddress] = make([]PortInfo, 0)
		devicePortIndexMap[DeviceAddress] = make(map[int]PortInfo) //清空
		for i, index := range indexList {
			info := PortInfo{index, nameList[i], desList[i], ifToNfIndexMap[index]}
			devicePortMap[DeviceAddress] = append(devicePortMap[DeviceAddress], info)
			devicePortIndexMap[DeviceAddress][info.NfIndex] = info
		}

		if isSave {
			if len(indexList) == len(nameList) && len(nameList) == len(desList) && len(desList) == len(ifInOct) && len(ifOutOct) == len(statusList){
				SaveWalkToInflux(curTime, DeviceAddress, indexList, nameList, desList, ifInOctList, ifOutOctList, statusList, ifToNfIndexMap)
			}
		}
	} else {
		return errors.New("snmp walk err response is not equal")
		vlogger.Logger.Printf("snmp walk err response is not equal")
	}
	return nil
}

func (task *DevicePortManager) RefreshConfig(deviceIp string) ([]PortInfo, error) {
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
		err := task.walkIndex(time.Now(), deviceIp, community, false)
		if err == nil {
			return devicePortMap[deviceIp], nil
		} else {
			return make([]PortInfo, 0), err
		}

	} else {
		return make([]PortInfo, 0), errors.New("no device " + deviceIp)
	}
}

func (task *DevicePortManager) AddConfig(DeviceCfg CommunityConfig) (int, string) {
	for _, addr := range task.snmpConfigs.DeviceCfg {
		if addr.DeviceAddress == DeviceCfg.DeviceAddress {
			return -1, "config exist!"
		}
	}
	a := net.ParseIP(DeviceCfg.DeviceAddress)
	if a == nil {
		return -1, "invalid ip address"
	}
	task.snmpConfigs.DeviceCfg = append(task.snmpConfigs.DeviceCfg, DeviceCfg)
	err := saveConfigToFile()
	if err != nil {
		return -1, "save config to file error"
	}
	return len(task.snmpConfigs.DeviceCfg), "add success!"
}

func (task *DevicePortManager) DeleteConfig(deviceAddr string) (int, string) {
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

func (task *DevicePortManager) ListConfig() ([]CommunityConfig) {
	return task.snmpConfigs.DeviceCfg
}

func (task *DevicePortManager) ListPortInfo(devAddress string) ([]PortInfo) {
	rwLock.Lock()
	defer rwLock.Unlock()
	return devicePortMap[devAddress]
}

func (task *DevicePortManager) PortInfo(devAddress string, nfIndex int) (PortInfo, error) {
	rwLock.Lock()
	defer rwLock.Unlock()
	if nfIndex == -1 {
		info := PortInfo{
			-1, "all port", "match all port", -1,
		}
		return info, nil
	}
	if _, ok := devicePortMap[devAddress]; !ok {
		return *new(PortInfo), errors.New("Can not find device")
	}
	if _, ok := devicePortIndexMap[devAddress][nfIndex]; !ok {
		return *new(PortInfo), errors.New("Can not find index")
	}
	return devicePortIndexMap[devAddress][nfIndex], nil
}

func saveConfigToFile() error {
	b, err := json.MarshalIndent(ManageInstance.snmpConfigs, "", "    ")
	if err == nil {
		return ioutil.WriteFile(snmpCfgFile, b, 0x777)
	} else {
		return err
	}
}
