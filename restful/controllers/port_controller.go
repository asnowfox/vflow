package controllers

import (
	"fmt"
	"github.com/astaxie/beego"
	"../../snmp"
)

// Operations about object
type PortController struct {
	beego.Controller
}



func (o *PortController) Get() {
	deviceIp := o.GetString("deviceIp")
	fmt.Printf("Call get method of device controller")
	devConfigs := snmp.SnmpTaskInstance.ListPortInfo(deviceIp)
	o.Data["json"] = devConfigs
	o.ServeJSON()
	return
}
