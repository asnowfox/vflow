package restful

import (
	_ "./routers"
	_ "./models"
	"github.com/astaxie/beego"
	"../vflow"
)

type BegooServer struct {

}
var(
	BeegoInstance *BegooServer
	NetflowInstance *vflow.NetflowV9
)

func NewBeegoServer(netflowv9 *vflow.NetflowV9) (*BegooServer){
	BeegoInstance = &BegooServer{

	}
	NetflowInstance = netflowv9
	return BeegoInstance
}

func (bs *BegooServer) Run(){
	go func(){
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
		beego.BConfig.CopyRequestBody = true
		beego.Run(":9999")
	}()
}
