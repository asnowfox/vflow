package controllers

import (
	"fmt"
	"github.com/VerizonDigital/vflow/snmp"
	"github.com/astaxie/beego"
)

// Operations about object
type PortController struct {
	beego.Controller
}



func (o *PortController) Get() {
	deviceIp := o.GetString("deviceIp")
	fmt.Printf("Call get method of device controller")
	devConfigs := snmp.ManageInstance.ListPortInfo(deviceIp)
	o.Data["json"] = devConfigs
	o.ServeJSON()
	return
}
