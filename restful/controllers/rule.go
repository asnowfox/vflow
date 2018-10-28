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
	inport,_ := o.GetUint16("inport")
	outport,_ := o.GetUint16("outport")
	dstAddress := o.GetString("dstAddress")
	fmt.Printf("call delete method of mirror controller, sourceId is %s\r\n", sourceId)

	index := -1
	if sourceId != "" {
		index = mirror.Netflowv9Instance.DeleteRule(sourceId,mirror.Rule{inport,outport,dstAddress})
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
	json.Unmarshal(o.Ctx.Input.RequestBody, &ob)

	index := mirror.Netflowv9Instance.AddRule(sourceId,ob)
	json := map[string]interface{}{}
	json["result"] = index
	o.Data["json"] = json
	o.ServeJSON()
}