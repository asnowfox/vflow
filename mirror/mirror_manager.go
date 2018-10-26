package mirror

import (
	"log"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"net"
	"fmt"
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
	InPort      int32  `yaml:"inport"`
	OutPort     int32  `yaml:"outport"`
	DistAddress string `yaml:"distAddress"`
}


func NewNetflowv9Mirror(mirrorCfg string, logger *log.Logger, mirrorInfIp string) (*Netflowv9Mirror, error) {
	ume := new(Netflowv9Mirror)
	ume.Logger = logger
	ume.mirrorCfgFile = mirrorCfg
	b, err := ioutil.ReadFile(mirrorCfg)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(b, &ume.mirrorConfigs)
	if err != nil {
		return ume, err
	}
	ume.initMap()
	fmt.Printf("Starting raw socket on interface %s....\n",mirrorInfIp)
	ume.rawSocket,_ = NewRawConn(net.ParseIP(mirrorInfIp))
	Netflowv9Instance = ume
	return ume, nil
}


type UdpClient struct {
	remoteAddress string
	port          string
	conn          *net.Conn
}

func (c *UdpClient) Send(b []byte) error {
	if c.conn == nil {
		c.openConn()
	}
	_, e := (*c.conn).Write(b)
	if e != nil {
		fmt.Printf("send error %s ",e)
		c.openConn()
	}
	return nil
}
func (c *UdpClient) openConn() error {
	conn, err := net.Dial("udp", c.remoteAddress+":"+c.port)
	if err != nil {
		return err
	}
	c.conn = &conn
	return nil
}

func (c *UdpClient) Close() error {
	(*c.conn).Close()
	return nil
}