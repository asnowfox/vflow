package controllers

import (
	"github.com/astaxie/beego"
	"../../mirror"
	"fmt"
	"encoding/json"
)

// Operations about object
type MirrorController struct {
	beego.Controller
	MirrorService mirror.Netflowv9Mirror
}

// @Title Get
// @Description find object by objectid
// @Param	objectId		path 	string	true		"the objectid you want to get"
// @Success 200 {object} models.Object
// @Failure 403 :objectId is empty
// @router /:objectId [get]
func (o *MirrorController) Get() {
	configs := mirror.Netflowv9Instance.GetConfig()
	sourceId := o.GetString("sourceId")
	fmt.Printf("call get method of mirror sourceId is %s\r\n", sourceId)
	if sourceId != "" {
		for _, e := range configs {
			if e.Source == sourceId {
				o.Data["json"] = e
				o.ServeJSON()
				return
			}
		}
	} else {
		configs := mirror.Netflowv9Instance.GetConfig()
		fmt.Printf("serve all configs\r\n")
		o.Data["json"] = configs
		o.ServeJSON()
		return
	}
	o.Data["json"] = map[string]interface{}{}
	o.ServeJSON()
	return
}

func (o *MirrorController) Delete() {
	sourceId := o.GetString("sourceId")
	fmt.Printf("call delete method of mirror controller, sourceId is %s\r\n", sourceId)

	index := -1
	if sourceId != "" {
		index = mirror.Netflowv9Instance.DeleteConfig(sourceId)
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
func (o *MirrorController) Post() {
	var ob mirror.Config

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
	index,msg:=mirror.Netflowv9Instance.AddConfig(ob)

	json["result"] = index
	json["message"] = msg
	o.Data["json"] = json
	o.ServeJSON()
}