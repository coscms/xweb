package tplex

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
)

func dump(i interface{}) {
	c, _ := json.MarshalIndent(i, "", " ")
	fmt.Println(string(c))
}

func FixDirSeparator(dir string) string {
	if runtime.GOOS == "windows" {
		return strings.Replace(dir, "\\", "/", -1)
	}
	return dir
}

func dirExists(dir string) bool {
	d, e := os.Stat(dir)
	switch {
	case e != nil:
		return false
	case !d.IsDir():
		return false
	}

	return true
}

func fileExists(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}

	return !info.IsDir()
}
