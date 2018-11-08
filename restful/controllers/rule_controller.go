package controllers

import (
	"github.com/astaxie/beego"
	"../../mirror"
	"encoding/json"
)

// Operations about object
type RuleController struct {
	beego.Controller
	MirrorService mirror.Netflowv9Mirror
}

// @Title Create
// @Description create object
// @Param	body		body 	models.Object	true		"The object content"
// @Success 200 {string} models.Object.Id
// @Failure 403 body is empty
// @router / [post]
func (o *RuleController) Post() {
	var ob mirror.Rule
	policyId := o.GetString("policyId")
	method := o.GetString("method")
	err := json.Unmarshal(o.Ctx.Input.RequestBody, &ob)
	json := map[string]interface{}{}
	if err != nil{
		json["result"] = -1
		json["message"] = "parse json error"
		o.Data["json"] = json
		o.ServeJSON()
		return
	}
	if method == "add"{
		index,msg := mirror.AddRule(policyId,ob)
		json["result"] = index
		json["message"] = msg
		o.Data["json"] = json
		o.ServeJSON()
	}else if method == "delete"{
		index,msg := mirror.DeleteRule(policyId,ob)
		json["result"] = index
		json["message"] = msg
		o.Data["json"] = json
		o.ServeJSON()
	}
}