package main

import (
	"fmt"

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

	config, err := webx.SimpleTLSConfig("cert.pem", "key.pem")
	if err != nil {
		fmt.Println(err)
		return
	}

	webx.RunTLS("0.0.0.0:9999", config)
}
