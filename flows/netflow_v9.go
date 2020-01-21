//: ----------------------------------------------------------------------------
//: Copyright (C) 2017 Verizon.  All Rights Reserved.
//: All Rights Reserved
//:
//: file:    netflow_v9.go
//: details: netflow decoders handler
//: author:  Mehrdad Arshad Rad
//: date:    04/21/2017
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

package flows

import (
	"bytes"
	"github.com/VerizonDigital/vflow/mirror"
	"github.com/VerizonDigital/vflow/netflow/v9"
	"github.com/VerizonDigital/vflow/producer"
	"github.com/VerizonDigital/vflow/utils"
	"github.com/VerizonDigital/vflow/vlogger"
	"github.com/mohae/deepcopy"
	"net"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// NetflowV9 represents netflow v9 collector
type NetflowV9 struct {
	port      int
	addr      string
	workers   int
	stop      bool
	stats     NetflowV9Stats
	pktStat   PacketStatistics
	pool      chan chan struct{}
	udpMirror *mirror.Netflowv9Mirror
}

// NetflowV9UDPMsg represents netflow v9 UDP data
type NetflowV9UDPMsg struct {
	raddr *net.UDPAddr
	body  []byte
}

// NetflowV9Stats represents netflow v9 stats
type NetflowV9Stats struct {
	UDPQueue     int
	MessageQueue int
	UDPCount     uint64
	DecodedCount uint64
	MQErrorCount uint64
	LostCount    uint64
	StartTime    int64
	Workers      int32
}

var (
	netflowV9UDPCh         = make(chan NetflowV9UDPMsg, 10000)
	netflowV9MainMQChannel = make(chan producer.MQMessage, 10000)
	mCacheNF9              netflow9.MemCache
	// ipfix udp payload pool
	netflowV9Buffer = &sync.Pool{
		New: func() interface{} {
			return make([]byte, utils.Opts.NetflowV9UDPSize)
		},
	}
)

// NewNetflowV9 constructs NetflowV9
func NewNetflowV9(exc *mirror.Netflowv9Mirror) *NetflowV9 {
	return &NetflowV9{
		port:      utils.Opts.NetflowV9Port,
		workers:   utils.Opts.NetflowV9Workers,
		pool:      make(chan chan struct{}, utils.MaxWorkers),
		udpMirror: exc,
	}
}

func (i *NetflowV9) Run() {

	// exit if the netflow v9 is disabled
	if !utils.Opts.NetflowV9Enabled {
		vlogger.Logger.Println("netflowv9 has been disabled")
		return
	}
	i.pktStat = *NewPacketStatistics()

	hostPort := net.JoinHostPort(i.addr, strconv.Itoa(i.port))
	udpAddr, _ := net.ResolveUDPAddr("udp", hostPort)

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		vlogger.Logger.Fatal(err)
	}

	atomic.AddInt32(&i.stats.Workers, int32(i.workers))
	for n := 0; n < i.workers; n++ {
		go func() {
			wQuit := make(chan struct{})
			i.pool <- wQuit
			i.netflowV9Worker(wQuit)
		}()
	}

	vlogger.Logger.Printf("netflow v9 is running (UDP: listening on [::]:%d workers#: %d)", i.port, i.workers)
	i.stats.StartTime = time.Now().Unix()

	mCacheNF9 = netflow9.GetCache(utils.Opts.NetflowV9TplCacheFile)
	if utils.MqEnabled {
		go func() {
			// 启动Producer 发送消息到消息队列
			p := producer.NewProducer(utils.Opts.MQName)
			p.MQConfigFile = path.Join(utils.Opts.VFlowConfigPath, utils.Opts.MQConfigFile)
			p.MQErrorCount = &i.stats.MQErrorCount
			p.Logger = vlogger.Logger
			p.Chan = netflowV9MainMQChannel

			if err := p.Run(); err != nil {
				vlogger.Logger.Fatal(err)
			}
		}()
	} else {
		vlogger.Logger.Printf("disable netflow v9 json mq transfer")
	}
	go func() {
		if !utils.Opts.DynWorkers {
			vlogger.Logger.Println("netflow v9 dynamic worker disabled")
			return
		}
		i.dynWorkers()
	}()

	for !i.stop {
		b := netflowV9Buffer.Get().([]byte)
		_ = conn.SetReadDeadline(time.Now().Add(1e9))
		n, remoteAddress, err := conn.ReadFromUDP(b)
		if err != nil {
			continue
		}
		atomic.AddUint64(&i.stats.UDPCount, 1)
		netflowV9UDPCh <- NetflowV9UDPMsg{remoteAddress, b[:n]}
	}
}

