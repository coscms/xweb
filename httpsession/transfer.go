package httpsession

import (
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Transfer provide and set sessionid
type Transfer interface {
	SetMaxAge(maxAge time.Duration)
	Get(req *http.Request) (Id, error)
	Set(req *http.Request, rw http.ResponseWriter, id Id)
	Clear(rw http.ResponseWriter)
}

// CookieRetriever provide sessionid from cookie
type CookieTransfer struct {
	name     string
	maxAge   time.Duration
	lock     sync.Mutex
	secure   bool
	rootPath string
	domain   string
}

func NewCookieTransfer(name string, maxAge time.Duration, secure bool, rootPath string) *CookieTransfer {
	return &CookieTransfer{
		name:     name,
		maxAge:   maxAge,
		secure:   secure,
		rootPath: rootPath,
	}
}

func (transfer *CookieTransfer) SetMaxAge(maxAge time.Duration) {
	transfer.maxAge = maxAge
}

func (transfer *CookieTransfer) Get(req *http.Request) (Id, error) {
	cookie, err := req.Cookie(transfer.name)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}
		return "", err
	}
	if cookie.Value == "" {
		return Id(""), nil
	}
	id, _ := url.QueryUnescape(cookie.Value)
	//id := cookie.Value
	return Id(id), nil
}

func (transfer *CookieTransfer) Set(req *http.Request, rw http.ResponseWriter, id Id) {
	sid := url.QueryEscape(string(id))
	transfer.lock.Lock()
	defer transfer.lock.Unlock()
	//sid := string(id)
	cookie, _ := req.Cookie(transfer.name)
	if cookie == nil {
		cookie = &http.Cookie{
			Name:     transfer.name,
			Value:    sid,
			Path:     transfer.rootPath,
			Domain:   transfer.domain,
			HttpOnly: true,
			Secure:   transfer.secure,
		}
		if transfer.maxAge > 0 {
			cookie.Expires = time.Now().Add(transfer.maxAge)
		}

		req.AddCookie(cookie)
	} else {
		cookie.Value = sid
	}
	http.SetCookie(rw, cookie)
}

func (transfer *CookieTransfer) Clear(rw http.ResponseWriter) {
	cookie := http.Cookie{
		Name:     transfer.name,
		Path:     transfer.rootPath,
		Domain:   transfer.domain,
		HttpOnly: true,
		Secure:   transfer.secure,
		Expires:  time.Date(0, 1, 1, 0, 0, 0, 0, time.Local),
		MaxAge:   -1,
	}
	http.SetCookie(rw, &cookie)
}

var _ Transfer = NewCookieTransfer("test", 0, false, "/")

// CookieRetriever provide sessionid from url
/*type UrlTransfer struct {
}

func NewUrlTransfer() *UrlTransfer {
	return &UrlTransfer{}
}

func (transfer *UrlTransfer) Get(req *http.Request) (string, error) {
	return "", nil
}

func (transfer *UrlTransfer) Set(rw http.ResponseWriter, id Id) {

}

var (
	_ Transfer = NewUrlTransfer()
)
*/
