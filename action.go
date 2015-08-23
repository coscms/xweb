package xweb

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/coscms/xweb/httpsession"
	"github.com/coscms/xweb/log"
	"github.com/coscms/xweb/uuid"
)

type Mapper struct {
}

type T map[string]interface{}

type ActionOption struct {
	AutoMapForm bool
	CheckXsrf   bool
}

// An Action object or it's substruct is created for every incoming HTTP request.
// It provides information
// about the request, including the http.Request object, the GET and POST params,
// and acts as a Writer for the response.
type Action struct {
	Request *http.Request
	App     *App
	Option  *ActionOption
	http.ResponseWriter
	C             reflect.Value
	session       *httpsession.Session
	T             T
	f             T
	RootTemplate  *template.Template
	RequestBody   []byte
	StatusCode    int
	ResponseSize  int64
	JsonpCallback string
}

// Protocol returns request protocol name, such as HTTP/1.1 .
func (c *Action) Protocol() string {
	return c.Request.Proto
}

// Uri returns full request url with query string, fragment.
func (c *Action) Uri() string {
	return c.Request.RequestURI
}

// Url returns request url path (without query string, fragment).
func (c *Action) Url() string {
	return c.Request.URL.String()
}

// Site returns base site url as scheme://domain type.
func (c *Action) Site() string {
	return c.Scheme() + "://" + c.Domain()
}

// Scheme returns request scheme as "http" or "https".
func (c *Action) Scheme() string {
	if c.Request.URL.Scheme != "" {
		return c.Request.URL.Scheme
	} else if c.Request.TLS == nil {
		return "http"
	} else {
		return "https"
	}
}

// Domain returns host name.
// Alias of Host method.
func (c *Action) Domain() string {
	return c.Host()
}

// Host returns host name.
// if no host info in request, return localhost.
func (c *Action) Host() string {
	if c.Request.Host != "" {
		hostParts, _, _ := net.SplitHostPort(c.Request.Host)
		if len(hostParts) > 0 {
			return hostParts
		}
		return c.Request.Host
	}
	return "localhost"
}

// Is returns boolean of this request is on given method, such as Is("POST").
func (c *Action) Is(method string) bool {
	return c.Method() == method
}

// IsAjax returns boolean of this request is generated by ajax.
func (c *Action) IsAjax() bool {
	return c.Header("X-Requested-With") == "XMLHttpRequest"
}

// IsSecure returns boolean of this request is in https.
func (c *Action) IsSecure() bool {
	return c.Scheme() == "https"
}

// IsSecure returns boolean of this request is in webSocket.
func (c *Action) IsWebsocket() bool {
	return c.Header("Upgrade") == "websocket"
}

// IsSecure returns boolean of whether file uploads in this request or not..
func (c *Action) IsUpload() bool {
	return c.Request.MultipartForm != nil
}

// ClientIP implements a best effort algorithm to return the real client IP, it parses
// X-Real-IP and X-Forwarded-For in order to work properly with reverse-proxies such us: nginx or haproxy.
func (c *Action) IP() string {
	ip := strings.TrimSpace(c.Header("X-Real-Ip"))
	if len(ip) > 0 {
		return ip
	}
	ip = c.Header("X-Forwarded-For")
	if ip != "" {
		if index := strings.IndexByte(ip, ','); index >= 0 {
			ip = ip[0:index]
		}
		ip = strings.TrimSpace(ip)
		if len(ip) > 0 {
			return ip
		}
	}
	ip = strings.TrimSpace(c.Request.RemoteAddr)
	ip, _, _ = net.SplitHostPort(ip)
	return ip
}

// Proxy returns proxy client ips slice.
func (c *Action) Proxy() []string {
	if ips := c.Header("X-Forwarded-For"); ips != "" {
		return strings.Split(ips, ",")
	}
	return []string{}
}

// Refer returns http referer header.
func (c *Action) Refer() string {
	return c.Header("Referer")
}

