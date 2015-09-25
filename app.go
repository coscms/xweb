package xweb

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/coscms/tagfast"
	"github.com/coscms/xweb/httpsession"
	"github.com/coscms/xweb/log"
)

var (
	mapperType = reflect.TypeOf(Mapper{})
)

type JSON struct {
	Data interface{}
}

type JSONP struct {
	Data     interface{}
	Callback string
}

type XML struct {
	Data interface{}
}

type FILE struct {
	Data string
}

const (
	Debug = iota + 1
	Product
	XSRF_TAG string = "_xsrf"
)

type App struct {
	BasePath           string
	Name               string
	Domain             string
	Routes             []Route
	RoutesEq           map[string]map[string]Route //r["/example"]["POST"]
	filters            []Filter
	Server             *Server
	AppConfig          *AppConfig
	Config             *CONF
	Actions            map[string]interface{}
	ActionsPath        map[reflect.Type]string
	ActionsNamePath    map[string]string
	ActionsMethodRoute map[string]map[string]string
	FuncMaps           template.FuncMap
	Logger             *log.Logger
	VarMaps            T
	SessionManager     *httpsession.Manager //Session manager
	RootTemplate       *template.Template
	ErrorTemplate      *template.Template
	StaticVerMgr       *StaticVerMgr
	TemplateMgr        *TemplateMgr
	ContentEncoding    string
	RequestTime        time.Time
	Cryptor
	XsrfManager
}

type AppConfig struct {
	Mode              int
	StaticDir         string
	TemplateDir       string
	SessionOn         bool
	MaxUploadSize     int64
	CookieSecret      string
	CookieLimitIP     bool
	CookieLimitUA     bool
	CookiePrefix      string
	CookieDomain      string
	StaticFileVersion bool
	CacheTemplates    bool
	ReloadTemplates   bool
	CheckXsrf         bool
	SessionTimeout    time.Duration
	FormMapToStruct   bool
	EnableHttpCache   bool
	AuthBasedOnCookie bool
}

type Route struct {
	Path           string          //path string
	CompiledRegexp *regexp.Regexp  //path regexp
	HttpMethods    map[string]bool //GET POST HEAD DELETE etc.
	HandlerMethod  string          //struct method name
	HandlerElement reflect.Type    //handler element
}

func NewApp(path string, name string) *App {
	return &App{
		BasePath: path,
		Name:     name,
		RoutesEq: make(map[string]map[string]Route),
		AppConfig: &AppConfig{
			Mode:              Product,
			StaticDir:         "static",
			TemplateDir:       "templates",
			SessionOn:         true,
			SessionTimeout:    3600,
			MaxUploadSize:     10 * 1024 * 1024,
			StaticFileVersion: true,
			CacheTemplates:    true,
			ReloadTemplates:   true,
			CheckXsrf:         true,
			FormMapToStruct:   true,
		},
		Config: &CONF{
			Bool:      make(map[string]bool),
			Interface: make(map[string]interface{}),
			String:    make(map[string]string),
			Int:       make(map[string]int64),
			Float:     make(map[string]float64),
			Byte:      make(map[string][]byte),
			Conf:      make(map[string]*CONF),
		},
		Actions:            map[string]interface{}{},
		ActionsPath:        map[reflect.Type]string{},
		ActionsNamePath:    map[string]string{},
		ActionsMethodRoute: make(map[string]map[string]string),
		FuncMaps:           DefaultFuncs,
		VarMaps:            T{},
		filters:            make([]Filter, 0),
		StaticVerMgr:       DefaultStaticVerMgr,
		TemplateMgr:        DefaultTemplateMgr,
		Cryptor:            DefaultCryptor,
		XsrfManager:        DefaultXsrfManager,
	}
}

func (a *App) IsRootApp() bool {
	return a.BasePath == "/"
}

