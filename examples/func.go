package main

import (
	"fmt"

	"github.com/coscms/webx"
)

type MainAction struct {
	*webx.Action

	hello webx.Mapper `webx:"/(.*)"`
}

func (c *MainAction) Hello(world string) error {
	return c.RenderString(fmt.Sprintf("hello {{if isWorld}}%v{{else}}go{{end}}", world))
}

func (c *MainAction) IsWorld() bool {
	return true
}

func (c *MainAction) Init() {
	fmt.Println("init mainaction")
	c.Assign("isWorld", c.IsWorld)
}

func main() {
	webx.AddAction(&MainAction{})
	webx.Run("0.0.0.0:9999")
}
