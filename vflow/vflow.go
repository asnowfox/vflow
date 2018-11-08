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
package vflow

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"../restful"
	"../mirror"
	"../flows"
)

var (
	opts   *flows.Options
	logger *log.Logger
)



func main() {
	var (
		wg       sync.WaitGroup
		signalCh = make(chan os.Signal, 1)
	)

	opts = flows.GetOptions()

	runtime.GOMAXPROCS(opts.GetCPU())
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	logger = opts.Logger

	logger.Printf("startting flow mirror with config file %s....\n",opts.ForwardFile)
	mirror.Init(opts.ForwardFile,logger)

	flowMirror,err1 := mirror.NewNetFlowv9Mirror()
	ipfixMirror,err2 := mirror.NewIPFixMirror()

	if err1 != nil {
		logger.Printf("can not init netflow mirror. reason %s\n", err1)
	}else{
		flowMirror.Run()
	}
	if err2 != nil {
		logger.Printf("can not init ipfix mirror. reason %s\n", err2)
	}else{
		ipfixMirror.Run()
	}

	sFlow := flows.NewSFlow()

	ipfix := flows.NewIPFIX(ipfixMirror)
	netflow9 := flows.NewNetflowV9(flowMirror)


	protos := []flows.Proto{sFlow, ipfix, netflow9}

	for _, p := range protos {
		wg.Add(1)
		go func(p flows.Proto) {
			defer wg.Done()
			p.Run()
		}(p)
	}

	//go statsHTTPServer(ipfix, sFlow, netflow9, flowMirror)

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
