//: ----------------------------------------------------------------------------
//: Copyright (C) 2017 Verizon.  All Rights Reserved.
//: All Rights Reserved
//:
//: file:    options.go
//: details: vFlow options :: configuration and command line
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

package utils

import (
	"flag"
	"fmt"
	"github.com/VerizonDigital/vflow/vlogger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
)

var (
	version    string
	MaxWorkers = runtime.NumCPU() * 1e4
)

type arrUInt32Flags []uint32

var Opts *Options

var MqEnabled = false

// Options represents options
type Options struct {
	// global options
	Verbose          bool   `yaml:"verbose"`
	LogFile          string `yaml:"log-file"`
	FlowForwardFile  string `yaml:"flow-forward-cfg"`
	QueueForwardFile string `yaml:"queue-forward-cfg"`
	CommunityFile    string `yaml:"device-community-cfg"`
	PIDFile          string `yaml:"pid-file"`
	CPUCap           string `yaml:"cpu-cap"`
	DynWorkers       bool   `yaml:"dynamic-workers"`
	//Logger      *log.Logger
	version bool

	// stats options
	StatsEnabled  bool   `yaml:"stats-enabled"`
	StatsHTTPAddr string `yaml:"stats-http-addr"`
	StatsHTTPPort string `yaml:"stats-http-port"`

	// sFlow options
	SFlowEnabled    bool           `yaml:"sflow-enabled"`
	SFlowPort       int            `yaml:"sflow-port"`
	SFlowUDPSize    int            `yaml:"sflow-udp-size"`
	SFlowWorkers    int            `yaml:"sflow-workers"`
	SFlowTopic      string         `yaml:"sflow-topic"`
	SFlowTypeFilter arrUInt32Flags `yaml:"sflow-type-filter"`

	// IPFIX options
	IPFIXEnabled       bool   `yaml:"ipfix-enabled"`
	IPFIXRPCEnabled    bool   `yaml:"ipfix-rpc-enabled"`
	IPFIXPort          int    `yaml:"ipfix-port"`
	IPFIXAddr          string `yaml:"ipfix-addr"`
	IPFIXUDPSize       int    `yaml:"ipfix-udp-size"`
	IPFIXWorkers       int    `yaml:"ipfix-workers"`
	IPFIXTopic         string `yaml:"ipfix-topic"`
	IPFIXMirrorAddr    string `yaml:"ipfix-mirror-addr"`
	IPFIXMirrorPort    int    `yaml:"ipfix-mirror-port"`
	IPFIXMirrorWorkers int    `yaml:"ipfix-mirror-workers"`
	IPFIXTplCacheFile  string `yaml:"ipfix-tpl-cache-file"`

	// Netflow
	NetflowV9Enabled      bool   `yaml:"netflow9-enabled"`
	NetflowV9Port         int    `yaml:"netflow9-port"`
	NetflowV9UDPSize      int    `yaml:"netflow9-udp-size"`
	NetflowV9Workers      int    `yaml:"netflow9-workers"`
	NetflowV9Topic        string `yaml:"netflow9-topic"`
	NetflowV9TplCacheFile string `yaml:"netflow9-tpl-cache-file"`

	// producer
	MQName       string `yaml:"mq-name"`
	MQConfigFile string `yaml:"mq-config-file"`

	VFlowConfigPath string
}

func init() {
	if version == "" {
		version = "unknown"
	}
}

func (a *arrUInt32Flags) String() string {
	return "SFlow Type string"
}

func (a *arrUInt32Flags) Set(value string) error {
	arr := strings.Split(value, ",")
	for _, v := range arr {
		v64, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return err
		}
		*a = append(*a, uint32(v64))
	}

	return nil
}

// NewOptions constructs new options
func newOptions() *Options {
	return &Options{
		Verbose:          false,
		version:          false,
		DynWorkers:       true,
		PIDFile:          "/var/run/vflow.pid",
		FlowForwardFile:  "/etc/vflow/flow-forward.conf",
		QueueForwardFile: "/etc/vflow/queue-forward.conf",
		CommunityFile:    "/etc/vflow/snmp.conf",
		CPUCap:           "100%",

		StatsEnabled:  true,
		StatsHTTPPort: "8082",
		StatsHTTPAddr: "",

		SFlowEnabled:    true,
		SFlowPort:       6343,
		SFlowUDPSize:    1500,
		SFlowWorkers:    200,
		SFlowTopic:      "vflow.sflow",
		SFlowTypeFilter: []uint32{},

		IPFIXEnabled:       true,
		IPFIXRPCEnabled:    true,
		IPFIXPort:          4739,
		IPFIXUDPSize:       1500,
		IPFIXWorkers:       200,
		IPFIXTopic:         "vflow.ipfix",
		IPFIXMirrorAddr:    "",
		IPFIXMirrorPort:    4172,
		IPFIXMirrorWorkers: 5,
		IPFIXTplCacheFile:  "/tmp/vflow.templates",

		NetflowV9Enabled:      true,
		NetflowV9Port:         4729,
		NetflowV9UDPSize:      1500,
		NetflowV9Workers:      200,
		NetflowV9Topic:        "vflow.netflow9",
		NetflowV9TplCacheFile: "/tmp/netflowv9.templates",

		MQName:       "none",
		MQConfigFile: "mq.conf",

		VFlowConfigPath: "/etc/vflow",
	}
}