func (i *NetflowV9) Shutdown() {
	// exit if the netflow v9 is disabled
	if !utils.Opts.NetflowV9Enabled {
		vlogger.Logger.Println("netflow v9 disabled")
		return
	}

	// stop reading from UDP listener
	i.stop = true
	vlogger.Logger.Println("stopping netflow v9 service gracefully ...")
	time.Sleep(1 * time.Second)

	// dump the templates to storage
	if err := mCacheNF9.Dump(utils.Opts.NetflowV9TplCacheFile); err != nil {
		vlogger.Logger.Println("couldn't not dump template", err)
	}

	// logging and close UDP channel
	vlogger.Logger.Println("netflow v9 has been shutdown")
	close(netflowV9UDPCh)
}

func (i *NetflowV9) netflowV9Worker(wQuit chan struct{}) {
	var (
		decodedMsg *netflow9.Message
		msg        = NetflowV9UDPMsg{body: netflowV9Buffer.Get().([]byte)}
		buf        = new(bytes.Buffer)
		err        error
		ok         bool
		b          []byte
	)

LOOP:
	for {
		netflowV9Buffer.Put(msg.body[:utils.Opts.NetflowV9UDPSize])
		buf.Reset()

		select {
		case <-wQuit:
			break LOOP
		case msg, ok = <-netflowV9UDPCh:
			if !ok {
				break LOOP
			}
		}

		d := netflow9.NewDecoder(msg.raddr.IP, msg.body)
		if decodedMsg, err = d.Decode(mCacheNF9); err != nil {
			vlogger.Logger.Printf("%s decode data error: %e", msg.raddr.IP.String(), err)
			if decodedMsg == nil {
				continue
			}
		}
		//所有的worker的消息由 messageMirror接收
		if i.udpMirror != nil {
			msg := *decodedMsg
			i.udpMirror.ReceiveMessage(msg)
		}

		atomic.AddUint64(&i.stats.DecodedCount, 1)
		i.pktStat.recordSeq(decodedMsg.AgentID, decodedMsg.Header.SrcID, decodedMsg.Header.SeqNum)

		dstMessage := deepcopy.Copy(*decodedMsg).(netflow9.Message)

		if dstMessage.DataFlowSets != nil && utils.MqEnabled {
			for _, e := range dstMessage.DataFlowSets {
				b, err = dstMessage.JSONMarshal(buf, e.DataFlowRecords)
				if err != nil {
					vlogger.Logger.Println(err)
					continue
				}

				netflowV9MainMQChannel <- producer.MQMessage{Topic: utils.Opts.NetflowV9Topic, Msg: string(b[:])} //append([]byte{}, b...)
				if len(netflowV9MainMQChannel) >= 10000 {
					vlogger.Logger.Printf("current kafka channel length is great than 10000, length is %d .", len(netflowV9MainMQChannel))
				}

				if utils.Opts.Verbose {
					vlogger.Logger.Println(string(b))
				}
			} // 发送到主mq之后
			//这里匹配分发规则到各个消息队列
			forwardMessageToSubMQ(&dstMessage)
		} else if dstMessage.DataFlowSets == nil {
			vlogger.Logger.Printf("DecodedMsg.DataFlowSets is nil, AgentId %s. seqNum is %d.", decodedMsg.AgentID, decodedMsg.Header.SeqNum)
		}
	}
}

