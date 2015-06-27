package xweb

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"
)

// the func is the same as condition ? true : false
func Ternary(express bool, trueVal interface{}, falseVal interface{}) interface{} {
	if express {
		return trueVal
	}
	return falseVal
}

// internal utility methods
func webTime(t time.Time) string {
	ftime := t.Format(time.RFC1123)
	if strings.HasSuffix(ftime, "UTC") {
		ftime = ftime[0:len(ftime)-3] + "GMT"
	}
	return ftime
}

func JoinPath(paths ...string) string {
	if len(paths) < 1 {
		return ""
	}
	res := ""
	for _, p := range paths {
		res = path.Join(res, p)
	}
	return res
}

func PageSize(total, limit int) int {
	if total <= 0 {
		return 1
	} else {
		x := total % limit
		if x > 0 {
			return total/limit + 1
		} else {
			return total / limit
		}
	}
}

func SimpleParse(data string) map[string]string {
	configs := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		vs := strings.Split(line, "=")
		if len(vs) == 2 {
			configs[strings.TrimSpace(vs[0])] = strings.TrimSpace(vs[1])
		}
	}
	return configs
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

// Urlencode is a helper method that converts a map into URL-encoded form data.
// It is a useful when constructing HTTP POST requests.
func Urlencode(data map[string]string) string {
	var buf bytes.Buffer
	for k, v := range data {
		buf.WriteString(url.QueryEscape(k))
		buf.WriteByte('=')
		buf.WriteString(url.QueryEscape(v))
		buf.WriteByte('&')
	}
	s := buf.String()
	return s[0 : len(s)-1]
}

func UnTitle(s string) string {
	if len(s) < 2 {
		return strings.ToLower(s)
	}
	return strings.ToLower(string(s[0])) + s[1:]
}

var slugRegex = regexp.MustCompile(`(?i:[^a-z0-9\-_])`)

// Slug is a helper function that returns the URL slug for string s.
// It's used to return clean, URL-friendly strings that can be
// used in routing.
func Slug(s string, sep string) string {
	if s == "" {
		return ""
	}
	slug := slugRegex.ReplaceAllString(s, sep)
	if slug == "" {
		return ""
	}
	quoted := regexp.QuoteMeta(sep)
	sepRegex := regexp.MustCompile("(" + quoted + "){2,}")
	slug = sepRegex.ReplaceAllString(slug, sep)
	sepEnds := regexp.MustCompile("^" + quoted + "|" + quoted + "$")
	slug = sepEnds.ReplaceAllString(slug, "")
	return strings.ToLower(slug)
}

// NewCookie is a helper method that returns a new http.Cookie object.
// Duration is specified in seconds. If the duration is zero, the cookie is permanent.
// This can be used in conjunction with ctx.SetCookie.
func NewCookie(name string, value string, args ...interface{}) *http.Cookie {
	var (
		alen     int = len(args)
		utctime  time.Time
		age      int64
		path     string
		domain   string
		secure   bool
		httpOnly bool
	)
	if alen > 0 {
		switch alen {
		case 2:
			if v, ok := args[1].(string); ok {
				path = v
			}
		case 3:
			if v, ok := args[1].(string); ok {
				path = v
			}
			if v, ok := args[2].(string); ok {
				domain = v
			}
		case 4:
			if v, ok := args[1].(string); ok {
				path = v
			}
			if v, ok := args[2].(string); ok {
				domain = v
			}
			if v, ok := args[3].(bool); ok {
				secure = v
			}
		case 5:
			if v, ok := args[1].(string); ok {
				path = v
			}
			if v, ok := args[2].(string); ok {
				domain = v
			}
			if v, ok := args[3].(bool); ok {
				secure = v
			}
			if v, ok := args[4].(bool); ok {
				httpOnly = v
			}
		}
		switch args[0].(type) {
		case int:
			age = int64(args[0].(int))
		case int64:
			age = args[0].(int64)
		case time.Duration:
			age = int64(args[0].(time.Duration))
		}
	}
	if age == 0 {
		// 2^31 - 1 seconds (roughly 2038)
		utctime = time.Unix(2147483647, 0)
	} else {
		utctime = time.Unix(time.Now().Unix()+age, 0)
	}
	return &http.Cookie{
		Name:       name,
		Value:      value,
		Path:       path,
		Domain:     domain,
		Expires:    utctime,
		RawExpires: "",
		MaxAge:     0,
		Secure:     secure,
		HttpOnly:   httpOnly,
		Raw:        "",
		Unparsed:   make([]string, 0),
	}
}

func removeStick(uri string) string {
	uri = strings.TrimRight(uri, "/")
	if uri == "" {
		uri = "/"
	}
	return uri
}

var (
	fieldCache      = make(map[reflect.Type]map[string]int)
	fieldCacheMutex sync.RWMutex
)

// this method cache fields' index to field name
func fieldByName(v reflect.Value, name string) reflect.Value {
	t := v.Type()
	fieldCacheMutex.RLock()
	cache, ok := fieldCache[t]
	fieldCacheMutex.RUnlock()
	if !ok {
		cache = make(map[string]int)
		for i := 0; i < v.NumField(); i++ {
			cache[t.Field(i).Name] = i
		}
		fieldCacheMutex.Lock()
		fieldCache[t] = cache
		fieldCacheMutex.Unlock()
	}

	if i, ok := cache[name]; ok {
		return v.Field(i)
	}

	return reflect.Zero(t)
}

func XsrfName() string {
	return XSRF_TAG
}

func getCookieSig(key string, val []byte, timestamp string) string {
	hm := hmac.New(sha1.New, []byte(key))

	hm.Write(val)
	hm.Write([]byte(timestamp))

	hex := fmt.Sprintf("%02x", hm.Sum(nil))
	return hex
}

func redirect(w http.ResponseWriter, url string, status ...int) error {
	s := 302
	if len(status) > 0 {
		s = status[0]
	}
	w.Header().Set("Location", url)
	w.WriteHeader(s)
	_, err := w.Write([]byte("Redirecting to: " + url))
	return err
}

func Download(w http.ResponseWriter, fpath string) error {
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer f.Close()

	fName := fpath[len(path.Dir(fpath))+1:]
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%v\"", fName))
	_, err = io.Copy(w, f)
	return err
}
