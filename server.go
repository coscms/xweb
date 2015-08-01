package xweb

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	runtimePprof "runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/coscms/xweb/httpsession"
	"github.com/coscms/xweb/log"
)

// ServerConfig is configuration for server objects.
type ServerConfig struct {
	Addr                   string
	Port                   int
	RecoverPanic           bool
	Profiler               bool
	EnableGzip             bool
	StaticExtensionsToGzip []string
	Url                    string
	UrlPrefix              string
	UrlSuffix              string
	StaticHtmlDir          string
	SessionTimeout         time.Duration
}

// Server represents a xweb server.
type Server struct {
	Config         *ServerConfig
	Apps           map[string]*App   //r["root"]
	App2Domain     map[string]string //r["www.coscms.com"]="root"
	Domain2App     map[string]string //r["root"]="www.coscms.com"
	AppsNamePath   map[string]string //r["root"]="/"
	Name           string
	SessionManager *httpsession.Manager
	RootApp        *App
	Logger         *log.Logger
	Env            map[string]interface{}

	http.ResponseWriter
	*http.Request
	//save the listener so it can be closed
	l net.Listener
}

func NewServer(name string) *Server {
	s := &Server{
		Config:       Config,
		Env:          map[string]interface{}{},
		Apps:         map[string]*App{},
		App2Domain:   map[string]string{},
		Domain2App:   map[string]string{},
		AppsNamePath: map[string]string{},
		Name:         name,
	}
	Servers[s.Name] = s

	s.SetLogger(log.New(os.Stdout, "", log.Ldefault()))

	app := NewApp("/", "root")
	s.AddApp(app)
	return s
}

func (s *Server) AddApp(a *App) {
	a.BasePath = strings.TrimRight(a.BasePath, "/") + "/"
	s.Apps[a.BasePath] = a

	if a.Name != "" {
		s.AppsNamePath[a.Name] = a.BasePath
	}

	a.Server = s
	a.Logger = s.Logger

	if a.BasePath == "/" {
		s.RootApp = a
	}
}

func (s *Server) AddAction(cs ...interface{}) {
	s.RootApp.AddAction(cs...)
}

func (s *Server) AutoAction(c ...interface{}) {
	s.RootApp.AutoAction(c...)
}

func (s *Server) AddRouter(url string, c interface{}) {
	s.RootApp.AddRouter(url, c)
}

func (s *Server) AddTmplVar(name string, varOrFun interface{}) {
	s.RootApp.AddTmplVar(name, varOrFun)
}

func (s *Server) AddTmplVars(t *T) {
	s.RootApp.AddTmplVars(t)
}

func (s *Server) AddFilter(filter Filter) {
	s.RootApp.AddFilter(filter)
}

func (s *Server) AddConfig(name string, value interface{}) {
	s.RootApp.SetConfig(name, value)
}

func (s *Server) SetConfig(name string, value interface{}) {
	s.RootApp.SetConfig(name, value)
}

func (s *Server) GetConfig(name string) interface{} {
	return s.RootApp.GetConfig(name)
}

func (s *Server) error(w http.ResponseWriter, status int, content string) error {
	return s.RootApp.error(w, status, content)
}

func (s *Server) initServer() {
	if s.Config == nil {
		s.Config = &ServerConfig{}
		s.Config.Profiler = true
	}

	for _, app := range s.Apps {
		app.initApp()
	}
}

// ServeHTTP is the interface method for Go's http server package
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.ResponseWriter = w
	s.Request = req
	s.Process()
}

