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
	webx.AutoAction(&MainAction{})
	webx.Run("0.0.0.0:9999")
	//visit http://localhost:9999/main/world
}
