package xweb

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/howeyc/fsnotify"
)

/**
 * 默认模板函数
 * 除了这里定义的之外，还可以使用当前Action（即在方法中使用Render的Action）中定义的可导出的属性和方法（使用".属性"或".方法"来访问）
 * 另外还支持函数：
 * include      —— Include(tmplName string) interface{}
 * session      —— GetSession(key string) interface{}
 * cookie       —— Cookie(key string) string
 * XsrfFormHtml —— XsrfFormHtml() template.HTML
 * XsrfValue    —— XsrfValue() string
 * XsrfName     —— XsrfName() string
 * StaticUrl    —— StaticUrl(url string) string
 * 支持变量：
 * XwebVer      —— string
 */
var (
	DefaultFuncs template.FuncMap = template.FuncMap{
		"Now":         Now,
		"Eq":          Eq,
		"FormatDate":  FormatDate,
		"Add":         Add,
		"Subtract":    Subtract,
		"IsNil":       IsNil,
		"Url":         Url,
		"UrlFor":      UrlFor,
		"Html":        Html,
		"Js":          Js,
		"Css":         Css,
		"XsrfField":   XsrfName, //alias
		"HtmlAttr":    HtmlAttr,
		"ToHtmlAttrs": ToHtmlAttrs,
		"BuildUrl":    BuildUrl,
	}
	DefaultTemplateMgr *TemplateMgr = new(TemplateMgr)
)

func IsNil(a interface{}) bool {
	switch a.(type) {
	case nil:
		return true
	}
	return false
}

func Add(left interface{}, right interface{}) interface{} {
	var rleft, rright int64
	var fleft, fright float64
	var isInt bool = true
	switch left.(type) {
	case int:
		rleft = int64(left.(int))
	case int8:
		rleft = int64(left.(int8))
	case int16:
		rleft = int64(left.(int16))
	case int32:
		rleft = int64(left.(int32))
	case int64:
		rleft = left.(int64)
	case float32:
		fleft = float64(left.(float32))
		isInt = false
	case float64:
		fleft = left.(float64)
		isInt = false
	}

	switch right.(type) {
	case int:
		rright = int64(right.(int))
	case int8:
		rright = int64(right.(int8))
	case int16:
		rright = int64(right.(int16))
	case int32:
		rright = int64(right.(int32))
	case int64:
		rright = right.(int64)
	case float32:
		fright = float64(left.(float32))
		isInt = false
	case float64:
		fleft = left.(float64)
		isInt = false
	}

	var intSum int64 = rleft + rright

	if isInt {
		return intSum
	} else {
		return fleft + fright + float64(intSum)
	}
}

func Subtract(left interface{}, right interface{}) interface{} {
	var rleft, rright int64
	var fleft, fright float64
	var isInt bool = true
	switch left.(type) {
	case int:
		rleft = int64(left.(int))
	case int8:
		rleft = int64(left.(int8))
	case int16:
		rleft = int64(left.(int16))
	case int32:
		rleft = int64(left.(int32))
	case int64:
		rleft = left.(int64)
	case float32:
		fleft = float64(left.(float32))
		isInt = false
	case float64:
		fleft = left.(float64)
		isInt = false
	}

	switch right.(type) {
	case int:
		rright = int64(right.(int))
	case int8:
		rright = int64(right.(int8))
	case int16:
		rright = int64(right.(int16))
	case int32:
		rright = int64(right.(int32))
	case int64:
		rright = right.(int64)
	case float32:
		fright = float64(left.(float32))
		isInt = false
	case float64:
		fleft = left.(float64)
		isInt = false
	}

	if isInt {
		return rleft - rright
	} else {
		return fleft + float64(rleft) - (fright + float64(rright))
	}
}

func Now() time.Time {
	return time.Now()
}

func FormatDate(t time.Time, format string) string {
	return t.Format(format)
}

func Eq(left interface{}, right interface{}) bool {
	leftIsNil := (left == nil)
	rightIsNil := (right == nil)
	if leftIsNil || rightIsNil {
		if leftIsNil && rightIsNil {
			return true
		}
		return false
	}
	return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
}

func Html(raw string) template.HTML {
	return template.HTML(raw)
}

func HtmlAttr(raw string) template.HTMLAttr {
	return template.HTMLAttr(raw)
}

func ToHtmlAttrs(raw map[string]interface{}) (r map[template.HTMLAttr]interface{}) {
	r = make(map[template.HTMLAttr]interface{})
	for k, v := range raw {
		r[HtmlAttr(k)] = v
	}
	return
}

func Js(raw string) template.JS {
	return template.JS(raw)
}

