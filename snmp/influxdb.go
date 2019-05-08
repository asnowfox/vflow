package snmp

import (
	"github.com/VerizonDigital/vflow/vlogger"
	"github.com/coreos/etcd/client"
	"github.com/influxdata/influxdb/client/v2"
	"strconv"
	"time"
)

var (
	hostUrl  = "http://vflow-web:8086"
	dbName   = "flowMatrix"
	username = "admin"
	password = "vlfow"
)

func Init(db string, uname string, passwd string) {
	dbName = db
	username = uname
	password = passwd
}

func SaveWalkToInflux(curTime time.Time, deviceIp string, indexList []int, nameList []string,ifAlainMap map[int]string,
		ifInOctList []uint64, ifOutOctList []uint64, statusMap map[int]int,ifToNfIndexMap map[int]int) {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     hostUrl,
		Username: username,
		Password: password,
	})
	if err != nil {
		vlogger.Logger.Print(err)
	}
	defer c.Close()

	// Create a new point batch
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  dbName,
		Precision: "s",
	})
	if err != nil {
		vlogger.Logger.Print(err)
	}

	for i, index := range indexList {
		// Create a point and add to batch
		tags := map[string]string{"portIndex": strconv.Itoa(index),
			"ifDes": nameList[i], "ifAlian":ifAlainMap[index],
			"ofIndex": strconv.Itoa(ifToNfIndexMap[index]),
			"operStatus":strconv.Itoa(statusMap[index]),
			"allDes":strconv.Itoa(index)+"|"+nameList[i]+"|"+ ifAlainMap[index]+"|"+strconv.Itoa(ifToNfIndexMap[index])}
		fields := map[string]interface{}{
			"inOtc":  float64(ifInOctList[i]),
			"outOtc": float64(ifOutOctList[i]),
		}

		pt, err := client.NewPoint(deviceIp+"_snmp", tags, fields, curTime)

		if err != nil {
			vlogger.Logger.Printf("new point error " + err.Error())
		}

		bp.AddPoint(pt)
	}
	// Write the batch
	if err := c.Write(bp); err != nil {
		vlogger.Logger.Printf("write error " + err.Error())
	}else{
		vlogger.Logger.Printf("write data success ")
	}

	// Close client resources
	if err := c.Close(); err != nil {
		vlogger.Logger.Print(err)
	}
}
