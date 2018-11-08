package restful

import (
	"./routers"
	"github.com/astaxie/beego"
	"../vflow"
)

type BegooServer struct {

}
var(
	BeegoInstance *BegooServer
)

func NewBeegoServer(netflowv9 *flows.NetflowV9) (*BegooServer){
	BeegoInstance = &BegooServer{

	}
	routers.Init(netflowv9)
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
