package main

import (
	"github.com/coscms/webx"
)

type MainAction struct {
	*webx.Action

	hello webx.Mapper `webx:"/(.*)"`
}

func (c *MainAction) Hello(world string) {
	c.Write("hello %v", world)
}

func main() {
	webx.RootApp().AppConfig.SessionOn = false
	webx.AddRouter("/", &MainAction{})
	webx.Run("0.0.0.0:9999")
}
