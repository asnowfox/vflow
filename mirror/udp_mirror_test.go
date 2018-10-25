package mirror

import (
	"testing"
	"fmt"
	"time"
	"log"
	"os"
)

func TestReadConfig(t *testing.T) {
	nfv9Mirror,_ := NewNetflowv9Mirror("../scripts/nfv9.forward.conf",
		log.New(os.Stderr, "[vflow] ", log.Ldate|log.Ltime),)

	cfg := Config{}
	cfg.Source = "192.168.0.1"
	rule := Rule{}
	rule.DistAddress = "10.0.0.1:2222"
	rule.InPort = -1
	rule.OutPort = -1
	cfg.Rules = append(cfg.Rules, rule)
	nfv9Mirror.AddConfig(cfg)

	//nfv9Mirror.DeleteConfig("192.168.0.1")
	rule.OutPort = 1

	nfv9Mirror.AddRule("192.168.0.1",rule)
}

func TestSendUdp(t *testing.T) {
	um := NewUdpMirrorClient("localhost","4444")
	cnt := 100000
	curTime := time.Now().UnixNano()
	for i:=0;i<cnt;i++{
		s1 := "abdcdddd00df00000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000" +
			"000000000000000000000000\r\n"
		um.Send([]byte(s1))
	}
	costTime := time.Now().UnixNano() - curTime
	fmt.Printf("send %10d message cost %d ms, average speed is %15f packet/s\n",
		cnt ,costTime/1000/1000,float64(cnt)/float64(costTime)*1000*1000*1000)
}
