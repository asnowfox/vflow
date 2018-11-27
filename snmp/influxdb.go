package snmp

import (
	"github.com/influxdata/influxdb/client/v2"
	"log"
	"time"
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
		log.Fatal(err)
	}
	defer c.Close()

	// Create a new point batch
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  dbName,
		Precision: "s",
	})
	if err != nil {
		log.Fatal(err)
	}

	for i, index := range indexList {

		// Create a point and add to batch
		tags := map[string]string{"portIndex": string(index),"ifDes":nameList[i]}
		fields := map[string]interface{}{
			"inOtc": ifInOctList[i],
			"outOtc": ifOutOctList[i],
		}

		pt, err := client.NewPoint(deviceIp, tags, fields, time.Now())
		if err != nil {
			log.Fatal(err)
		}
		bp.AddPoint(pt)

		// Write the batch
		if err := c.Write(bp); err != nil {
			log.Fatal(err)
		}

		// Close client resources
		if err := c.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