// SubDomains returns sub domain string.
// if aa.bb.domain.com, returns aa.bb .
func (c *Action) SubDomains() string {
	parts := strings.Split(c.Host(), ".")
	return strings.Join(parts[len(parts)-2:], ".")
}

// Port returns request client port.
// when error or empty, return 80.
func (c *Action) Port() int {
	parts := strings.Split(c.Request.Host, ":")
	if len(parts) == 2 {
		port, _ := strconv.Atoi(parts[1])
		return port
	}
	return 80
}

// UserAgent returns request client user agent string.
func (c *Action) UserAgent() string {
	return c.Header("User-Agent")
}

// Query returns input data item string by a given string.
func (c *Action) Query(key string) string {
	c.Request.ParseForm()
	return c.Request.Form.Get(key)
}

// Header returns request header item string by a given string.
func (c *Action) Header(key string) string {
	return c.Request.Header.Get(key)
}

// Cookie returns request cookie item string by a given key.
// if non-existed, return empty string.
func (c *Action) Cookie(key string) string {
	ck, err := c.Request.Cookie(key)
	if err != nil {
		return ""
	}
	return ck.Value
}

// Body returns the raw request body data as bytes.
func (c *Action) Body() []byte {
	if len(c.RequestBody) > 0 {
		return c.RequestBody
	}

	requestbody, _ := ioutil.ReadAll(c.Request.Body)
	c.Request.Body.Close()
	bf := bytes.NewBuffer(requestbody)
	c.Request.Body = ioutil.NopCloser(bf)
	c.RequestBody = requestbody
	return requestbody
}

func (c *Action) DisableHttpCache() {
	c.SetHeader("Expires", "Mon, 26 Jul 1997 05:00:00 GMT")
	c.SetHeader("Last-Modified", webTime(time.Now().UTC()))
	c.SetHeader("Cache-Control", "no-store, no-cache, must-revalidate")
	c.SetHeader("Cache-Control", "post-check=0, pre-check=0")
	c.SetHeader("Pragma", "no-cache")
}

func (c *Action) HttpCache(content []byte) bool {
	h := md5.New()
	h.Write(content)
	Etag := hex.EncodeToString(h.Sum(nil))
	//c.SetHeader("Connection", "keep-alive")
	c.SetHeader("X-Cache", "HIT from COSCMS-Page-Cache")
	//c.SetHeader("X-Cache", "HIT from COSCMS-Page-Cache 2013-12-02 17:16:01")
	if inm := c.Header("If-None-Match"); inm != "" && inm == Etag {
		h := c.ResponseWriter.Header()
		delete(h, "Content-Type")
		delete(h, "Content-Length")
		//c.ResponseWriter.WriteHeader(http.StatusNotModified)
		c.NotModified()
		return true
	}
	c.SetHeader("Etag", Etag)
	c.SetHeader("Cache-Control", "public,max-age=1")
	return false
}

// Body sets response body content.
// if EnableGzip, compress content string.
// it sends out response body directly.
func (c *Action) SetBody(content []byte) error {
	if c.App.AppConfig.EnableHttpCache && c.HttpCache(content) {
		return nil
	}
	output_writer := c.ResponseWriter.(io.Writer)
	if c.App.Server.Config.EnableGzip == true && c.Header("Accept-Encoding") != "" {
		splitted := strings.SplitN(c.Header("Accept-Encoding"), ",", -1)
		encodings := make([]string, len(splitted))

		for i, val := range splitted {
			encodings[i] = strings.TrimSpace(val)
		}
		for _, val := range encodings {
			if val == "gzip" {
				c.SetHeader("Content-Encoding", "gzip")
				output_writer, _ = gzip.NewWriterLevel(c.ResponseWriter, gzip.BestSpeed)
				break
			} else if val == "deflate" {
				c.SetHeader("Content-Encoding", "deflate")
				output_writer, _ = flate.NewWriter(c.ResponseWriter, flate.BestSpeed)
				break
			}
		}
	} else {
		c.SetHeader("Content-Length", strconv.Itoa(len(content)))
	}
	size, err := output_writer.Write(content)
	c.ResponseSize += int64(size)
	switch output_writer.(type) {
	case *gzip.Writer:
		output_writer.(*gzip.Writer).Close()
	case *flate.Writer:
		output_writer.(*flate.Writer).Close()
	}
	return err
}