func forwardMessageToSubMQ(decodedMsg *netflow9.Message) {
	topicDataFlowSet := make(map[string][]netflow9.DataFlowRecord)
	for _, e := range decodedMsg.DataFlowSets {
		for _, record := range e.DataFlowRecords {

			topics := producer.ParseTopic(decodedMsg.AgentID, int32(record.InPort), int32(record.OutPort), record.Direction)
			if topics != nil { //找到了该条数据需要发送的topics
				for _, topic := range topics {
					if topicDataFlowSet[topic] == nil {
						topicDataFlowSet[topic] = make([]netflow9.DataFlowRecord, 0)
					}
					topicDataFlowSet[topic] = append(topicDataFlowSet[topic], record)
				}
			}
		}
	}

	for k := range topicDataFlowSet {
		buf := new(bytes.Buffer)
		decodedMsg.Header.Count = uint16(len(topicDataFlowSet[k]))
		b, err := decodedMsg.JSONMarshal(buf, topicDataFlowSet[k])
		if err != nil {
			vlogger.Logger.Println(err)
			continue
		}
		//k 为需要发送到的topic

		netflowV9MainMQChannel <- producer.MQMessage{Topic: k, Msg: string(b[:])}
		if utils.Opts.Verbose {
			vlogger.Logger.Println(string(b))
		}
	}
}

func (i *NetflowV9) status() *NetflowV9Stats {
	return &NetflowV9Stats{
		UDPQueue:     len(netflowV9UDPCh),
		MessageQueue: len(netflowV9MainMQChannel),
		UDPCount:     atomic.LoadUint64(&i.stats.UDPCount),
		DecodedCount: atomic.LoadUint64(&i.stats.DecodedCount),
		MQErrorCount: atomic.LoadUint64(&i.stats.MQErrorCount),
		Workers:      atomic.LoadInt32(&i.stats.Workers),
		StartTime:    atomic.LoadInt64(&i.stats.StartTime),
		LostCount:    atomic.LoadUint64(&i.stats.LostCount),
	}
}

func (i *NetflowV9) dynWorkers() {
	var load, nSeq, newWorkers, workers, n int
	tick := time.Tick(120 * time.Second)

	for {
		<-tick
		load = 0

		for n = 0; n < 30; n++ {
			time.Sleep(1 * time.Second)
			load += len(netflowV9UDPCh)
		}

		if load > 15 {

			switch {
			case load > 300:
				newWorkers = 100
			case load > 200:
				newWorkers = 60
			case load > 100:
				newWorkers = 40
			default:
				newWorkers = 30
			}

			workers = int(atomic.LoadInt32(&i.stats.Workers))
			if workers+newWorkers > utils.MaxWorkers {
				vlogger.Logger.Println("netflow v9 :: max out workers")
				continue
			}

			for n = 0; n < newWorkers; n++ {
				go func() {
					atomic.AddInt32(&i.stats.Workers, 1)
					wQuit := make(chan struct{})
					i.pool <- wQuit
					i.netflowV9Worker(wQuit)
				}()
			}

		}
		if load == 0 {
			nSeq++
		} else {
			nSeq = 0
			continue
		}

		if nSeq > 15 {
			for n = 0; n < 10; n++ {
				if len(i.pool) > i.workers {
					atomic.AddInt32(&i.stats.Workers, -1)
					wQuit := <-i.pool
					close(wQuit)
				}
			}

			nSeq = 0
		}
	}
}

func (i *NetflowV9) NetflowPacketLoss(agentId string) (int, error) {
	rtn, err := i.pktStat.getLost(agentId)
	if err != nil {
		return -1, err
	} else {
		return int(rtn + 0), nil
	}
}