// Process invokes the routing system for server s
// non-root app's route will override root app's if there is same path
func (s *Server) Process() {
	Event("ServerProcess", s, func(result bool) {
		if !result {
			return
		}
		w := s.ResponseWriter
		req := s.Request
		if s.Config.UrlSuffix != "" && strings.HasSuffix(req.URL.Path, s.Config.UrlSuffix) {
			req.URL.Path = strings.TrimSuffix(req.URL.Path, s.Config.UrlSuffix)
		}
		if s.Config.UrlPrefix != "" && strings.HasPrefix(req.URL.Path, "/"+s.Config.UrlPrefix) {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, "/"+s.Config.UrlPrefix)
		}
		if req.URL.Path[0] != '/' {
			req.URL.Path = "/" + req.URL.Path
		}
		var hostKey string
		if host, port, err := net.SplitHostPort(req.Host); err != nil {
			fmt.Println(err)
		} else if port != "80" {
			hostKey = req.Host
		} else {
			hostKey = host
		}
		var appName string
		if v, ok := s.Domain2App[hostKey]; ok {
			appName = v
		} else if v, ok := s.Domain2App["//"+hostKey]; ok {
			appName = v
		} else if v, ok := s.Domain2App[req.URL.Scheme+"://"+hostKey]; ok {
			appName = v
		}
		if appName != "" {
			if app := s.App(appName); app != nil {
				app.routeHandler(req, w)
				return
			}
		}
		for _, app := range s.Apps {
			if app != s.RootApp && strings.HasPrefix(req.URL.Path, app.BasePath) {
				app.routeHandler(req, w)
				return
			}
		}
		s.RootApp.routeHandler(req, w)
	})
}

// Run starts the web application and serves HTTP requests for s
func (s *Server) Run(addr string) {
	addrs := strings.Split(addr, ":")
	s.Config.Addr = addrs[0]
	s.Config.Port, _ = strconv.Atoi(addrs[1])

	s.initServer()

	mux := http.NewServeMux()
	if s.Config.Profiler {
		mux.Handle("/debug/pprof", http.HandlerFunc(pprof.Index))
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/block", pprof.Handler("block"))
		mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

		mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))

		mux.Handle("/debug/pprof/startcpuprof", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			StartCPUProfile()
		}))
		mux.Handle("/debug/pprof/stopcpuprof", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			StopCPUProfile()
		}))
		mux.Handle("/debug/pprof/memprof", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			runtime.GC()
			runtimePprof.WriteHeapProfile(rw)
		}))
		mux.Handle("/debug/pprof/gc", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			PrintGCSummary(rw)
		}))

	}

	Event("MuxHandle", mux, func(result bool) {
		if result {
			mux.Handle("/", s)
		}
	})

	s.Logger.Infof("http server is listening %s", addr)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		s.Logger.Error("ListenAndServe:", err)
	}
	s.l = l
	err = http.Serve(s.l, mux)
	s.l.Close()
}

// RunFcgi starts the web application and serves FastCGI requests for s.
func (s *Server) RunFcgi(addr string) {
	s.initServer()
	s.Logger.Infof("fcgi server is listening %s", addr)
	s.listenAndServeFcgi(addr)
}

// RunScgi starts the web application and serves SCGI requests for s.
func (s *Server) RunScgi(addr string) {
	s.initServer()
	s.Logger.Infof("scgi server is listening %s", addr)
	s.listenAndServeScgi(addr)
}

// RunTLS starts the web application and serves HTTPS requests for s.
func (s *Server) RunTLS(addr string, config *tls.Config) error {
	s.initServer()
	mux := http.NewServeMux()
	mux.Handle("/", s)
	l, err := tls.Listen("tcp", addr, config)
	if err != nil {
		s.Logger.Errorf("Listen: %v", err)
		return err
	}

	s.l = l

	s.Logger.Infof("https server is listening %s", addr)

	return http.Serve(s.l, mux)
}

// Close stops server s.
func (s *Server) Close() {
	if s.l != nil {
		s.l.Close()
	}
}

// SetLogger sets the logger for server s
func (s *Server) SetLogger(logger *log.Logger) {
	s.Logger = logger
	s.Logger.SetPrefix("[" + s.Name + "] ")
	if s.RootApp != nil {
		s.RootApp.Logger = s.Logger
	}
}

func (s *Server) InitSession() {
	if s.SessionManager == nil {
		s.SessionManager = httpsession.Default()
	}
	if s.Config.SessionTimeout > time.Second {
		s.SessionManager.SetMaxAge(s.Config.SessionTimeout)
	}
	s.SessionManager.Run()
	if s.RootApp != nil {
		s.RootApp.SessionManager = s.SessionManager
	}
}

func (s *Server) SetTemplateDir(path string) {
	s.RootApp.SetTemplateDir(path)
}

func (s *Server) SetStaticDir(path string) {
	s.RootApp.SetStaticDir(path)
}

func (s *Server) App(name string) *App {
	path, ok := s.AppsNamePath[name]
	if ok {
		return s.Apps[path]
	}
	return nil
}
