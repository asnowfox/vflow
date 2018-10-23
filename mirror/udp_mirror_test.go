package mirror

import (
	"testing"
	"fmt"
	"time"
)

func TestReadConfig(t *testing.T) {
	coll := new (UdpMirrorExchanger)

	err := coll.LoadCfgAndRun("C:/go_code/vflow/scripts/nfv9.forward.conf")
	if(err != nil){
		fmt.Printf("error is  %s\n", err)
	}

}

func TestSendUdp(t *testing.T) {
	um := NewMirror("localhost","4444")
	cnt := 100000
	curTime := time.Now().UnixNano()
	for i:=0;i<cnt;i++{
		s1 := "abdcdddd00df00000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000" +
			"000000000000000000000000\r\n"
		um.Send([]byte(s1))
	}
	costTime := time.Now().UnixNano() - curTime;
	fmt.Printf("send %10d message cost %d ms, average speed is %15f packet/s\n",
		cnt ,costTime/1000/1000,float64(cnt)/float64(costTime)*1000*1000*1000)
}
