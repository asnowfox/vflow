package restful

import (
	_ "./routers"
	_ "./models"
	"log"
	"github.com/astaxie/beego"
)

type BegooServer struct {
	logger *log.Logger

}
var(
	BeegoInstance *BegooServer
)

func NewBeegoServer(logger *log.Logger) (*BegooServer){
	BeegoInstance = &BegooServer{
		logger:logger,
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
