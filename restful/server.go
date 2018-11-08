package restful

import (
	_ "./routers"
	_ "./models"
	"github.com/astaxie/beego"
)

type BegooServer struct {

}
var(
	BeegoInstance *BegooServer
)

func NewBeegoServer() (*BegooServer){
	BeegoInstance = &BegooServer{

	}
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