// InitOptions gets options through cmd and file
func InitOptions() *Options {
	Opts = newOptions()

	Opts.vFlowFlagSet()
	Opts.vFlowVersion()

	if Opts.Verbose {
		vlogger.Logger.Printf("the full logging enabled")
		vlogger.Logger.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	if Opts.LogFile != "" {
		f, err := os.OpenFile(Opts.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			vlogger.Logger.Println(err)
		} else {
			vlogger.Logger.SetOutput(f)
		}
	}

	if ok := Opts.vFlowIsRunning(); ok {
		vlogger.Logger.Fatal("the vFlow already is running!")
	}

	Opts.vFlowPIDWrite()

	vlogger.Logger.Printf("Welcome to vFlow v.%s Apache License 2.0", version)
	vlogger.Logger.Printf("Copyright (C) 2018 Verizon. github.com/VerizonDigital/vflow")
	if Opts.MQName != "none" {
		MqEnabled = true
	}
	return Opts
}

func (Opts Options) vFlowPIDWrite() {
	f, err := os.OpenFile(Opts.PIDFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		vlogger.Logger.Println(err)
		return
	}

	_, err = fmt.Fprintf(f, "%d", os.Getpid())
	if err != nil {
		vlogger.Logger.Println(err)
	}
}

func (Opts Options) vFlowIsRunning() bool {
	b, err := ioutil.ReadFile(Opts.PIDFile)
	if err != nil {
		return false
	}

	cmd := exec.Command("kill", "-0", string(b))
	_, err = cmd.Output()
	if err != nil {
		return false
	}

	return true
}

func (Opts Options) vFlowVersion() {
	if Opts.version {
		fmt.Printf("vFlow version: %s\n", version)
		os.Exit(0)
	}
}

// GetCPU returns the number of the CPU
func (Opts Options) GetCPU() int {
	var (
		numCPU      int
		availCPU    = runtime.NumCPU()
		invalCPUErr = "the CPU percentage is invalid: it should be between 1-100"
		numCPUErr   = "the CPU number should be greater than zero!"
	)

	if strings.Contains(Opts.CPUCap, "%") {
		pctStr := strings.Trim(Opts.CPUCap, "%")

		pctInt, err := strconv.Atoi(pctStr)
		if err != nil {
			vlogger.Logger.Fatalf("invalid CPU cap")
		}

		if pctInt < 1 || pctInt > 100 {
			vlogger.Logger.Fatalf(invalCPUErr)
		}

		numCPU = int(float32(availCPU) * (float32(pctInt) / 100))
	} else {
		numInt, err := strconv.Atoi(Opts.CPUCap)
		if err != nil {
			vlogger.Logger.Fatalf("invalid CPU cap")
		}

		if numInt < 1 {
			vlogger.Logger.Fatalf(numCPUErr)
		}

		numCPU = numInt
	}

	if numCPU > availCPU {
		numCPU = availCPU
	}

	return numCPU
}

func (Opts *Options) vFlowFlagSet() {

	var config string
	flag.StringVar(&config, "config", "/etc/vflow/vflow.conf", "path to config file")

	vFlowLoadCfg(Opts)

	// global options
	flag.BoolVar(&Opts.Verbose, "verbose", Opts.Verbose, "enable/disable verbose logging")
	flag.BoolVar(&Opts.DynWorkers, "dynamic-workers", Opts.DynWorkers, "enable/disable dynamic workers")
	flag.BoolVar(&Opts.version, "version", Opts.version, "show version")
	flag.StringVar(&Opts.LogFile, "log-file", Opts.LogFile, "log file name")
	flag.StringVar(&Opts.PIDFile, "pid-file", Opts.PIDFile, "pid file name")
	flag.StringVar(&Opts.CPUCap, "cpu-cap", Opts.CPUCap, "Maximum amount of CPU [percent / number]")
	flag.StringVar(&Opts.FlowForwardFile, "flow-forward-cfg", Opts.FlowForwardFile, "netflow v9 forward config file")
	flag.StringVar(&Opts.QueueForwardFile, "queue-forward-cfg", Opts.QueueForwardFile, "netflow v9 kafka queue forward rule file")
	// stats options
	flag.BoolVar(&Opts.StatsEnabled, "stats-enabled", Opts.StatsEnabled, "enable/disable stats listener")
	flag.StringVar(&Opts.StatsHTTPPort, "stats-http-port", Opts.StatsHTTPPort, "stats port listener")
	flag.StringVar(&Opts.StatsHTTPAddr, "stats-http-addr", Opts.StatsHTTPAddr, "stats bind address listener")

	// sflow options
	flag.BoolVar(&Opts.SFlowEnabled, "sflow-enabled", Opts.SFlowEnabled, "enable/disable sflow listener")
	flag.IntVar(&Opts.SFlowPort, "sflow-port", Opts.SFlowPort, "sflow port number")
	flag.IntVar(&Opts.SFlowUDPSize, "sflow-max-udp-size", Opts.SFlowUDPSize, "sflow maximum UDP size")
	flag.IntVar(&Opts.SFlowWorkers, "sflow-workers", Opts.SFlowWorkers, "sflow workers number")
	flag.StringVar(&Opts.SFlowTopic, "sflow-topic", Opts.SFlowTopic, "sflow topic name")
	flag.Var(&Opts.SFlowTypeFilter, "sflow-type-filter", "sflow type filter")

	// ipfix options
	flag.BoolVar(&Opts.IPFIXEnabled, "ipfix-enabled", Opts.IPFIXEnabled, "enable/disable IPFIX listener")
	flag.BoolVar(&Opts.IPFIXRPCEnabled, "ipfix-rpc-enabled", Opts.IPFIXRPCEnabled, "enable/disable RPC IPFIX")
	flag.IntVar(&Opts.IPFIXPort, "ipfix-port", Opts.IPFIXPort, "IPFIX port number")
	flag.StringVar(&Opts.IPFIXAddr, "ipfix-addr", Opts.IPFIXAddr, "IPFIX IP address to bind to")
	flag.IntVar(&Opts.IPFIXUDPSize, "ipfix-max-udp-size", Opts.IPFIXUDPSize, "IPFIX maximum UDP size")
	flag.IntVar(&Opts.IPFIXWorkers, "ipfix-workers", Opts.IPFIXWorkers, "IPFIX workers number")
	flag.StringVar(&Opts.IPFIXTopic, "ipfix-topic", Opts.IPFIXTopic, "ipfix topic name")
	flag.StringVar(&Opts.IPFIXTplCacheFile, "ipfix-tpl-cache-file", Opts.IPFIXTplCacheFile, "IPFIX template cache file")
	flag.StringVar(&Opts.IPFIXMirrorAddr, "ipfix-mirror-addr", Opts.IPFIXMirrorAddr, "IPFIX mirror destination address")
	flag.IntVar(&Opts.IPFIXMirrorPort, "ipfix-mirror-port", Opts.IPFIXMirrorPort, "IPFIX mirror destination port number")
	flag.IntVar(&Opts.IPFIXMirrorWorkers, "ipfix-mirror-workers", Opts.IPFIXMirrorWorkers, "IPFIX mirror workers number")

	// netflow version 9
	flag.BoolVar(&Opts.NetflowV9Enabled, "netflow9-enabled", Opts.NetflowV9Enabled, "enable/disable netflow version 9 listener")
	flag.IntVar(&Opts.NetflowV9Port, "netflow9-port", Opts.NetflowV9Port, "Netflow Version 9 port number")
	flag.IntVar(&Opts.NetflowV9UDPSize, "netflow9-max-udp-size", Opts.NetflowV9UDPSize, "Netflow version 9 maximum UDP size")
	flag.IntVar(&Opts.NetflowV9Workers, "netflow9-workers", Opts.NetflowV9Workers, "Netflow version 9 workers number")
	flag.StringVar(&Opts.NetflowV9Topic, "netflow9-topic", Opts.NetflowV9Topic, "Netflow version 9 topic name")
	flag.StringVar(&Opts.NetflowV9TplCacheFile, "netflow9-tpl-cache-file", Opts.NetflowV9TplCacheFile, "Netflow version 9 template cache file")

	// producer options
	flag.StringVar(&Opts.MQName, "mqueue", Opts.MQName, "producer message queue name")
	flag.StringVar(&Opts.MQConfigFile, "mqueue-conf", Opts.MQConfigFile, "producer message queue configuration file")

	flag.Usage = func() {
		flag.PrintDefaults()
		_, _ = fmt.Fprintf(os.Stderr, `
    Example:
	# set workers
	vflow -sflow-workers 15 -ipfix-workers 20

	# set 3rd party ipfix collector
	vflow -ipfix-mirror-addr 192.168.1.10 -ipfix-mirror-port 4319

	# enaable verbose logging
	vflow -verbose=true

	# for more information
	https://github.com/VerizonDigital/vflow/blob/master/docs/config.md

    `)

	}

	flag.Parse()
}

func vFlowLoadCfg(Opts *Options) {
	var file = path.Join(Opts.VFlowConfigPath, "vflow.conf")

	for i, flagString := range os.Args {
		if flagString == "-config" {
			file = os.Args[i+1]
			Opts.VFlowConfigPath, _ = path.Split(file)
			break
		}
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		vlogger.Logger.Println(err)
		return
	}
	err = yaml.Unmarshal(b, Opts)
	if err != nil {
		vlogger.Logger.Println(err)
	}
}
