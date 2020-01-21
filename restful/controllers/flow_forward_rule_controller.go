package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/VerizonDigital/vflow/mirror"
	"github.com/VerizonDigital/vflow/utils"
	"github.com/astaxie/beego"
	"strconv"
	"strings"
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
	var ob utils.Rule
	policyId := o.GetString("policyId")
	ruleId := o.GetString("ruleId")
	method := o.GetString("method")
	jsonRtn := map[string]interface{}{}
	if method == "add" {
		err := json.Unmarshal(o.Ctx.Input.RequestBody, &ob)
		if err != nil {
			jsonRtn["result"] = -1
			jsonRtn["id"] = ""
			jsonRtn["message"] = "parse json error"
			o.Data["json"] = jsonRtn
			o.ServeJSON()
			return
		}
		index, msg := mirror.AddRule(policyId, ob)
		id := strconv.Itoa(int(ob.Port)) + "_" + strconv.Itoa(int(ob.Direction)) + "_" + ob.Source
		if index < 0 {
			id = ""
		}
		jsonRtn["result"] = index
		jsonRtn["id"] = id
		jsonRtn["message"] = msg
		fmt.Printf("add result %s\r\n", jsonRtn)

		o.Data["json"] = jsonRtn
		o.ServeJSON()
	} else if method == "delete" {
		strs := strings.Split(ruleId, "_")
		if len(strs) != 3 {
			jsonRtn["result"] = -1
			jsonRtn["id"] = ruleId
			jsonRtn["message"] = "unknow id"
			o.Data["json"] = jsonRtn
			o.ServeJSON()
			return
		}
		port, e1 := strconv.Atoi(strs[0])
		direction, e2 := strconv.Atoi(strs[1])
		if e1 != nil && e2 != nil {
			jsonRtn["result"] = -1
			jsonRtn["id"] = ruleId
			jsonRtn["message"] = "Unknow rule id"
			o.Data["json"] = jsonRtn
			o.ServeJSON()
			return
		}
		rule := utils.Rule{
			Source:      strs[2],
			Port:        int32(port),
			Direction:   int(direction),
			DistAddress: make([]string, 0),
		}
		index, msg := mirror.DeleteRule(policyId, rule)
		jsonRtn := map[string]interface{}{}
		jsonRtn["id"] = ruleId
		jsonRtn["result"] = index
		jsonRtn["message"] = msg
		o.Data["json"] = jsonRtn
		fmt.Printf("result %s\r\n", o.Data)
		o.ServeJSON()
		return
	}
}