func (a *App) initApp() {
	var isRootApp bool = a.IsRootApp()
	if a.AppConfig.StaticFileVersion {
		if isRootApp || a.Server.RootApp.AppConfig.StaticDir != a.AppConfig.StaticDir {
			if !isRootApp {
				a.StaticVerMgr = new(StaticVerMgr)
			}
			a.StaticVerMgr.Init(a, a.AppConfig.StaticDir)
		} else {
			a.StaticVerMgr = a.Server.RootApp.StaticVerMgr
		}
	}
	if a.AppConfig.CacheTemplates {
		if isRootApp || a.Server.RootApp.AppConfig.TemplateDir != a.AppConfig.TemplateDir {
			if !isRootApp {
				a.TemplateMgr = new(TemplateMgr)
			}
			a.TemplateMgr.Init(a, a.AppConfig.TemplateDir, a.AppConfig.ReloadTemplates)
		} else {
			a.TemplateMgr = a.Server.RootApp.TemplateMgr
		}
	}
	a.FuncMaps["StaticUrl"] = a.StaticUrl
	a.FuncMaps["XsrfName"] = XsrfName
	a.VarMaps["XwebVer"] = Version

	if a.AppConfig.SessionOn {
		if a.Server.SessionManager != nil {
			a.SessionManager = a.Server.SessionManager
		} else {
			a.SessionManager = httpsession.Default()
			if a.AppConfig.SessionTimeout > time.Second {
				a.SessionManager.SetMaxAge(a.AppConfig.SessionTimeout)
			}
			a.SessionManager.Run()
		}
	}

	if a.Logger == nil {
		a.Logger = a.Server.Logger
	}
}

func (a *App) DelDomain() {
	a.Domain = ""
	if domain, ok := a.Server.App2Domain[a.Name]; ok {
		delete(a.Server.App2Domain, a.Name)
		delete(a.Server.Domain2App, domain)
	}
}

func (a *App) SetDomain(domain string) {
	a.Domain = domain
	a.Server.App2Domain[a.Name] = domain
	a.Server.Domain2App[domain] = a.Name
}

func (a *App) SetStaticDir(dir string) {
	a.AppConfig.StaticDir = dir
}

func (a *App) SetTemplateDir(path string) {
	a.AppConfig.TemplateDir = path
}

func (a *App) getTemplatePath(name string) string {
	templateFile := path.Join(a.AppConfig.TemplateDir, name)
	if fileExists(templateFile) {
		return templateFile
	}
	return ""
}

func (app *App) SetConfig(name string, val interface{}) {
	app.Config.SetInterface(name, val)
}

func (app *App) GetConfig(name string) interface{} {
	return app.Config.GetInterface(name)
}

func (app *App) SetConfigString(name string, val string) {
	app.Config.SetString(name, val)
}

func (app *App) GetConfigString(name string) string {
	return app.Config.GetString(name)
}

func (app *App) AddAction(cs ...interface{}) {
	for _, c := range cs {
		app.AddRouter("/", c)
	}
}

func (app *App) AutoAction(cs ...interface{}) {
	for _, c := range cs {
		t := reflect.Indirect(reflect.ValueOf(c)).Type()
		name := t.Name()
		if strings.HasSuffix(name, "Action") {
			path := strings.ToLower(name[:len(name)-6])
			app.AddRouter("/"+path, c)
		} else {
			app.Warn("AutoAction needs a named ends with Action")
		}
	}
}

func (app *App) Assign(name string, varOrFun interface{}) {
	if reflect.TypeOf(varOrFun).Kind() == reflect.Func {
		app.FuncMaps[name] = varOrFun
	} else {
		app.VarMaps[name] = varOrFun
	}
}

func (app *App) MultiAssign(t *T) {
	for name, value := range *t {
		app.Assign(name, value)
	}
}

func (app *App) AddFilter(filter Filter) {
	app.filters = append(app.filters, filter)
}

func (app *App) Debug(params ...interface{}) {
	args := append([]interface{}{"[" + app.Name + "]"}, params...)
	app.Logger.Debug(args...)
}

func (app *App) Info(params ...interface{}) {
	args := append([]interface{}{"[" + app.Name + "]"}, params...)
	app.Logger.Info(args...)
}

func (app *App) Warn(params ...interface{}) {
	args := append([]interface{}{"[" + app.Name + "]"}, params...)
	app.Logger.Warn(args...)
}

func (app *App) Error(params ...interface{}) {
	args := append([]interface{}{"[" + app.Name + "]"}, params...)
	app.Logger.Error(args...)
}

func (app *App) Fatal(params ...interface{}) {
	args := append([]interface{}{"[" + app.Name + "]"}, params...)
	app.Logger.Fatal(args...)
}