func (c *Action) XsrfValue() string {
	c.App.XsrfManager.Init(c)
	var name string = c.App.AppConfig.CookiePrefix + XSRF_TAG
	var val string = c.App.XsrfManager.Get(name)
	if val == "" {
		val = uuid.NewRandom().String()
		c.App.XsrfManager.Set(name, val)
	}
	return val
}

func (c *Action) XsrfFormHtml() template.HTML {
	if c.App.AppConfig.CheckXsrf {
		return template.HTML(fmt.Sprintf(`<input type="hidden" name="%v" value="%v" />`,
			XSRF_TAG, c.XsrfValue()))
	}
	return template.HTML("")
}

// WriteString writes string data into the response object.
func (c *Action) WriteBytes(bytes []byte) error {
	err := c.SetBody(bytes)
	if err != nil {
		c.App.Error("Error during write:", err)
	}
	return err
}

func (c *Action) Write(content string, values ...interface{}) error {
	if len(values) > 0 {
		content = fmt.Sprintf(content, values...)
	}
	err := c.SetBody([]byte(content))
	if err != nil {
		c.App.Error("Error during write:", err)
	}
	return err
}

// Abort is a helper method that sends an HTTP header and an optional
// body. It is useful for returning 4xx or 5xx errors.
// Once it has been called, any return value from the handler will
// not be written to the response.
func (c *Action) Abort(status int, body string) error {
	c.StatusCode = status
	return c.App.error(c.ResponseWriter, status, body)
}

// Redirect is a helper method for 3xx redirects.
func (c *Action) Redirect(url string, status ...int) error {
	if len(status) == 0 {
		c.StatusCode = 302
	} else {
		c.StatusCode = status[0]
	}
	return c.App.Redirect(c.ResponseWriter, c.Request.URL.Path, url, status...)
}

// Notmodified writes a 304 HTTP response
func (c *Action) NotModified() {
	c.StatusCode = 304
	c.ResponseWriter.WriteHeader(304)
}

// NotFound writes a 404 HTTP response
func (c *Action) NotFound(message string) error {
	c.StatusCode = 404
	return c.Abort(404, message)
}

// ParseStruct mapping forms' name and values to struct's field
// For example:
//		<form>
//			<input name="user.id"/>
//			<input name="user.name"/>
//			<input name="user.age"/>
//		</form>
//
//		type User struct {
//			Id int64
//			Name string
//			Age string
//		}
//
//		var user User
//		err := action.MapForm(&user)
//
func (c *Action) MapForm(st interface{}, names ...string) error {
	v := reflect.ValueOf(st)
	var name string
	if len(names) == 0 {
		name = UnTitle(v.Type().Elem().Name())
	} else {
		name = names[0]
	}
	return c.App.namedStructMap(v.Elem(), c.Request, name)
}

// ContentType sets the Content-Type header for an HTTP response.
// For example, c.ContentType("json") sets the content-type to "application/json"
// If the supplied value contains a slash (/) it is set as the Content-Type
// verbatim. The return value is the content type as it was
// set, or an empty string if none was found.
func (c *Action) SetContentType(val string) string {
	var ctype string
	if strings.ContainsRune(val, '/') {
		ctype = val
	} else {
		if !strings.HasPrefix(val, ".") {
			val = "." + val
		}
		ctype = mime.TypeByExtension(val)
	}
	if ctype != "" {
		c.SetHeader("Content-Type", ctype)
	}
	return ctype
}

// SetCookie adds a cookie header to the response.
func (c *Action) SetCookie(cookie *http.Cookie) {
	c.SetHeader("Set-Cookie", cookie.String())
}

