package controllers

import (
	"fmt"
	"github.com/VerizonDigital/vflow/snmp"
	"github.com/astaxie/beego"
	"sort"
)

// Operations about object
type PortController struct {
	beego.Controller
}

func (o *PortController) Get() {
	deviceIp := o.GetString("deviceIp")
	fmt.Printf("Call get method of device controller")
	ports := snmp.ManageInstance.ListPortInfo(deviceIp)
	sort.Slice(ports, func(i, j int) bool {
		return ports[i].IfIndex < ports[j].IfIndex
	})
	o.Data["json"] = ports
	o.ServeJSON()
	return
}
