package main

import (
	"fmt"

	"github.com/coscms/webx"
)

type MainAction struct {
	*webx.Action

	hello webx.Mapper `webx:"/(.*)"`
}

var content string = `
	base path is {{.Basepath}}
`

func (c *MainAction) Basepath() string {
	return c.App.BasePath
}

func (c *MainAction) Hello(world string) {
	err := c.RenderString(content)
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	webx.AddAction(&MainAction{})
	webx.Run("0.0.0.0:9999")
}