func (c *Action) NewCookie(name string, value string, args ...interface{}) *http.Cookie {
	length := len(args)
	if length < 1 {
		args = append(args, c.App.AppConfig.SessionTimeout)
	}
	if length < 2 {
		args = append(args, "/")
	}
	if length < 3 {
		args = append(args, c.App.AppConfig.CookieDomain)
	}
	if length < 4 {
		args = append(args, c.IsSecure())
	}
	return NewCookie(c.App.AppConfig.CookiePrefix+name, value, args...)
}

func (c *Action) GetCookie(cookieName string) (*http.Cookie, error) {
	return c.Request.Cookie(c.App.AppConfig.CookiePrefix + cookieName)
}

func (c *Action) SetSecureCookie(name string, val string, args ...interface{}) {
	//base64 encode the val
	if len(c.App.AppConfig.CookieSecret) == 0 {
		c.App.Error("Secret Key for secure cookies has not been set. Please assign a cookie secret to web.Config.CookieSecret.")
		return
	}
	vs := Base64Encode(val)
	vb := []byte(vs)
	key := c.App.AppConfig.CookieSecret
	if c.App.AppConfig.CookieLimitIP {
		key += "|" + c.IP()
	}
	if c.App.AppConfig.CookieLimitUA {
		key += "|" + c.UserAgent()
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := getCookieSig(key, vb, timestamp)
	cookie := strings.Join([]string{vs, timestamp, sig}, "|")
	c.SetCookie(c.NewCookie(name, cookie, args...))
}

func (c *Action) GetSecureCookie(name string) (string, bool) {
	key := c.App.AppConfig.CookieSecret
	if c.App.AppConfig.CookieLimitIP {
		key += "|" + c.IP()
	}
	if c.App.AppConfig.CookieLimitUA {
		key += "|" + c.UserAgent()
	}
	name = c.App.AppConfig.CookiePrefix + name
	for _, cookie := range c.Request.Cookies() {
		if cookie.Name != name {
			continue
		}

		parts := strings.SplitN(cookie.Value, "|", 3)

		val := parts[0]
		timestamp := parts[1]
		sig := parts[2]

		if getCookieSig(key, []byte(val), timestamp) != sig {
			c.SetCookie(NewCookie(name, "", -86400))
			return "", false
		}

		ts, _ := strconv.ParseInt(timestamp, 0, 64)

		if time.Now().Unix()-31*86400 > ts {
			c.SetCookie(NewCookie(name, "", -86400))
			return "", false
		}

		return Base64Decode(val), true
	}
	return "", false
}

func (c *Action) Method() string {
	return c.Request.Method
}

func (c *Action) Go(m string, anotherc ...interface{}) error {
	var t reflect.Type
	if len(anotherc) > 0 {
		t = reflect.TypeOf(anotherc[0]).Elem()
	} else {
		t = reflect.TypeOf(c.C.Interface()).Elem()
	}

	if _, ok := c.App.ActionsPath[t]; !ok {
		return NotFound()
	}

	uris := strings.Split(m, "?")

	actionFullName := t.Name()

	if mr, ok := c.App.ActionsMethodRoute[actionFullName]; ok {
		if r, ok := mr[uris[0]]; ok {
			return c.Redirect(r)
		}
	}
	return NotFound()
}

func (c *Action) Flush() {
	flusher, _ := c.ResponseWriter.(http.Flusher)
	flusher.Flush()
}

func (c *Action) BasePath() string {
	return c.App.BasePath
}

func (c *Action) Namespace() string {
	return c.App.ActionsPath[c.C.Type()]
}

func (c *Action) Debug(params ...interface{}) {
	c.App.Debug(params...)
}

func (c *Action) Info(params ...interface{}) {
	c.App.Info(params...)
}

func (c *Action) Warn(params ...interface{}) {
	c.App.Warn(params...)
}

func (c *Action) Error(params ...interface{}) {
	c.App.Error(params...)
}

func (c *Action) Fatal(params ...interface{}) {
	c.App.Fatal(params...)
}

func (c *Action) Panic(params ...interface{}) {
	c.App.Panic(params...)
}

func (c *Action) Debugf(format string, params ...interface{}) {
	c.App.Debugf(format, params...)
}

func (c *Action) Infof(format string, params ...interface{}) {
	c.App.Infof(format, params...)
}

func (c *Action) Warnf(format string, params ...interface{}) {
	c.App.Warnf(format, params...)
}

func (c *Action) Errorf(format string, params ...interface{}) {
	c.App.Errorf(format, params...)
}

func (c *Action) Fatalf(format string, params ...interface{}) {
	c.App.Fatalf(format, params...)
}

func (c *Action) Panicf(format string, params ...interface{}) {
	c.App.Panicf(format, params...)
}

// Include method provide to template for {{include "xx.tmpl"}}
func (c *Action) Include(tmplName string) interface{} {
	t := c.RootTemplate.New(tmplName)
	t.Funcs(c.GetFuncs())

	content, err := c.getTemplate(tmplName)
	if err != nil {
		c.Errorf("RenderTemplate %v read err: %s", tmplName, err)
		return ""
	}

	constr := string(content)

	Event("BeforeRender", []interface{}{c, &constr}, func(_ bool) {})

	tmpl, err := t.Parse(constr)
	if err != nil {
		c.Errorf("Parse %v err: %v", tmplName, err)
		return ""
	}
	newbytes := bytes.NewBufferString("")
	err = tmpl.Execute(newbytes, c.C.Elem().Interface())
	if err != nil {
		c.Errorf("Parse %v err: %v", tmplName, err)
		return ""
	}

	tplcontent, err := ioutil.ReadAll(newbytes)
	if err != nil {
		c.Errorf("Parse %v err: %v", tmplName, err)
		return ""
	}
	return template.HTML(string(tplcontent))
}

// render the template with vars map, you can have zero or one map
func (c *Action) NamedRender(name, content string, params ...*T) error {
	c.f["include"] = c.Include
	if c.App.AppConfig.SessionOn {
		c.f["session"] = c.GetSession
	} else {
		c.f["session"] = func(key string) interface{} {
			return ""
		}
	}
	c.f["cookie"] = c.Cookie
	c.f["XsrfFormHtml"] = c.XsrfFormHtml
	c.f["XsrfValue"] = c.XsrfValue
	if len(params) > 0 {
		c.AddTmplVars(params[0])
	}

	c.RootTemplate = template.New(name)
	c.RootTemplate.Funcs(c.GetFuncs())

	Event("BeforeRender", []interface{}{c, &content}, func(_ bool) {})

	tmpl, err := c.RootTemplate.Parse(content)
	if err == nil {
		newbytes := bytes.NewBufferString("")
		err = tmpl.Execute(newbytes, c.C.Elem().Interface())
		if err == nil {
			tplcontent, err := ioutil.ReadAll(newbytes)
			if err == nil {
				Event("AfterRender", []interface{}{c, &tplcontent}, func(result bool) {
					if result {
						err = c.SetBody(tplcontent)
					}
				})
			}
		}
	}
	return err
}

func (c *Action) getTemplate(tmpl string) ([]byte, error) {
	if c.App.AppConfig.CacheTemplates {
		return c.App.TemplateMgr.GetTemplate(tmpl)
	}
	path := c.App.getTemplatePath(tmpl)
	if path == "" {
		return nil, errors.New(fmt.Sprintf("No template file %v found", path))
	}

	return ioutil.ReadFile(path)
}

// render the template with vars map, you can have zero or one map
func (c *Action) Render(tmpl string, params ...*T) error {
	content, err := c.getTemplate(tmpl)
	if err == nil {
		err = c.NamedRender(tmpl, string(content), params...)
	}
	return err
}

func (c *Action) GetFuncs() template.FuncMap {
	funcs := c.App.FuncMaps
	if c.f != nil {
		for k, v := range c.f {
			funcs[k] = v
		}
	}

	return funcs
}

func (c *Action) SetConfig(name string, value interface{}) {
	c.App.Config[name] = value
}

func (c *Action) GetConfig(name string) interface{} {
	return c.App.Config[name]
}

func (c *Action) RenderString(content string, params ...*T) error {
	h := md5.New()
	h.Write([]byte(content))
	name := h.Sum(nil)
	return c.NamedRender(string(name), content, params...)
}

// SetHeader sets a response header. the current value
// of that header will be overwritten .
func (c *Action) SetHeader(key string, value string) {
	c.ResponseWriter.Header().Set(key, value)
}

// add a name value for template
func (c *Action) AddTmplVar(name string, varOrFunc interface{}) {
	if varOrFunc == nil {
		c.T[name] = varOrFunc
		return
	}

	if reflect.ValueOf(varOrFunc).Type().Kind() == reflect.Func {
		c.f[name] = varOrFunc
	} else {
		c.T[name] = varOrFunc
	}
}

// add names and values for template
func (c *Action) AddTmplVars(t *T) {
	for name, value := range *t {
		c.AddTmplVar(name, value)
	}
}

func (c *Action) ServeJson(obj interface{}) {
	content, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		http.Error(c.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	c.SetHeader("Content-Length", strconv.Itoa(len(content)))
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	size, _ := c.ResponseWriter.Write(content)
	c.ResponseSize += int64(size)
}

func (c *Action) ServeJsonp(obj interface{}, callback string) {
	content, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		http.Error(c.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	if callback == "" {
		if c.JsonpCallback == "" {
			callback = c.GetString("callback")
		} else {
			callback = c.JsonpCallback
		}
	}
	if callback != "" {
		temp := callback + "(" + string(content) + ");"
		content = []byte(temp)
	}
	c.SetHeader("Content-Length", strconv.Itoa(len(content)))
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	size, _ := c.ResponseWriter.Write(content)
	c.ResponseSize += int64(size)
}

func (c *Action) ServeXml(obj interface{}) {
	content, err := xml.Marshal(obj)
	if err != nil {
		http.Error(c.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	c.SetHeader("Content-Length", strconv.Itoa(len(content)))
	c.ResponseWriter.Header().Set("Content-Type", "application/xml")
	size, _ := c.ResponseWriter.Write(content)
	c.ResponseSize += int64(size)
}

func (c *Action) ServeFile(fpath string) {
	c.ResponseWriter.Header().Del("Content-Type")
	http.ServeFile(c.ResponseWriter, c.Request, fpath)
}

func (c *Action) GetSlice(key string) []string {
	return c.Request.Form[key]
}

func (c *Action) GetForm() url.Values {
	return c.Request.Form
}

func (c *Action) GetString(key string) string {
	s := c.GetSlice(key)
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

func (c *Action) GetInt(key string) (int64, error) {
	return strconv.ParseInt(c.GetString(key), 10, 64)
}

func (c *Action) GetBool(key string) (bool, error) {
	return strconv.ParseBool(c.GetString(key))
}

func (c *Action) GetFloat(key string) (float64, error) {
	return strconv.ParseFloat(c.GetString(key), 64)
}

func (c *Action) GetFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return c.Request.FormFile(key)
}

func (c *Action) GetLogger() *log.Logger {
	return c.App.Logger
}

func (c *Action) SaveToFile(fromfile, tofile string) error {
	file, _, err := c.Request.FormFile(fromfile)
	if err != nil {
		return err
	}
	defer file.Close()
	f, err := os.OpenFile(tofile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, file)
	return err
}

func (c *Action) Session() *httpsession.Session {
	if c.session == nil {
		c.session = c.App.SessionManager.Session(c.Request, c.ResponseWriter)
	}
	return c.session
}

func (c *Action) GetSession(key string) interface{} {
	return c.Session().Get(key)
}

func (c *Action) SetSession(key string, value interface{}) {
	c.Session().Set(key, value)
}

func (c *Action) DelSession(keys ...string) {
	for _, key := range keys {
		c.Session().Del(key)
	}
}
