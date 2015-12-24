package main

import (
	"os"

	"github.com/coscms/webx"
	"github.com/coscms/webx/lib/log"
)

type MainAction struct {
	*webx.Action

	hello webx.Mapper `webx:"/(.*)"`
}

func (c *MainAction) Hello(world string) {
	c.Write("hello %v", world)
}

func main() {
	f, err := os.Create("server.log")
	if err != nil {
		println(err.Error())
		return
	}
	logger := log.New(f, "", log.Ldate|log.Ltime)

	webx.AddAction(&MainAction{})
	webx.SetLogger(logger)
	webx.Run("0.0.0.0:9999")
}
