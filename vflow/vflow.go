//: ----------------------------------------------------------------------------
//: Copyright (C) 2017 Verizon.  All Rights Reserved.
//: All Rights Reserved
//:
//: file:    vflow.go
//: details: TODO
//: author:  Mehrdad Arshad Rad
//: date:    02/01/2017
//:
//: Licensed under the Apache License, Version 2.0 (the "License");
//: you may not use this file except in compliance with the License.
//: You may obtain a copy of the License at
//:
//:     http://www.apache.org/licenses/LICENSE-2.0
//:
//: Unless required by applicable law or agreed to in writing, software
//: distributed under the License is distributed on an "AS IS" BASIS,
//: WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//: See the License for the specific language governing permissions and
//: limitations under the License.
//: ----------------------------------------------------------------------------

// Package main is the vflow binary
package main

import (
	"github.com/VerizonDigital/vflow/flows"
	"github.com/VerizonDigital/vflow/mirror"
	"github.com/VerizonDigital/vflow/producer"
	"github.com/VerizonDigital/vflow/restful"
	"github.com/VerizonDigital/vflow/snmp"
	"github.com/VerizonDigital/vflow/utils"
	"github.com/VerizonDigital/vflow/vlogger"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
)

var (
	opts   *utils.Options
	logger *log.Logger
)

func main() {
	var (
		wg       sync.WaitGroup
		signalCh = make(chan os.Signal, 1)
	)

	opts = utils.InitOptions()

	runtime.GOMAXPROCS(opts.GetCPU())
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	vlogger.Logger.Printf("startting flow mirror with config file %s....\n", opts.FlowForwardFile)
	task, _ := snmp.NewDevicePortManager(opts.CommunityFile)
	task.Run()
	// 初始化流转发文件
	err := mirror.Init(opts.FlowForwardFile)
	if err != nil {
		vlogger.Logger.Printf("Init forwarding error,Exit!")
		os.Exit(-1)
	}
	// 初始化消息转发文件
	err = producer.Init(opts.QueueForwardFile)
	if err != nil {
		vlogger.Logger.Printf("Init queue forward error,Exit!")
		os.Exit(-1)
	}

	flowMirror, err1 := mirror.NewNetflowv9Mirror()
	ipfixMirror, err2 := mirror.NewIPFixMirror()

	if err1 != nil {
		logger.Printf("can not init netflow mirror. reason %s\n", err1)
		os.Exit(-1)
	}
	if err2 != nil {
		logger.Printf("can not init ipfix mirror. reason %s\n", err2)
		os.Exit(-1)
	} else {
		ipfixMirror.Run()
	}

	sFlow := flows.NewSFlow()
	ipfix := flows.NewIPFIX(ipfixMirror)
	netflow9 := flows.NewNetflowV9(flowMirror)
	//delay int32,dstAddress[] DeviceSnmpConfig

	protos := []flows.Proto{sFlow, ipfix, netflow9}

	//利用wait group 确保三种协议启动
	for _, p := range protos {
		wg.Add(1)
		go func(p flows.Proto) {
			defer wg.Done()
			p.Run()
		}(p)
	}

	//go statsHTTPServer(ipfix, sFlow, netflow9, flowMirror)
	//启动BeegoServer
	beegoServer := restful.NewBeegoServer(netflow9)
	beegoServer.Run()

	<-signalCh

	for _, p := range protos {
		wg.Add(1)
		go func(p flows.Proto) {
			defer wg.Done()
			p.Shutdown()
		}(p)
	}
	wg.Wait()
}
