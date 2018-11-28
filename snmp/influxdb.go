package snmp

import (
	"github.com/influxdata/influxdb/client/v2"
	"../vlogger"
	"time"
	"fmt"
	"github.com/influxdata/platform/kit/errors"
	"strconv"
)

var (
	hostUrl = "http://vflow-web:8086"
	dbName   = "flowMatrix"
	username = "admin"
	password = "vlfow"
)

func Init(db string, uname string, passwd string) {
	dbName = db
	username = uname
	password = passwd
}

func SaveWalkToInflux(deviceIp string,indexList []int, nameList []string, ifInOctList []uint64, ifOutOctList []uint64) {
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
		tags := map[string]string{"portIndex":strconv.Itoa(index),"ifDes":nameList[i],}
		fields := map[string]interface{}{
			"inOtc": uint64(ifInOctList[i]),
			"outOtc": uint64(ifOutOctList[i]),
		}

		pt, err := client.NewPoint(deviceIp+"_snmp", tags, fields, time.Now())

		if err != nil {
			vlogger.Logger.Print("new point error "+err.Error())
		}

		bp.AddPoint(pt)
	}
	// Write the batch
	if err := c.Write(bp); err != nil {

		vlogger.Logger.Print("write error "+err.Error())
		fmt.Println(err.(*errors.Error))
	}

	// Close client resources
	if err := c.Close(); err != nil {
		vlogger.Logger.Print(err)
	}
}
