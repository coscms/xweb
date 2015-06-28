package xweb

type XsrfManager interface {
	Init(*Action)
	Get(key string) string
	Set(key, val string)
	Valid(key, val string) bool
}

var DefaultXsrfManager XsrfManager = &XsrfCookieStorage{}

type XsrfCookieStorage struct {
	*Action
}

func (c *XsrfCookieStorage) Get(key string) string {
	var val string
	if res, err := c.Request.Cookie(key); err == nil && res.Value != "" {
		val = c.App.Cryptor.Decode(res.Value, c.App.AppConfig.CookieSecret)
	}
	return val
}

func (c *XsrfCookieStorage) Set(key, val string) {
	val = c.App.Cryptor.Encode(val, c.App.AppConfig.CookieSecret)
	c.SetCookie(NewCookie(key, val, c.App.AppConfig.SessionTimeout, "", "", false, true))
}

func (c *XsrfCookieStorage) Valid(key, val string) bool {
	return c.Get(key) == val
}

func (c *XsrfCookieStorage) Init(a *Action) {
	if c.Action == nil {
		c.Action = a
	}
}

type XsrfSessionStorage struct {
	*Action
}

func (c *XsrfSessionStorage) Get(key string) string {
	return c.Session().Get(key).(string)
}

func (c *XsrfSessionStorage) Set(key, val string) {
	c.Session().Set(key, val)
}

func (c *XsrfSessionStorage) Valid(key, val string) bool {
	return c.Get(key) == val
}

func (c *XsrfSessionStorage) Init(a *Action) {
	if c.Action == nil {
		c.Action = a
	}
}
