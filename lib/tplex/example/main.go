package main

import (
	"fmt"
	"os"
	"time"

	"github.com/coscms/webx/lib/log"
	"github.com/coscms/webx/lib/tplex"
)

func main() {
	tpl := tplex.New(log.New(os.Stdout, "", log.Ldefault()), "./template/", true)
	for i := 0; i < 5; i++ {
		ts := time.Now()
		fmt.Printf("==========%v: %v========\\\n", i, ts)
		str := tpl.Fetch("test", nil, map[string]interface{}{
			"test": "one---" + fmt.Sprintf("%v", i),
			"r":    []string{"one", "two", "three"},
		})
		fmt.Printf("%v\n", str)
		fmt.Printf("==========cost: %v========/\n", time.Now().Sub(ts).Seconds())
	}
}