func (app *App) Panic(params ...interface{}) {
	args := append([]interface{}{"[" + app.Name + "]"}, params...)
	app.Logger.Panic(args...)
}

func (app *App) Debugf(format string, params ...interface{}) {
	app.Logger.Debugf("["+app.Name+"] "+format, params...)
}

func (app *App) Infof(format string, params ...interface{}) {
	app.Logger.Infof("["+app.Name+"] "+format, params...)
}

func (app *App) Warnf(format string, params ...interface{}) {
	app.Logger.Warnf("["+app.Name+"] "+format, params...)
}

func (app *App) Errorf(format string, params ...interface{}) {
	app.Logger.Errorf("["+app.Name+"] "+format, params...)
}

func (app *App) Fatalf(format string, params ...interface{}) {
	app.Logger.Fatalf("["+app.Name+"] "+format, params...)
}

func (app *App) Panicf(format string, params ...interface{}) {
	app.Logger.Panicf("["+app.Name+"] "+format, params...)
}

func (app *App) filter(w http.ResponseWriter, req *http.Request) bool {
	for _, filter := range app.filters {
		if !filter.Do(w, req) {
			return false
		}
	}
	return true
}

/**
 * r		: 	网址
 * methods	:   HTTP访问方式（GET、POST...）
 * t		:	控制器实例反射
 * handler	:	控制器中的方法函数名
 */
func (a *App) addRoute(r string, methods map[string]bool, t reflect.Type, handler string) {
	cr, err := regexp.Compile(r)
	if err != nil {
		a.Errorf("Error in route regex %q: %s", r, err)
		return
	}
	a.Routes = append(a.Routes, Route{Path: r, CompiledRegexp: cr, HttpMethods: methods, HandlerMethod: handler, HandlerElement: t})
}

func (a *App) addEqRoute(r string, methods map[string]bool, t reflect.Type, handler string) {
	if _, ok := a.RoutesEq[r]; !ok {
		a.RoutesEq[r] = make(map[string]Route)
	}
	for v, _ := range methods {
		a.RoutesEq[r][v] = Route{HandlerMethod: handler, HandlerElement: t}
	}
}

func (app *App) AddRouter(url string, c interface{}) {
	t := reflect.TypeOf(c).Elem()
	v := reflect.ValueOf(c)
	actionFullName := t.Name()
	actionShortName := strings.TrimSuffix(actionFullName, "Action")
	actionShortName = strings.ToLower(actionShortName)
	app.ActionsPath[t] = url
	app.Actions[actionFullName] = c
	app.ActionsNamePath[actionFullName] = url
	app.ActionsMethodRoute[actionFullName] = make(map[string]string)

	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Type != mapperType {
			continue
		}
		name := t.Field(i).Name
		a := strings.Title(name)
		m := v.MethodByName(a)
		if !m.IsValid() {
			continue
		}

		tag := t.Field(i).Tag
		tagStr := tag.Get("xweb")
		methods := map[string]bool{} //map[string]bool{"GET": true, "POST": true}
		var p string
		var isEq bool
		if tagStr != "" {
			tags := strings.Split(tagStr, " ")
			path := tagStr
			length := len(tags)
			if length >= 2 { //`xweb:"GET|POST /index"`
				for _, method := range strings.Split(tags[0], "|") {
					method = strings.ToUpper(method)
					methods[method] = true
				}
				path = tags[1]
				if regexp.QuoteMeta(path) == path {
					isEq = true
				}
				if path == "" {
					path = name
				}
				if tags[1][0] != '/' {
					path = "/" + actionShortName + "/" + path
				}
			} else if length == 1 {
				if matched, _ := regexp.MatchString(`^[A-Z]+(\|[A-Z]+)*$`, tags[0]); !matched {
					//非全大写字母时，判断为网址规则
					path = tags[0]
					if regexp.QuoteMeta(path) == path {
						isEq = true
					}
					if tags[0][0] != '/' { //`xweb:"index"`
						path = "/" + actionShortName + "/" + path
					}
					methods["GET"] = true
					methods["POST"] = true
				} else { //`xweb:"GET|POST"`
					for _, method := range strings.Split(tags[0], "|") {
						method = strings.ToUpper(method)
						methods[method] = true
					}
					path = "/" + actionShortName + "/" + name
					isEq = true
				}
			} else {
				path = "/" + actionShortName + "/" + name
				isEq = true
				methods["GET"] = true
				methods["POST"] = true
			}
			p = strings.TrimRight(url, "/") + path
		} else {
			p = strings.TrimRight(url, "/") + "/" + actionShortName + "/" + name
			isEq = true
			methods["GET"] = true
			methods["POST"] = true
		}
		p = removeStick(p)
		if isEq {
			app.addEqRoute(p, methods, t, a)
			app.ActionsMethodRoute[actionFullName][name] = p
		} else {
			app.addRoute(p, methods, t, a)
		}
		app.Debug("Action:", actionFullName+"."+a+";", "Route Information:", p+";", "Request Method:", methods)
	}
}