func Css(raw string) template.CSS {
	return template.CSS(raw)
}

//Usage:Url("/user/login","appName","servName") or Url("/user/login","appName") or Url("/user/login") or UrlFor()
func Url(args ...string) string {
	var (
		route    string
		appName  string = "root"
		servName string = "main"
	)
	size := len(args)
	switch size {
	case 1:
		route = args[0]
	case 2:
		route = args[0]
		appName = args[1]
	case 3:
		route = args[0]
		appName = args[1]
		servName = args[2]
	}
	return BuildUrl(route, appName, servName, size)
}

func BuildUrl(route, appName, servName string, size int) string {
	var appUrl, url, prefix, suffix string
	if server, ok := Servers[servName]; ok {
		prefix = server.Config.UrlPrefix
		suffix = server.Config.UrlSuffix
		var appDomain string
		if domain, ok := server.App2Domain[appName]; ok {
			appUrl = "/"
			appDomain = domain
		} else if appPath, ok := server.AppsNamePath[appName]; ok {
			appUrl = appPath
		}
		if appDomain != "" {
			if strings.Contains(appDomain, "//") {
				url = appDomain
			} else {
				url = "http://" + appDomain
			}
		} else {
			url = server.Config.Url
		}
	}
	url = strings.TrimRight(url, "/") + "/"
	if size == 0 {
		return url
	}
	if appUrl != "/" {
		appUrl = strings.TrimLeft(appUrl, "/")
		if length := len(appUrl); length > 0 && appUrl[length-1] != '/' {
			appUrl = appUrl + "/"
		}
	} else {
		appUrl = ""
	}
	url += prefix + appUrl
	if route == "" {
		return url
	}
	if suffix != "" {
		parts := strings.SplitN(route, "?", 2)
		posIdx := strings.LastIndex(parts[0], "/") + 1
		isDir := posIdx == len(parts[0])
		if !isDir {
			if !strings.Contains(parts[0][posIdx:], ".") {
				if len(parts) == 2 {
					route = parts[0] + suffix + "?" + parts[1]
				} else {
					route = parts[0] + suffix
				}
			}
		}
	}
	url += strings.TrimLeft(route, "/")
	return url
}

//Usage:UrlFor("main:root:/user/login") or UrlFor("root:/user/login") or UrlFor("/user/login") or UrlFor()
//这里的main代表Server名称；root代表App名称；后面的内容为Action中方法函数所对应的网址
func UrlFor(args ...string) string {
	s := [3]string{"main", "root", ""}
	var u []string
	size := len(args)
	if size > 0 {
		u = strings.Split(args[0], ":")
	} else {
		u = []string{""}
	}
	switch len(u) {
	case 1:
		s[2] = u[0]
	case 2:
		s[1] = u[0]
		s[2] = u[1]
	default:
		s[0] = u[0]
		s[1] = u[1]
		s[2] = u[2]
	}
	return BuildUrl(s[2], s[1], s[0], size)
}

type TemplateMgr struct {
	Caches        map[string][]byte
	mutex         *sync.Mutex
	RootDir       string
	Ignores       map[string]bool
	IsReload      bool
	app           *App
	Preprocessor  func([]byte) []byte
	timerCallback func() bool
	TimerCallback func() bool
	initialized   bool
}

func (self *TemplateMgr) Moniter(rootDir string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	//fmt.Println("[xweb] TemplateMgr watcher is start.")
	defer watcher.Close()
	done := make(chan bool)
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				if ev == nil {
					break
				}
				if _, ok := self.Ignores[filepath.Base(ev.Name)]; ok {
					break
				}
				d, err := os.Stat(ev.Name)
				if err != nil {
					break
				}

				if ev.IsCreate() {
					if d.IsDir() {
						watcher.Watch(ev.Name)
					} else {
						tmpl := ev.Name[len(self.RootDir)+1:]
						content, err := ioutil.ReadFile(ev.Name)
						if err != nil {
							self.app.Errorf("loaded template %v failed: %v", tmpl, err)
							break
						}
						self.app.Infof("loaded template file %v success", tmpl)
						self.CacheTemplate(tmpl, content)
					}
				} else if ev.IsDelete() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						tmpl := ev.Name[len(self.RootDir)+1:]
						self.CacheDelete(tmpl)
					}
				} else if ev.IsModify() {
					if d.IsDir() {
					} else {
						tmpl := ev.Name[len(self.RootDir)+1:]
						content, err := ioutil.ReadFile(ev.Name)
						if err != nil {
							self.app.Errorf("reloaded template %v failed: %v", tmpl, err)
							break
						}

						self.CacheTemplate(tmpl, content)
						self.app.Infof("reloaded template %v success", tmpl)
					}
				} else if ev.IsRename() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						tmpl := ev.Name[len(self.RootDir)+1:]
						self.CacheDelete(tmpl)
					}
				}
			case err := <-watcher.Error:
				self.app.Error("error:", err)
			case <-time.After(time.Second * 2):
				if self.timerCallback != nil {
					if self.timerCallback() == false {
						close(done)
						return
					}
				}
				//fmt.Printf("TemplateMgr timer operation: %v.\n", time.Now())
			}
		}
	}()

	err = filepath.Walk(self.RootDir, func(f string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return watcher.Watch(f)
		}
		return nil
	})

	if err != nil {
		self.app.Error(err.Error())
		return err
	}

	<-done
	//fmt.Println("[xweb] TemplateMgr watcher is closed.")
	return nil
}

