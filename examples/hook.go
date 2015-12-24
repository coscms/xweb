package main

import (
	"fmt"
	"time"

	"github.com/coscms/webx"
)

type MainAction struct {
	*webx.Action

	start time.Time

	hello webx.Mapper `webx:"/(.*)"`
}

func (c *MainAction) Hello(world string) {
	c.Write("hello %v", world)
}

func (c *MainAction) Before(structName, actionName string) bool {
	c.start = time.Now()
	fmt.Println("before", c.start)
	return true
}

func (c *MainAction) After(structName, actionName string, actionResult interface{}) bool {
	fmt.Println("after", time.Now().Sub(c.start))
	return true
}

func main() {
	webx.AddRouter("/", &MainAction{})
	webx.Run("0.0.0.0:9999")
}
