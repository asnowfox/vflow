package snmp

import (
	"github.com/alouca/gosnmp"
	"log"
)

type NameIndex struct {
	IfName  string
	IfIndex  string
}



func  WalkIndex(deviceAddress string,community string){
	s, err := gosnmp.NewGoSNMP(deviceAddress, community, gosnmp.Version2c, 5)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := s.Get(".1.3.6.1.2.1.1.1.0")
	if err == nil {
		for _, v := range resp.Variables {
			switch v.Type {
			case gosnmp.OctetString:
				log.Printf("Response: %s : %s : %s \n", v.Name, v.Value.(string), v.Type.String())
			}
		}
	}else{
		log.Printf("snmp walk err %e",err)
	}
}