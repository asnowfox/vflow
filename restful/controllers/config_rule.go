package controllers

import (
	"github.com/astaxie/beego"
	"fmt"
	"../../mirror"
	"encoding/json"
)

// Operations about object
type RuleController struct {
	beego.Controller
	MirrorService mirror.Netflowv9Mirror
}


func (o *RuleController) Delete() {
	sourceId := o.GetString("sourceId")
	inport,_ := o.GetInt32("inport")
	outport,_ := o.GetInt32("outport")
	dstAddress := o.GetString("dstAddress")
	fmt.Printf("call delete method of mirror controller, sourceId is %s\r\n", sourceId)

	index := -1
	if sourceId != "" {
		index = mirror.DeleteRule(sourceId,mirror.Rule{sourceId,inport,outport,dstAddress})
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
func (o *RuleController) Post() {
	var ob mirror.Rule
	sourceId := o.GetString("sourceId")
	err := json.Unmarshal(o.Ctx.Input.RequestBody, &ob)
	json := map[string]interface{}{}
	if err != nil{
		json["result"] = -1
		json["message"] = "parse json error"
		o.Data["json"] = json
		o.ServeJSON()
		return
	}
	index,msg := mirror.AddRule(sourceId,ob)

	json["result"] = index
	json["message"] = msg
	o.Data["json"] = json

	o.ServeJSON()
}