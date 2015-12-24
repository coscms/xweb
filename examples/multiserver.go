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
	mc := &MainAction{}

	server1 := webx.NewServer()
	server1.AddRouter("/", mc)
	go server1.Run("0.0.0.0:9999")

	server2 := webx.NewServer()
	server2.AddRouter("/", mc)
	go server2.Run("0.0.0.0:8999")

	<-make(chan int)
}
