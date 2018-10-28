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

func NewBeegoServer(logger *log.Logger) (*BegooServer){
	return &BegooServer{
		logger:logger,
	}
}

func (bs *BegooServer) Run(){
	go func(){
		if beego.BConfig.RunMode == "dev" {
			beego.BConfig.WebConfig.DirectoryIndex = true
			beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
			beego.BConfig.CopyRequestBody = true
		}
		beego.Run(":9999")
	}()
}
