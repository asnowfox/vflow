package controllers

import (
	"fmt"
	"github.com/astaxie/beego"
	"../../vflow"
)


// Operations about object
type StatsController struct {
	beego.Controller
	netflowv9 flows.NetflowV9
}

func (o *StatsController) InitService(netflowv9 flows.NetflowV9){
	o.netflowv9 = netflowv9
}
// @Title Get
// @Description find object by objectid
// @Param	objectId		path 	string	true		"the objectid you want to get"
// @Success 200 {object} models.Object
// @Failure 403 :objectId is empty
// @router /:objectId [get]
func (o *StatsController) Get() {
	agentId := o.GetString("agentId")
	fmt.Printf("call get packet loss. agentId is %s\r\n",agentId)
	if agentId != "" {
		loss,err := o.netflowv9.NetflowPacketLoss(agentId)

		json := map[string]interface{}{}
		json["loss"] = loss
		json["message"] = err
		o.Data["json"] = json
		o.ServeJSON()
	} else {

	}
	o.Data["json"] = map[string]interface{}{}
	o.ServeJSON()
	return
}