func (self *TemplateMgr) CacheAll(rootDir string) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	fmt.Print("Reading the contents of the template files, please wait... ")
	err := filepath.Walk(rootDir, func(f string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		tmpl := f[len(rootDir)+1:]
		tmpl = FixDirSeparator(tmpl)
		if _, ok := self.Ignores[filepath.Base(tmpl)]; !ok {
			fpath := filepath.Join(self.RootDir, tmpl)
			content, err := ioutil.ReadFile(fpath)
			if err != nil {
				self.app.Debugf("load template %s error: %v", fpath, err)
				return err
			}
			self.app.Debug("loaded template", fpath)
			self.Caches[tmpl] = content
		}
		return nil
	})
	fmt.Println("Complete.")
	return err
}

func (self *TemplateMgr) defaultTimerCallback() func() bool {
	return func() bool {
		if self.TimerCallback != nil {
			return self.TimerCallback()
		}
		//更改模板主题后，关闭当前监控，重新监控新目录
		if self.app.AppConfig.TemplateDir == self.RootDir {
			return true
		}
		self.Caches = make(map[string][]byte)
		self.Ignores = make(map[string]bool)
		self.RootDir = self.app.AppConfig.TemplateDir
		go self.Moniter(self.RootDir)
		return false
	}
}

func (self *TemplateMgr) Close() {
	self.TimerCallback = func() bool {
		self.Caches = make(map[string][]byte)
		self.Ignores = make(map[string]bool)
		self.TimerCallback = nil
		return false
	}
	self.initialized = false
}

func (self *TemplateMgr) Init(app *App, rootDir string, reload bool) error {
	if self.initialized {
		if rootDir == self.RootDir {
			return nil
		} else {
			self.TimerCallback = func() bool {
				self.Caches = make(map[string][]byte)
				self.Ignores = make(map[string]bool)
				self.TimerCallback = nil
				return false
			}
		}
	} else if !reload {
		self.TimerCallback = func() bool {
			self.TimerCallback = nil
			return false
		}
	}
	self.RootDir = rootDir
	self.Caches = make(map[string][]byte)
	self.Ignores = make(map[string]bool)
	self.mutex = &sync.Mutex{}
	self.app = app
	if dirExists(rootDir) {
		//self.CacheAll(rootDir)
		if reload {
			self.timerCallback = self.defaultTimerCallback()
			go self.Moniter(rootDir)
		}
	}

	if len(self.Ignores) == 0 {
		self.Ignores["*.tmp"] = false
	}
	self.initialized = true
	return nil
}

func (self *TemplateMgr) GetTemplate(tmpl string) ([]byte, error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if content, ok := self.Caches[tmpl]; ok {
		self.app.Debugf("load template %v from cache", tmpl)
		return content, nil
	}

	content, err := ioutil.ReadFile(filepath.Join(self.RootDir, tmpl))
	if err == nil {
		self.app.Debugf("load template %v from the file:", tmpl)
		self.Caches[tmpl] = content
	}
	return content, err
}

func (self *TemplateMgr) CacheTemplate(tmpl string, content []byte) {
	if self.Preprocessor != nil {
		content = self.Preprocessor(content)
	}
	self.mutex.Lock()
	defer self.mutex.Unlock()
	tmpl = FixDirSeparator(tmpl)
	self.app.Debugf("update template %v on cache", tmpl)
	self.Caches[tmpl] = content
	return
}

func (self *TemplateMgr) CacheDelete(tmpl string) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	tmpl = FixDirSeparator(tmpl)
	if _, ok := self.Caches[tmpl]; ok {
		self.app.Debugf("delete template %v from cache", tmpl)
		delete(self.Caches, tmpl)
	}
	return
}
