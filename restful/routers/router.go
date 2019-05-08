// @APIVersion 1.0.0
// @Title beego Test API
// @Description beego has a very cool tools to autogenerate documents for your API
// @Contact astaxie@gmail.com
// @TermsOfServiceUrl http://beego.me/
// @License Apache 2.0
// @LicenseUrl http://www.apache.org/licenses/LICENSE-2.0.html
package routers

import (
	"fmt"
	"github.com/VerizonDigital/vflow/flows"
	"github.com/VerizonDigital/vflow/restful/controllers"
	"github.com/astaxie/beego"
)

func Init(netflowv9 *flows.NetflowV9) {
	fmt.Printf("router ininted.\n")
	beego.Router("/user", &controllers.UserController{})
	beego.Router("/policy", &controllers.PolicyController{})
	beego.Router("/rule", &controllers.RuleController{})
	beego.Router("/device", &controllers.DeviceController{})
	beego.Router("/port", &controllers.PortController{})

	controller := &controllers.StatsController{}
	controller.InitService(*netflowv9)
	beego.Router("/stats",controller)
}
