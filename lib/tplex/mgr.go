package tplex

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/coscms/xweb/log"
	"github.com/howeyc/fsnotify"
)

type TemplateMgr struct {
	Caches        map[string][]byte
	mutex         *sync.Mutex
	RootDir       string
	NewRoorDir    string
	Ignores       map[string]bool
	IsReload      bool
	Logger        *log.Logger
	Preprocessor  func([]byte) []byte
	timerCallback func() bool
	TimerCallback func() bool
	initialized   bool
}

func (self *TemplateMgr) Moniter(rootDir string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	//fmt.Println("[xweb] TemplateMgr watcher is start.")
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
						tmpl := ev.Name[len(self.RootDir)+1:]
						content, err := ioutil.ReadFile(ev.Name)
						if err != nil {
							self.Logger.Errorf("loaded template %v failed: %v", tmpl, err)
							break
						}
						self.Logger.Infof("loaded template file %v success", tmpl)
						self.CacheTemplate(tmpl, content)
					}
				} else if ev.IsDelete() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						tmpl := ev.Name[len(self.RootDir)+1:]
						self.CacheDelete(tmpl)
					}
				} else if ev.IsModify() {
					if d.IsDir() {
					} else {
						tmpl := ev.Name[len(self.RootDir)+1:]
						content, err := ioutil.ReadFile(ev.Name)
						if err != nil {
							self.Logger.Errorf("reloaded template %v failed: %v", tmpl, err)
							break
						}

						self.CacheTemplate(tmpl, content)
						self.Logger.Infof("reloaded template %v success", tmpl)
					}
				} else if ev.IsRename() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						tmpl := ev.Name[len(self.RootDir)+1:]
						self.CacheDelete(tmpl)
					}
				}
			case err := <-watcher.Error:
				self.Logger.Error("error:", err)
			case <-time.After(time.Second * 2):
				if self.timerCallback != nil {
					if self.timerCallback() == false {
						close(done)
						return
					}
				}
				//fmt.Printf("TemplateMgr timer operation: %v.\n", time.Now())
			}
		}
	}()

	err = filepath.Walk(self.RootDir, func(f string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return watcher.Watch(f)
		}
		return nil
	})

	if err != nil {
		self.Logger.Error(err.Error())
		return err
	}

	<-done
	//fmt.Println("[xweb] TemplateMgr watcher is closed.")
	return nil
}

func (self *TemplateMgr) CacheAll(rootDir string) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	fmt.Print("Reading the contents of the template files, please wait... ")
	err := filepath.Walk(rootDir, func(f string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		tmpl := f[len(rootDir)+1:]
		tmpl = FixDirSeparator(tmpl)
		if _, ok := self.Ignores[filepath.Base(tmpl)]; !ok {
			fpath := filepath.Join(self.RootDir, tmpl)
			content, err := ioutil.ReadFile(fpath)
			if err != nil {
				self.Logger.Debugf("load template %s error: %v", fpath, err)
				return err
			}
			self.Logger.Debug("loaded template", fpath)
			self.Caches[tmpl] = content
		}
		return nil
	})
	fmt.Println("Complete.")
	return err
}

func (self *TemplateMgr) defaultTimerCallback() func() bool {
	return func() bool {
		if self.TimerCallback != nil {
			return self.TimerCallback()
		}
		//更改模板主题后，关闭当前监控，重新监控新目录
		if self.NewRoorDir == "" || self.NewRoorDir == self.RootDir {
			return true
		}
		self.Caches = make(map[string][]byte)
		self.Ignores = make(map[string]bool)
		self.RootDir = self.NewRoorDir
		go self.Moniter(self.RootDir)
		return false
	}
}

func (self *TemplateMgr) Close() {
	self.TimerCallback = func() bool {
		self.Caches = make(map[string][]byte)
		self.Ignores = make(map[string]bool)
		self.TimerCallback = nil
		return false
	}
	self.initialized = false
}

func (self *TemplateMgr) Init(logger *log.Logger, rootDir string, reload bool) error {
	if self.initialized {
		if rootDir == self.RootDir {
			return nil
		} else {
			self.TimerCallback = func() bool {
				self.Caches = make(map[string][]byte)
				self.Ignores = make(map[string]bool)
				self.TimerCallback = nil
				return false
			}
		}
	} else if !reload {
		self.TimerCallback = func() bool {
			self.TimerCallback = nil
			return false
		}
	}
	self.RootDir = rootDir
	self.Caches = make(map[string][]byte)
	self.Ignores = make(map[string]bool)
	self.mutex = &sync.Mutex{}
	self.Logger = logger
	if dirExists(rootDir) {
		//self.CacheAll(rootDir)
		if reload {
			self.timerCallback = self.defaultTimerCallback()
			go self.Moniter(rootDir)
		}
	}

	if len(self.Ignores) == 0 {
		self.Ignores["*.tmp"] = false
	}
	self.initialized = true
	return nil
}

func (self *TemplateMgr) GetTemplate(tmpl string) ([]byte, error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if content, ok := self.Caches[tmpl]; ok {
		self.Logger.Debugf("load template %v from cache", tmpl)
		return content, nil
	}

	content, err := ioutil.ReadFile(filepath.Join(self.RootDir, tmpl))
	if err == nil {
		self.Logger.Debugf("load template %v from the file:", tmpl)
		self.Caches[tmpl] = content
	}
	return content, err
}

func (self *TemplateMgr) CacheTemplate(tmpl string, content []byte) {
	if self.Preprocessor != nil {
		content = self.Preprocessor(content)
	}
	self.mutex.Lock()
	defer self.mutex.Unlock()
	tmpl = FixDirSeparator(tmpl)
	self.Logger.Debugf("update template %v on cache", tmpl)
	self.Caches[tmpl] = content
	return
}

func (self *TemplateMgr) CacheDelete(tmpl string) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	tmpl = FixDirSeparator(tmpl)
	if _, ok := self.Caches[tmpl]; ok {
		self.Logger.Debugf("delete template %v from cache", tmpl)
		delete(self.Caches, tmpl)
	}
	return
}
