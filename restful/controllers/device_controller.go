package controllers

import (
	"encoding/json"
	"github.com/VerizonDigital/vflow/snmp"
	"github.com/VerizonDigital/vflow/vlogger"
	"github.com/astaxie/beego"
)

// Operations about object
type DeviceController struct {
	beego.Controller
}

func (o *DeviceController) Get() {
	devConfigs := snmp.ManageInstance.ListConfig()
	o.Data["json"] = devConfigs
	o.ServeJSON()
	return
}

func (o *DeviceController) Post() {
	method := o.GetString("method")
	cnt := -1
	if method == "add" {
		vlogger.Logger.Printf("call add method of device controller")
		var ob snmp.CommunityConfig
		err := json.Unmarshal(o.Ctx.Input.RequestBody, &ob)
		msg := "parse json error."
		if err == nil {
			cnt, msg = snmp.ManageInstance.AddConfig(ob)
		}
		json := map[string]interface{}{}
		json["result"] = cnt
		json["id"] = ob.DeviceAddress
		json["message"] = msg
		o.Data["json"] = json
		o.ServeJSON()
		return
	} else if method == "delete" {
		deviceIp := o.GetString("deviceIp")
		vlogger.Logger.Printf("call delete method of device controller, deviceIP is %s\r\n", deviceIp)
		cnt, msg := snmp.ManageInstance.DeleteConfig(deviceIp)
		json := map[string]interface{}{}
		json["result"] = cnt
		json["message"] = msg
		o.Data["json"] = json
		o.ServeJSON()
		return
	} else if method == "refresh" {
		deviceIp := o.GetString("deviceIp")
		vlogger.Logger.Printf("call refresh method of device controller, deviceIP is %s\r\n", deviceIp)
		ports, err := snmp.ManageInstance.RefreshConfig(deviceIp)
		if err != nil {
			json := map[string]interface{}{}
			json["ports"] = ports
			json["result"] = -1
			json["id"] = deviceIp
			json["message"] = err.Error()
			o.Data["json"] = json
			o.ServeJSON()
		} else {
			json := map[string]interface{}{}
			json["ports"] = ports
			json["message"] = "updated"
			json["id"] = deviceIp
			json["result"] = 1
			o.Data["json"] = json
			o.ServeJSON()
		}
		return
	} else {
		json := map[string]interface{}{}
		json["result"] = -1
		json["message"] = "Unknown method " + method
		o.Data["json"] = json
		o.ServeJSON()
		return
	}
}