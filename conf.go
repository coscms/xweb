package webx

func NewCONF() *CONF {
	return &CONF{
		Bool:      make(map[string]bool),
		Interface: make(map[string]interface{}),
		String:    make(map[string]string),
		Int:       make(map[string]int64),
		Float:     make(map[string]float64),
		Byte:      make(map[string][]byte),
		Conf:      make(map[string]*CONF),
	}
}

type CONF struct {
	Bool      map[string]bool
	Interface map[string]interface{}
	String    map[string]string
	Int       map[string]int64
	Float     map[string]float64
	Byte      map[string][]byte
	Conf      map[string]*CONF
}

func (a *CONF) GetConf(k string) *CONF {
	if a.Conf == nil {
		a.InitConf()
	}
	if v, ok := a.Conf[k]; ok {
		return v
	}
	a.Conf[k] = &CONF{}
	a.Conf[k].Init()
	return a.Conf[k]
}

func (a *CONF) SetConf(k string, v *CONF) {
	if a.Conf == nil {
		a.InitConf()
	}
	a.Conf[k] = v
}

func (a *CONF) DelConf(k string) {
	if a.Conf == nil {
		a.InitConf()
	}
	if _, ok := a.Conf[k]; ok {
		delete(a.Conf, k)
	}
}

func (a *CONF) GetBool(k string) bool {
	if v, ok := a.Bool[k]; ok {
		return v
	}
	return false
}

func (a *CONF) SetBool(k string, v bool) {
	a.Bool[k] = v
}

func (a *CONF) DelBool(k string) {
	if _, ok := a.Bool[k]; ok {
		delete(a.Bool, k)
	}
}

func (a *CONF) GetInterface(k string) interface{} {
	if v, ok := a.Interface[k]; ok {
		return v
	}
	return nil
}

func (a *CONF) SetInterface(k string, v interface{}) {
	a.Interface[k] = v
}

func (a *CONF) DelInterface(k string) {
	if _, ok := a.Interface[k]; ok {
		delete(a.Interface, k)
	}
}

func (a *CONF) GetString(k string) string {
	if v, ok := a.String[k]; ok {
		return v
	}
	return ""
}

func (a *CONF) SetString(k string, v string) {
	a.String[k] = v
}

func (a *CONF) DelString(k string) {
	if _, ok := a.String[k]; ok {
		delete(a.String, k)
	}
}

func (a *CONF) GetInt(k string) int64 {
	if v, ok := a.Int[k]; ok {
		return v
	}
	return 0
}

func (a *CONF) SetInt(k string, v int64) {
	a.Int[k] = v
}

func (a *CONF) DelInt(k string) {
	if _, ok := a.Int[k]; ok {
		delete(a.Int, k)
	}
}

func (a *CONF) GetFloat(k string) float64 {
	if v, ok := a.Float[k]; ok {
		return v
	}
	return 0.0
}

func (a *CONF) SetFloat(k string, v float64) {
	a.Float[k] = v
}

func (a *CONF) DelFloat(k string) {
	if _, ok := a.Float[k]; ok {
		delete(a.Float, k)
	}
}

func (a *CONF) GetByte(k string) []byte {
	if v, ok := a.Byte[k]; ok {
		return v
	}
	return []byte{}
}

func (a *CONF) SetByte(k string, v []byte) {
	a.Byte[k] = v
}

func (a *CONF) DelByte(k string) {
	if _, ok := a.Byte[k]; ok {
		delete(a.Byte, k)
	}
}

func (a *CONF) Init() {
	a.Bool = make(map[string]bool)
	a.Interface = make(map[string]interface{})
	a.String = make(map[string]string)
	a.Int = make(map[string]int64)
	a.Float = make(map[string]float64)
	a.Byte = make(map[string][]byte)
	a.Conf = make(map[string]*CONF)
}

func (a *CONF) InitBool() {
	a.Bool = make(map[string]bool)
}
func (a *CONF) InitInterface() {
	a.Interface = make(map[string]interface{})
}
func (a *CONF) InitString() {
	a.String = make(map[string]string)
}
func (a *CONF) InitInt() {
	a.Int = make(map[string]int64)
}
func (a *CONF) InitFloat() {
	a.Float = make(map[string]float64)
}
func (a *CONF) InitByte() {
	a.Byte = make(map[string][]byte)
}

func (a *CONF) InitConf() {
	a.Conf = make(map[string]*CONF)
}
