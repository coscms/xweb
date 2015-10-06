package xweb

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/howeyc/fsnotify"
)

var DefaultStaticVerMgr *StaticVerMgr = new(StaticVerMgr)

type StaticVerMgr struct {
	Caches        map[string]string
	Combined      map[string][]string
	Combines      map[string]bool
	mutex         *sync.Mutex
	Path          string
	Ignores       map[string]bool
	app           *App
	timerCallback func() bool
	TimerCallback func() bool
	initialized   bool
}

func (self *StaticVerMgr) Moniter(staticPath string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	done := make(chan bool)
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				if ev == nil {
					break
				}
				if _, ok := self.Ignores[filepath.Base(ev.Name)]; ok {
					break
				}
				d, err := os.Stat(ev.Name)
				if err != nil {
					break
				}

				if ev.IsCreate() {
					if d.IsDir() {
						watcher.Watch(ev.Name)
					} else {
						url := ev.Name[len(self.Path)+1:]
						self.CacheItem(url)
					}
				} else if ev.IsDelete() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						url := ev.Name[len(self.Path)+1:]
						self.CacheDelete(url)
						self.DeleteCombined(url)
					}
				} else if ev.IsModify() {
					if d.IsDir() {
					} else {
						url := ev.Name[len(staticPath)+1:]
						self.CacheItem(url)
						self.DeleteCombined(url)
					}
				} else if ev.IsRename() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						url := ev.Name[len(staticPath)+1:]
						self.CacheDelete(url)
						self.DeleteCombined(url)
					}
				}
			case err := <-watcher.Error:
				self.app.Errorf("error: %v", err)
			case <-time.After(time.Second * 2):
				if self.timerCallback != nil {
					if self.timerCallback() == false {
						close(done)
						return
					}
				}
			}
		}
	}()

	err = filepath.Walk(staticPath, func(f string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return watcher.Watch(f)
		}
		return nil
	})

	if err != nil {
		fmt.Println(err)
		return err
	}

	<-done
	return nil
}

func (self *StaticVerMgr) defaultTimerCallback() func() bool {
	return func() bool {
		if self.TimerCallback != nil {
			return self.TimerCallback()
		}
		//更改模板主题后，关闭当前监控，重新监控新目录
		if self.app.AppConfig.StaticDir == self.Path {
			return true
		}
		for f, _ := range self.Combines {
			os.Remove(self.Path + f)
		}
		self.Caches = make(map[string]string)
		self.Combined = make(map[string][]string)
		self.Combines = make(map[string]bool)
		self.Path = self.app.AppConfig.StaticDir
		go self.Moniter(self.Path)
		return false
	}
}

func (self *StaticVerMgr) Close() {
	self.TimerCallback = func() bool {
		for f, _ := range self.Combines {
			os.Remove(self.Path + f)
		}
		self.Caches = make(map[string]string)
		self.Combined = make(map[string][]string)
		self.Combines = make(map[string]bool)
		self.TimerCallback = nil
		return false
	}
	self.initialized = false
}

func (self *StaticVerMgr) Init(app *App, staticPath string) error {
	if self.initialized {
		if staticPath == self.Path {
			return nil
		} else {
			self.TimerCallback = func() bool {
				for f, _ := range self.Combines {
					os.Remove(self.Path + f)
				}
				self.Caches = make(map[string]string)
				self.Combined = make(map[string][]string)
				self.Combines = make(map[string]bool)
				self.TimerCallback = nil
				return false
			}
		}
	}
	self.Path = staticPath
	self.Caches = make(map[string]string)
	self.Combined = make(map[string][]string)
	self.Combines = make(map[string]bool)
	self.mutex = &sync.Mutex{}
	self.Ignores = map[string]bool{".DS_Store": true}
	self.app = app

	if dirExists(staticPath) {
		//self.CacheAll(staticPath)
		self.timerCallback = self.defaultTimerCallback()
		go self.Moniter(staticPath)
	}
	self.initialized = true
	return nil
}

func (self *StaticVerMgr) getFileVer(url string) string {
	//content, err := ioutil.ReadFile(path.Join(self.Path, url))
	fPath := filepath.Join(self.Path, url)
	self.app.Debug("loaded static ", fPath)
	f, err := os.Open(fPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	fInfo, err := f.Stat()
	if err != nil {
		return ""
	}

	content := make([]byte, int(fInfo.Size()))
	_, err = f.Read(content)
	if err == nil {
		h := md5.New()
		io.WriteString(h, string(content))
		return fmt.Sprintf("%x", h.Sum(nil))[0:4]
	}
	return ""
}

func (self *StaticVerMgr) CacheAll(staticPath string) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	fmt.Print("Getting static file version number, please wait... ")
	err := filepath.Walk(staticPath, func(f string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		rp := f[len(staticPath)+1:]
		rp = FixDirSeparator(rp)
		if _, ok := self.Ignores[filepath.Base(rp)]; !ok {
			self.Caches[rp] = self.getFileVer(rp)
		}
		return nil
	})
	fmt.Println("Complete.")
	return err
}

func (self *StaticVerMgr) GetVersion(url string) string {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if ver, ok := self.Caches[url]; ok {
		return ver
	}

	ver := self.getFileVer(url)
	if ver != "" {
		self.Caches[url] = ver
	}
	return ver
}

func (self *StaticVerMgr) CacheDelete(url string) {
	url = FixDirSeparator(url)
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if _, ok := self.Caches[url]; ok {
		delete(self.Caches, url)
		self.app.Infof("static file %s is deleted.\n", url)
	}
}

func (self *StaticVerMgr) CacheItem(url string) {
	url = FixDirSeparator(url)
	ver := self.getFileVer(url)
	if ver != "" {
		self.mutex.Lock()
		defer self.mutex.Unlock()
		self.Caches[url] = ver
		self.app.Infof("static file %s is created.", url)
	}
}

func (self *StaticVerMgr) DeleteCombined(url string) {
	url = "/" + FixDirSeparator(url)
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if val, ok := self.Combined[url]; ok {
		for _, v := range val {
			if _, has := self.Combines[v]; !has {
				continue
			}
			err := os.Remove(self.Path + v)
			delete(self.Combines, v)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func (self *StaticVerMgr) RecordCombined(fromUrl string, combineUrl string) {
	if self.Combined == nil {
		return
	}
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if _, ok := self.Combined[fromUrl]; !ok {
		self.Combined[fromUrl] = make([]string, 0)
	}
	self.Combined[fromUrl] = append(self.Combined[fromUrl], combineUrl)
}

func (self *StaticVerMgr) RecordCombines(combineUrl string) {
	if self.Combines == nil {
		return
	}
	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.Combines[combineUrl] = true
}

func (self *StaticVerMgr) IsCombined(combineUrl string) (ok bool) {
	if self.Combines == nil {
		return
	}
	self.mutex.Lock()
	defer self.mutex.Unlock()
	_, ok = self.Combines[combineUrl]
	return
}
