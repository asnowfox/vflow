package controllers

import (
	"encoding/json"
	"github.com/VerizonDigital/vflow/producer"
	"github.com/VerizonDigital/vflow/restful/models"
	"github.com/astaxie/beego"
)

// Operations about object
type MQPolicyController struct {
	beego.Controller
	MirrorService producer.Producer
}

// @Title Get
// @Description find object by objectid
// @Param	objectId		path 	string	true		"the objectid you want to get"
// @Success 200 {object} models.Object
// @Failure 403 :objectId is empty
// @router /:objectId [get]
func (o *MQPolicyController) Get() {
	policyId := o.GetString("policyId")
	if policyId != "" {
		policy := producer.GetPolicyById(policyId)
		if policy == nil {
			o.Data["json"] = "{}"
			o.ServeJSON()
			return
		} else {
			o.Data["json"] = models.TransQueuePolicy(*policy)
			o.ServeJSON()
			return
		}
	} else {
		data := make([]models.QueuePolicyModel, 0)
		configs := producer.GetPolicies()
		for _, p := range configs {
			data = append(data, models.TransQueuePolicy(p))
		}
		o.Data["json"] = data
		o.ServeJSON()
		return
	}
}

// @Title Create
// @Description create object
// @Param	body		body 	models.Object	true		"The object content"
// @Success 200 {string} models.Object.Id
// @Failure 403 body is empty
// @router / [post]
func (o *MQPolicyController) Post() {
	method := o.GetString("method")
	if method == "add" {
		var ob producer.QueuePolicy
		err := json.Unmarshal(o.Ctx.Input.RequestBody, &ob)
		jsonMap := map[string]interface{}{}
		if err != nil {
			jsonMap["result"] = -1
			jsonMap["id"] = ""
			jsonMap["message"] = "parse json error"
			o.Data["json"] = jsonMap
			o.ServeJSON()
			return
		}
		index, msg := producer.AddQueuePolicy(ob)
		jsonMap["result"] = index
		jsonMap["id"] = ob.PolicyId
		jsonMap["message"] = msg
		o.Data["json"] = jsonMap
		o.ServeJSON()
		return
	} else if method == "delete" {
		policyId := o.GetString("policyId")
		index, msg := producer.DeleteQueuePolicy(policyId)
		jsonMap := map[string]interface{}{}
		jsonMap["result"] = index
		jsonMap["message"] = msg
		o.Data["json"] = jsonMap
		o.ServeJSON()
		return
	} else if method == "update" {
		policyId := o.GetString("policyId")
		var ob producer.QueuePolicy
		err := json.Unmarshal(o.Ctx.Input.RequestBody, &ob)
		jsonMap := map[string]interface{}{}
		if err != nil {
			jsonMap["result"] = -1
			jsonMap["id"] = ob.PolicyId
			jsonMap["message"] = "parse json error"
			o.Data["json"] = jsonMap
			o.ServeJSON()
			return
		}
		index, msg := producer.UpdateQueuePolicy(policyId, ob)
		jsonMap["result"] = index
		jsonMap["message"] = msg
		jsonMap["id"] = ob.PolicyId
		o.Data["json"] = jsonMap
		o.ServeJSON()
		return
	} else {
		jsonMap := map[string]interface{}{}
		jsonMap["result"] = -1
		o.Data["json"] = jsonMap
		jsonMap["id"] = ""
		jsonMap["message"] = "can not handle method " + method
		o.ServeJSON()
	}
}
