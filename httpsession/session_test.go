package httpsession

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/lunny/csession"
)

type handler struct {
	sm *Manager
}

var testid = 1

func (h *handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	session := h.sm.Session(req, rw)
	t := session.Get("test")
	if t == nil {
		session.Set("test", testid)
		fmt.Println(testid, "new sessionid:", session.id)
	} else {
		fmt.Println(testid, "get sessionid:", session.id, "test:", t)
	}
	testid = testid + 1
	if testid == 3 {
		session.Invalidate(rw)
	}
}

func TestSession(t *testing.T) {
	go func() {
		h := &handler{sm: Default()}
		http.ListenAndServe("0.0.0.0:8333", h)
	}()

	time.Sleep(5 * time.Second)

	session := csession.New()

	for i := 0; i < 5; i++ {
		_, err := session.Get("http://127.0.0.1:8333")
		if err != nil {
			fmt.Println(err)
			return
		}
		time.Sleep(1 * time.Second)
	}

	time.Sleep(5 * time.Second)
}
