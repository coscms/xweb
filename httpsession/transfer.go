package httpsession

import (
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Transfer provide and set sessionid
type Transfer interface {
	Get(req *http.Request) (Id, error)
	Set(req *http.Request, rw http.ResponseWriter, id Id)
	Clear(rw http.ResponseWriter)
}

// CookieRetriever provide sessionid from cookie
type CookieTransfer struct {
	name   string
	expire time.Duration
	lock   sync.Mutex
}

func NewCookieTransfer(name string, expire time.Duration) *CookieTransfer {
	return &CookieTransfer{name: name, expire: expire}
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
			MaxAge:   int(transfer.expire / time.Second),
			Path:     "/",
			HttpOnly: true,
			//Secure:   true,
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
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now(),
		MaxAge:   -1,
	}
	http.SetCookie(rw, &cookie)
}

var _ Transfer = NewCookieTransfer("test", DefaultExpireTime)

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
