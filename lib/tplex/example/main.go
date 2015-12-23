package main

import (
	"fmt"
	"os"

	"github.com/coscms/xweb/lib/tplex"
	"github.com/coscms/xweb/log"
)

func main() {
	tpl := tplex.New(log.New(os.Stdout, "", log.Ldefault()), "./template/", true)
	str := tpl.Fetch("test.html", nil, map[string]string{
		"test": "one",
	})
	fmt.Printf("%v", str)
}
