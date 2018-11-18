package controllers

import (
	"github.com/astaxie/beego"
	"../../mirror"
	"../models"
	"fmt"
	"encoding/json"
)

// Operations about object
type PolicyController struct {
	beego.Controller
	MirrorService mirror.Netflowv9Mirror
}

// @Title Get
// @Description find object by objectid
// @Param	objectId		path 	string	true		"the objectid you want to get"
// @Success 200 {object} models.Object
// @Failure 403 :objectId is empty
// @router /:objectId [get]
func (o *PolicyController) Get() {
	policyId := o.GetString("policyId")
	fmt.Printf("call get method of policy policyId is %s\r\n", policyId)
	if policyId != "" {
		policy := mirror.GetPolicyById(policyId)
		if policy == nil {
			o.Data["json"] = "{}"
			o.ServeJSON()
			return
		} else {
			o.Data["json"] = models.TransPolicy(*policy)
			o.ServeJSON()
			return
		}
	} else {
		data := make([]models.RPolicy,0)
		configs := mirror.GetPolicies()
		for _,p := range configs {
			data = append(data, models.TransPolicy(p))
		}
		fmt.Printf("serve all configs\r\n")
		o.Data["json"] = data
		o.ServeJSON()
		return
	}
	o.Data["json"] = map[string]interface{}{}
	o.ServeJSON()
	return
}

// @Title Create
// @Description create object
// @Param	body		body 	models.Object	true		"The object content"
// @Success 200 {string} models.Object.Id
// @Failure 403 body is empty
// @router / [post]
func (o *PolicyController) Post() {
	method := o.GetString("method")
	if method == "add" {
		var ob mirror.Policy
		err := json.Unmarshal(o.Ctx.Input.RequestBody, &ob)
		json := map[string]interface{}{}
		if err != nil {
			json["result"] = -1
			json["id"] = ""
			json["message"] = "parse json error"
			o.Data["json"] = json
			o.ServeJSON()
			return
		}
		index, msg := mirror.AddPolicy(ob)
		json["result"] = index
		json["id"] = ob.PolicyId
		json["message"] = msg
		o.Data["json"] = json
		o.ServeJSON()
		return
	} else if method == "delete" {
		policyId := o.GetString("policyId")
		index, msg := mirror.DeletePolicy(policyId)
		json := map[string]interface{}{}
		json["result"] = index
		json["message"] = msg
		o.Data["json"] = json
		o.ServeJSON()
		return
	} else if method == "update" {
		policyId := o.GetString("policyId")
		var ob mirror.Policy
		err := json.Unmarshal(o.Ctx.Input.RequestBody, &ob)
		json := map[string]interface{}{}
		if err != nil {
			json["result"] = -1
			json["id"] = ob.PolicyId
			json["message"] = "parse json error"
			o.Data["json"] = json
			o.ServeJSON()
			return
		}
		index, msg := mirror.UpdatePolicy(policyId, ob)
		json["result"] = index
		json["message"] = msg
		json["id"] = ob.PolicyId
		o.Data["json"] = json
		o.ServeJSON()
		return
	} else {
		json := map[string]interface{}{}
		json["result"] = -1
		o.Data["json"] = json
		json["id"] = ""
		json["message"] = "can not handle method " + method
		o.ServeJSON()
	}
}
