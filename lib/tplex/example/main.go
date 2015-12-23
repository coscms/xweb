package main

import (
	"fmt"
	"os"
	"time"

	"github.com/coscms/xweb/lib/tplex"
	"github.com/coscms/xweb/log"
)

func main() {
	tpl := tplex.New(log.New(os.Stdout, "", log.Ldefault()), "./template/", true)
	for i := 0; i < 5; i++ {
		ts := time.Now()
		fmt.Printf("==========%v: %v========\\\n", i, ts)
		str := tpl.Fetch("test.html", nil, map[string]string{
			"test": "one->" + fmt.Sprintf("%v", i),
		})
		fmt.Printf("%v\n", str)
		fmt.Printf("==========cost: %v========/\n", time.Now().Sub(ts).Seconds())
	}
}
