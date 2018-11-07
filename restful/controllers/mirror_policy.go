package controllers

import (
	"github.com/astaxie/beego"
	"../../mirror"
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
	policies := mirror.GetPolicies()
	policyId := o.GetString("policyId")
	fmt.Printf("call get method of policy policyId is %s\r\n", policyId)
	if policyId != "" {
		for _, e := range policies {
			if e.PolicyId == policyId {
				o.Data["json"] = e
				o.ServeJSON()
				return
			}
		}
	} else {
		configs := mirror.GetPolicies()
		fmt.Printf("serve all configs\r\n")
		o.Data["json"] = configs
		o.ServeJSON()
		return
	}
	o.Data["json"] = map[string]interface{}{}
	o.ServeJSON()
	return
}

func (o *PolicyController) Delete() {
	policyId := o.GetString("policyId")
	fmt.Printf("call delete method of mirror controller, sourceId is %s\r\n", policyId)

	index := -1
	if policyId != "" {
		index = mirror.DeletePolicy(policyId)
	}
	json := map[string]interface{}{}
	json["result"] = index
	o.Data["json"] = json
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
	var ob mirror.Policy

	fmt.Printf("add post message is %s, bytes length is %d.\n",
		string(o.Ctx.Input.RequestBody), len(o.Ctx.Input.RequestBody))
	err := json.Unmarshal(o.Ctx.Input.RequestBody, &ob)
	json := map[string]interface{}{}
	if err != nil {
		json["result"] = -1
		json["message"] = "parse json error"
		o.Data["json"] = json
		o.ServeJSON()
		return
	}
	index,msg:=mirror.AddPolicy(ob)

	json["result"] = index
	json["message"] = msg
	o.Data["json"] = json
	o.ServeJSON()
}