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
	app1 := webx.NewApp("/")
	app1.AddAction(&MainAction{})
	webx.AddApp(app1)

	app2 := webx.NewApp("/user/")
	app2.AddAction(&MainAction{})
	webx.AddApp(app2)

	webx.Run("0.0.0.0:9999")
}