func (a *App) ElapsedTimeString() string {
	return fmt.Sprintf("%.3fs", a.ElapsedTime())
}

func (a *App) ElapsedTime() float64 {
	return time.Now().Sub(a.RequestTime).Seconds()
}

func (a *App) VisitedLog(req *http.Request, statusCode int, requestPath string, responseSize int64) {
	if statusCode == 0 {
		statusCode = 200
	}
	if statusCode >= 200 && statusCode < 400 {
		a.Info(req.RemoteAddr, req.Method, statusCode, requestPath, responseSize, a.ElapsedTimeString())
	} else {
		a.Error(req.RemoteAddr, req.Method, statusCode, requestPath, responseSize, a.ElapsedTimeString())
	}
}

// the main route handler in web.go
func (a *App) routeHandler(req *http.Request, w http.ResponseWriter) {
	var (
		requestPath  string = req.URL.Path
		statusCode   int    = 0
		responseSize int64  = 0
	)
	defer func() {
		a.VisitedLog(req, statusCode, requestPath, responseSize)
	}()

	if !a.IsRootApp() || a.Server.Config.UrlSuffix != "" || a.Server.Config.UrlPrefix != "" {
		// static files, needed op
		if req.Method == "GET" || req.Method == "HEAD" {
			success, size := a.TryServingFile(requestPath, req, w)
			if success {
				statusCode = 200
				responseSize = size
				return
			}
			if requestPath == "/favicon.ico" {
				statusCode = 404
				a.error(w, 404, "Page not found")
				return
			}
		}
	}

	//ignore errors from ParseForm because it's usually harmless.
	ct := req.Header.Get("Content-Type")
	if strings.Contains(ct, "multipart/form-data") {
		req.ParseMultipartForm(a.AppConfig.MaxUploadSize)
	} else {
		req.ParseForm()
	}

	//Set the default content-type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if !a.filter(w, req) {
		statusCode = 302
		return
	}
	requestPath = req.URL.Path //支持filter更改req.URL.Path

	reqPath := removeStick(requestPath)
	if a.Domain == "" && a.BasePath != "/" {
		reqPath = "/" + strings.TrimPrefix(reqPath, a.BasePath)
	}
	allowMethod := Ternary(req.Method == "HEAD", "GET", req.Method).(string)
	isFind := false
	if routes, ok := a.RoutesEq[reqPath]; ok {
		if route, ok := routes[allowMethod]; ok {
			var isBreak bool = false
			var args []reflect.Value
			isBreak, statusCode, responseSize = a.run(req, w, route, args)
			if isBreak {
				return
			}
			isFind = true
		}
	}
	if !isFind {
		for _, route := range a.Routes {
			cr := route.CompiledRegexp

			//if the methods don't match, skip this handler (except HEAD can be used in place of GET)
			if _, ok := route.HttpMethods[allowMethod]; !ok {
				continue
			}

			if !cr.MatchString(reqPath) {
				continue
			}

			match := cr.FindStringSubmatch(reqPath)

			if match[0] != reqPath {
				continue
			}

			var args []reflect.Value
			for _, arg := range match[1:] {
				args = append(args, reflect.ValueOf(arg))
			}
			var isBreak bool = false
			isBreak, statusCode, responseSize = a.run(req, w, route, args)
			if isBreak {
				return
			}
		}
	}
	// try serving index.html or index.htm
	if req.Method == "GET" || req.Method == "HEAD" {
		if ok, size := a.TryServingFile(path.Join(requestPath, "index.html"), req, w); ok {
			statusCode = 200
			responseSize = size
			return
		} else if ok, size := a.TryServingFile(path.Join(requestPath, "index.htm"), req, w); ok {
			statusCode = 200
			responseSize = size
			return
		}
	}

	a.error(w, 404, "Page not found")
	statusCode = 404
}

