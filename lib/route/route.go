package route

import (
	//"fmt"
	"reflect"
	"regexp"
)

func NewRoute() *Route {
	return &Route{
		Static: make(map[string]*StaticRoute),
		Regexp: make([]*RegexpRoute, 0),
	}
}

type Route struct {
	Static map[string]*StaticRoute
	Regexp []*RegexpRoute
}

//map[string]*StaticRoute
type StaticRoute struct {
	ReflectType   reflect.Type
	RequestMethod map[string]bool
	Extensions    map[string]bool
	BothFuncs     map[string]bool
	ExecuteFunc   string
}

//[]*RegexpRoute
type RegexpRoute struct {
	RouteRule     string
	Regexp        *regexp.Regexp
	StaticLength  int
	StaticPath    string
	ReflectType   reflect.Type
	RequestMethod map[string]bool
	Extensions    map[string]bool
	BothFuncs     map[string]bool
	ExecuteFunc   string
}

func (r *Route) Set(route string, execFunc string,
	reqMethod map[string]bool, extensions map[string]bool,
	group map[string]bool, refType reflect.Type) {
	routeReg := regexp.QuoteMeta(route)
	if route == routeReg {
		a := &StaticRoute{
			ReflectType:   refType,
			RequestMethod: reqMethod,
			Extensions:    extensions,
			BothFuncs:     group,
			ExecuteFunc:   execFunc,
		}
		r.Static[route] = a
	} else {
		length, staticPath, regexpInstance := r.Rego(route, routeReg)
		a := &RegexpRoute{
			RouteRule:     route,
			Regexp:        regexpInstance,
			StaticLength:  length,
			StaticPath:    staticPath,
			ReflectType:   refType,
			RequestMethod: reqMethod,
			Extensions:    extensions,
			BothFuncs:     group,
			ExecuteFunc:   execFunc,
		}
		r.Regexp = append(r.Regexp, a)
	}
}

func (r *Route) Get(reqPath string, reqMethod string,
	extension string) ([]reflect.Value, string, reflect.Type, bool, bool, bool) {
	if route, ok := r.Static[reqPath]; ok {
		onMethod, ok := route.RequestMethod[reqMethod]
		if ok {
			onExtension, ok := route.Extensions[extension]
			if !ok {
				onExtension = false
			}
			onGroup, ok := route.BothFuncs[reqMethod+"_"+extension]
			if !ok {
				onGroup = false
			}
			return nil, route.ExecuteFunc, route.ReflectType, onMethod, onExtension, onGroup
		}
	}
	length := len(reqPath)
	for _, route := range r.Regexp {
		if route.StaticLength >= length || reqPath[0:route.StaticLength] != route.StaticPath {
			continue
		}
		onMethod, ok := route.RequestMethod[reqMethod]
		if !ok {
			continue
		}
		part := reqPath[route.StaticLength:]
		p := route.Regexp.FindStringSubmatch(part)
		if len(p) < 1 || p[0] != part {
			continue
		}
		var args []reflect.Value
		for _, arg := range p[1:] {
			args = append(args, reflect.ValueOf(arg))
		}
		onExtension, ok := route.Extensions[extension]
		if !ok {
			onExtension = false
		}
		onGroup, ok := route.BothFuncs[reqMethod+"_"+extension]
		if !ok {
			onGroup = false
		}
		return args, route.ExecuteFunc, route.ReflectType, onMethod, onExtension, onGroup
	}
	return nil, "", nil, false, false, false
}

func (r *Route) Rego(vOriginal string, vNew string) (length int, staticPath string, regexpInstance *regexp.Regexp) {
	var same []byte = make([]byte, 0)
	for k, v := range []byte(vNew) {
		if vOriginal[k] == v {
			same = append(same, v)
		} else {
			break
		}
	}
	length = len(same)
	staticPath = string(same)
	regexpInstance = regexp.MustCompile(vOriginal[length:])
	//println(vOriginal, staticPath, vOriginal[length:])
	return
}