func (a *App) run(req *http.Request, w http.ResponseWriter, route Route, args []reflect.Value) (isBreak bool, statusCode int, responseSize int64) {
	isBreak = true
	vc := reflect.New(route.HandlerElement)
	c := &Action{
		Request:        req,
		App:            a,
		ResponseWriter: w,
		T:              T{},
		f:              T{},
		Option: &ActionOption{
			AutoMapForm: a.AppConfig.FormMapToStruct,
			CheckXsrf:   a.AppConfig.CheckXsrf,
		},
	}

	for k, v := range a.VarMaps {
		c.T[k] = v
	}

	//设置Action字段的值
	fieldA := vc.Elem().FieldByName("Action")
	if fieldA.IsValid() {
		fieldA.Set(reflect.ValueOf(c))
	}

	//设置C字段的值
	fieldC := vc.Elem().FieldByName("C")
	if fieldC.IsValid() {
		fieldC.Set(reflect.ValueOf(vc))
	}

	//执行Init方法
	initM := vc.MethodByName("Init")
	if initM.IsValid() {
		params := []reflect.Value{}
		initM.Call(params)
	}

	//表单数据自动映射到结构体
	if c.Option.AutoMapForm {
		a.StructMap(vc.Elem(), req)
	}

	//验证XSRF
	if c.Option.CheckXsrf {
		a.XsrfManager.Init(c)
		if req.Method == "POST" {
			formVals := req.Form[XSRF_TAG]
			var formVal string
			if len(formVals) > 0 {
				formVal = formVals[0]
			}
			if formVal == "" || !a.XsrfManager.Valid(a.AppConfig.CookiePrefix+XSRF_TAG, formVal) {
				a.error(w, 500, "xsrf token error.")
				a.Error("xsrf token error.")
				statusCode = 500
				return
			}
		}
	}
	//执行Before方法
	structName := reflect.ValueOf(route.HandlerElement.Name())
	actionName := reflect.ValueOf(route.HandlerMethod)
	initM = vc.MethodByName("Before")
	if initM.IsValid() {
		structAction := []reflect.Value{structName, actionName}
		if ok := initM.Call(structAction); ok[0].Kind() == reflect.Bool && !ok[0].Bool() {
			responseSize = c.ResponseSize
			return
		}
	}

	ret, err := a.SafelyCall(vc, route.HandlerMethod, args)
	if err != nil {
		//there was an error or panic while calling the handler
		if a.AppConfig.Mode == Debug {
			a.error(w, 500, fmt.Sprintf("<pre>handler error: %v</pre>", err))
		} else if a.AppConfig.Mode == Product {
			a.error(w, 500, "Server Error")
		}
		statusCode = 500
		responseSize = c.ResponseSize
		return
	}
	statusCode = fieldA.Interface().(*Action).StatusCode

	//执行After方法
	initM = vc.MethodByName("After")
	if initM.IsValid() {
		structAction := []reflect.Value{structName, actionName}
		for _, v := range ret {
			structAction = append(structAction, v)
		}
		if len(structAction) != initM.Type().NumIn() {
			a.Errorf("Error : %v.After(): The number of params is not adapted.", structName)
			return
		}
		ret = initM.Call(structAction)
	}

	if len(ret) == 0 {
		responseSize = c.ResponseSize
		return
	}

	sval := ret[0]
	intf := sval.Interface()
	kind := sval.Kind()
	var content []byte
	if intf == nil || kind == reflect.Bool {
		responseSize = c.ResponseSize
		return
	} else if kind == reflect.String {
		content = []byte(sval.String())
	} else if kind == reflect.Slice && sval.Type().Elem().Kind() == reflect.Uint8 {
		content = intf.([]byte)
	} else if _, ok := intf.(bool); ok {
		responseSize = c.ResponseSize
		return
	} else if obj, ok := intf.(JSON); ok {
		c.ServeJson(obj.Data)
		responseSize = c.ResponseSize
		return
	} else if obj, ok := intf.(JSONP); ok {
		c.ServeJsonp(obj.Data, obj.Callback)
		responseSize = c.ResponseSize
		return
	} else if obj, ok := intf.(XML); ok {
		c.ServeXml(obj.Data)
		responseSize = c.ResponseSize
		return
	} else if file, ok := intf.(FILE); ok {
		c.ServeFile(file.Data)
		return
	} else if err, ok := intf.(error); ok {
		if err != nil {
			a.Error("Error:", err)
			a.error(w, 500, "Server Error")
			statusCode = 500
		} else {
			responseSize = c.ResponseSize
		}
		return
	} else if str, ok := intf.(string); ok {
		content = []byte(str)
	} else if byt, ok := intf.([]byte); ok {
		content = byt
	} else {
		a.Warnf("unknown returned result type %v, ignored %v", kind, intf)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	size, err := w.Write(content)
	if err != nil {
		a.Errorf("Error during write: %v", err)
		statusCode = 500
		return
	}
	responseSize = int64(size)
	return
}

func (a *App) error(w http.ResponseWriter, status int, content string) error {
	w.WriteHeader(status)
	if errorTmpl == "" {
		errTmplFile := a.AppConfig.TemplateDir + "/_error.html"
		if file, err := os.Stat(errTmplFile); err == nil && !file.IsDir() {
			if b, e := ioutil.ReadFile(errTmplFile); e == nil {
				errorTmpl = string(b)
			}
		}
		if errorTmpl == "" {
			errorTmpl = defaultErrorTmpl
		}
	}
	res := fmt.Sprintf(errorTmpl, status, statusText[status],
		status, statusText[status], content, Version)
	_, err := w.Write([]byte(res))
	return err
}

func (a *App) StaticUrl(url string) string {
	var basePath string
	if a.AppConfig.StaticDir == RootApp().AppConfig.StaticDir {
		basePath = RootApp().BasePath
	} else {
		basePath = a.BasePath
	}
	if !a.AppConfig.StaticFileVersion {
		return path.Join(basePath, url)
	}
	ver := a.StaticVerMgr.GetVersion(url)
	if ver == "" {
		return path.Join(basePath, url)
	}
	return path.Join(basePath, url+"?v="+ver)
}

// safelyCall invokes `function` in recover block
func (a *App) SafelyCall(vc reflect.Value, method string, args []reflect.Value) (resp []reflect.Value, err error) {
	defer func() {
		if e := recover(); e != nil {
			if !a.Server.Config.RecoverPanic {
				// go back to panic
				panic(e)
			} else {
				resp = nil
				var content string
				content = fmt.Sprintf("Handler crashed with error: %v", e)
				for i := 1; ; i += 1 {
					_, file, line, ok := runtime.Caller(i)
					if !ok {
						break
					} else {
						content += "\n"
					}
					content += fmt.Sprintf("%v %v", file, line)
				}
				a.Error(content)
				err = errors.New(content)
				return
			}
		}
	}()
	function := vc.MethodByName(method)
	return function.Call(args), err
}

// Init content-length header.
func (a *App) InitHeadContent(w http.ResponseWriter, contentLength int64) {
	if a.ContentEncoding == "gzip" {
		w.Header().Set("Content-Encoding", "gzip")
	} else if a.ContentEncoding == "deflate" {
		w.Header().Set("Content-Encoding", "deflate")
	} else {
		w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	}
}

// tryServingFile attempts to serve a static file, and returns
// whether or not the operation is successful.
func (a *App) TryServingFile(name string, req *http.Request, w http.ResponseWriter) (bool, int64) {
	newPath := name
	if strings.HasPrefix(name, a.BasePath) {
		newPath = name[len(a.BasePath):]
	}
	var size int64
	staticFile := filepath.Join(a.AppConfig.StaticDir, newPath)
	finfo, err := os.Stat(staticFile)
	if err != nil {
		return false, size
	}
	if !finfo.IsDir() {
		size = finfo.Size()
		isStaticFileToCompress := false
		if a.Server.Config.EnableGzip && a.Server.Config.StaticExtensionsToGzip != nil && len(a.Server.Config.StaticExtensionsToGzip) > 0 {
			for _, statExtension := range a.Server.Config.StaticExtensionsToGzip {
				if strings.HasSuffix(strings.ToLower(staticFile), strings.ToLower(statExtension)) {
					isStaticFileToCompress = true
					break
				}
			}
		}
		if isStaticFileToCompress {
			a.ContentEncoding = GetAcceptEncodingZip(req)
			memzipfile, err := OpenMemZipFile(staticFile, a.ContentEncoding)
			if err != nil {
				return false, size
			}
			a.InitHeadContent(w, finfo.Size())
			http.ServeContent(w, req, staticFile, finfo.ModTime(), memzipfile)
		} else {
			http.ServeFile(w, req, staticFile)
		}
		return true, size
	}
	return false, size
}

// StructMap function mapping params to controller's properties
func (a *App) StructMap(m interface{}, r *http.Request) error {
	return a.namedStructMap(m, r, "")
}

// user[name][test]
func SplitJson(s string) ([]string, error) {
	res := make([]string, 0)
	var begin, end int
	var isleft bool
	for i, r := range s {
		switch r {
		case '[':
			isleft = true
			if i > 0 && s[i-1] != ']' {
				if begin == end {
					return nil, errors.New("unknow character")
				}
				res = append(res, s[begin:end+1])
			}
			begin = i + 1
			end = begin
		case ']':
			if !isleft {
				return nil, errors.New("unknow character")
			}
			isleft = false
			if begin != end {
				//return nil, errors.New("unknow character")

				res = append(res, s[begin:end+1])
				begin = i + 1
				end = begin
			}
		default:
			end = i
		}
		if i == len(s)-1 && begin != end {
			res = append(res, s[begin:end+1])
		}
	}
	return res, nil
}

func (a *App) namedStructMap(m interface{}, r *http.Request, topName string) error {
	vc := reflect.ValueOf(m).Elem()
	tc := reflect.TypeOf(m).Elem()
	for k, t := range r.Form {
		if k == XSRF_TAG || k == "" {
			continue
		}

		if topName != "" {
			if !strings.HasPrefix(k, topName) {
				continue
			}
			k = k[len(topName)+1:]
		}

		v := t[0]
		names := strings.Split(k, ".")
		var err error
		if len(names) == 1 {
			names, err = SplitJson(k)
			if err != nil {
				a.Warn("Unrecognize form key", k, err)
				continue
			}
		}

		var value reflect.Value = vc
		for i, name := range names {
			name = strings.Title(name)
			if i != len(names)-1 {
				if value.Kind() != reflect.Struct {
					a.Warnf("arg error, value kind is %v", value.Kind())
					break
				}

				//fmt.Println(name)
				value = value.FieldByName(name)
				if !value.IsValid() {
					a.Warnf("(%v value is not valid %v)", name, value)
					break
				}
				if !value.CanSet() {
					a.Warnf("can not set %v -> %v", name, value.Interface())
					break
				}
				if tagfast.Tag2(tc, name, "form_options") == "-" {
					continue
				}
				if value.Kind() == reflect.Ptr {
					if value.IsNil() {
						value.Set(reflect.New(value.Type().Elem()))
					}
					value = value.Elem()
				}
			} else {
				if value.Kind() != reflect.Struct {
					a.Warnf("arg error, value %v kind is %v", name, value.Kind())
					break
				}
				tv := value.FieldByName(name)
				if !tv.IsValid() {
					break
				}
				if !tv.CanSet() {
					a.Warnf("can not set %v to %v", k, tv)
					break
				}
				if tagfast.Tag2(tc, name, "form_options") == "-" {
					continue
				}
				if tv.Kind() == reflect.Ptr {
					tv.Set(reflect.New(tv.Type().Elem()))
					tv = tv.Elem()
				}

				var l interface{}
				switch k := tv.Kind(); k {
				case reflect.String:
					switch tagfast.Tag2(tc, name, "form_filter") {
					case "html":
						v = DefaultHtmlFilter(v)
					}
					l = v
					tv.Set(reflect.ValueOf(l))
				case reflect.Bool:
					l = (v != "false" && v != "0")
					tv.Set(reflect.ValueOf(l))
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
					x, err := strconv.Atoi(v)
					if err != nil {
						a.Warnf("arg %v as int: %v", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Int64:
					x, err := strconv.ParseInt(v, 10, 64)
					if err != nil {
						a.Warnf("arg %v as int64: %v", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Float32, reflect.Float64:
					x, err := strconv.ParseFloat(v, 64)
					if err != nil {
						a.Warnf("arg %v as float64: %v", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					x, err := strconv.ParseUint(v, 10, 64)
					if err != nil {
						a.Warnf("arg %v as uint: %v", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Struct:
					if tvf, ok := tv.Interface().(FromConversion); ok {
						err := tvf.FromString(v)
						if err != nil {
							a.Warnf("struct %v invoke FromString faild", tvf)
						}
					} else if tv.Type().String() == "time.Time" {
						x, err := time.Parse("2006-01-02 15:04:05.000 -0700", v)
						if err != nil {
							x, err = time.Parse("2006-01-02 15:04:05", v)
							if err != nil {
								x, err = time.Parse("2006-01-02", v)
								if err != nil {
									a.Warnf("unsupported time format %v, %v", v, err)
									break
								}
							}
						}
						l = x
						tv.Set(reflect.ValueOf(l))
					} else {
						a.Warn("can not set an struct which is not implement Fromconversion interface")
					}
				case reflect.Ptr:
					a.Warn("can not set an ptr of ptr")
				case reflect.Slice, reflect.Array:
					tt := tv.Type().Elem()
					tk := tt.Kind()
					if tk == reflect.String {
						tv.Set(reflect.ValueOf(t))
						break
					}

					if tv.IsNil() {
						tv.Set(reflect.MakeSlice(tv.Type(), len(t), len(t)))
					}

					for i, s := range t {
						var err error
						switch tk {
						case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int8, reflect.Int64:
							var v int64
							v, err = strconv.ParseInt(s, 10, tt.Bits())
							if err == nil {
								tv.Index(i).SetInt(v)
							}
						case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
							var v uint64
							v, err = strconv.ParseUint(s, 10, tt.Bits())
							if err == nil {
								tv.Index(i).SetUint(v)
							}
						case reflect.Float32, reflect.Float64:
							var v float64
							v, err = strconv.ParseFloat(s, tt.Bits())
							if err == nil {
								tv.Index(i).SetFloat(v)
							}
						case reflect.Bool:
							var v bool
							v, err = strconv.ParseBool(s)
							if err == nil {
								tv.Index(i).SetBool(v)
							}
						case reflect.Complex64, reflect.Complex128:
							// TODO:
							err = fmt.Errorf("unsupported slice element type %v", tk.String())
						default:
							err = fmt.Errorf("unsupported slice element type %v", tk.String())
						}
						if err != nil {
							a.Warnf("slice error: %v, %v", name, err)
							break
						}
					}
				default:
					break
				}
			}
		}
	}
	return nil
}

func (app *App) Redirect(w http.ResponseWriter, requestPath, url string, status ...int) error {
	err := redirect(w, url, status...)
	if err != nil {
		app.Errorf("redirect error: %s", err)
		return err
	}
	return nil
}

func (app *App) Action(name string) interface{} {
	if v, ok := app.Actions[name]; ok {
		return v
	}
	return nil
}

/*
example:
{
	"AdminAction":{
		"Index":["GET","POST"],
		"Add":	["GET","POST"],
		"Edit":	["GET","POST"]
	}
}
*/
func (app *App) Nodes() (r map[string]map[string][]string) {
	r = make(map[string]map[string][]string)
	for _, val := range app.Routes {
		name := val.HandlerElement.Name()
		if _, ok := r[name]; !ok {
			r[name] = make(map[string][]string)
		}
		if _, ok := r[name][val.HandlerMethod]; !ok {
			r[name][val.HandlerMethod] = make([]string, 0)
		}
		for k, _ := range val.HttpMethods {
			r[name][val.HandlerMethod] = append(r[name][val.HandlerMethod], k) //FUNC1:[POST,GET]
		}
	}
	for _, vals := range app.RoutesEq {
		for k, v := range vals {
			name := v.HandlerElement.Name()
			if _, ok := r[name]; !ok {
				r[name] = make(map[string][]string)
			}
			if _, ok := r[name][v.HandlerMethod]; !ok {
				r[name][v.HandlerMethod] = make([]string, 0)
			}
			r[name][v.HandlerMethod] = append(r[name][v.HandlerMethod], k) //FUNC1:[POST,GET]
		}
	}
	return
}
