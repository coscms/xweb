package tplex

import (
	"bytes"
	"encoding/json"
	"fmt"
	htmlTpl "html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/coscms/xweb/log"
)

func New(logger *log.Logger, templateDir string, cached ...bool) *TempateEx {
	t := &TempateEx{
		CachedTemplate: make(map[string]*htmlTpl.Template),
		CachedRelation: make(map[string]string),
		TemplateDir:    templateDir,
		TemplateMgr:    new(TemplateMgr),
		DelimLeft:      "{{",
		DelimRight:     "}}",
	}
	mgrCtlLen := len(cached)
	if mgrCtlLen > 0 && cached[0] {
		reloadTemplates := true
		if mgrCtlLen > 1 {
			reloadTemplates = cached[1]
		}
		t.TemplateMgr.Init(logger, templateDir, reloadTemplates)
	}
	return t
}

type TempateEx struct {
	CachedTemplate map[string]*htmlTpl.Template
	CachedRelation map[string]string
	TemplateDir    string
	TemplateMgr    *TemplateMgr
	BeforeRender   func(*string)
	DelimLeft      string
	DelimRight     string
	incTagRegex    *regexp.Regexp
}

func (self *TempateEx) Fetch(tmplName string, funcMap htmlTpl.FuncMap, values interface{}) interface{} {
	tmpl, ok := self.CachedTemplate[tmplName]
	if !ok {
		b, err := self.RawContent(tmplName)
		if err != nil {
			return fmt.Sprintf("RenderTemplate %v read err: %s", tmplName, err)
		}

		content := string(b)
		if self.BeforeRender != nil {
			self.BeforeRender(&content)
		}

		content = self.ContainsSubTpl(content)

		t := htmlTpl.New(tmplName)
		t.Delims(self.DelimLeft, self.DelimRight)
		t.Funcs(funcMap)

		tmpl, err = t.Parse(content)
		if err != nil {
			return fmt.Sprintf("Parse %v err: %v", tmplName, err)
		}
		self.CachedTemplate[tmplName] = tmpl
		self.CachedRelation[tmplName] = tmplName
	}
	return self.Parse(tmpl, values)
}

func (self *TempateEx) ContainsSubTpl(content string) string {
	self.incTagRegex = regexp.MustCompile(regexp.QuoteMeta(self.DelimLeft) + `Include "([^"]+)"` + regexp.QuoteMeta(self.DelimRight))
	matches := self.incTagRegex.FindAllStringSubmatch(content, -1)
	c, _ := json.MarshalIndent(matches, "", " ")
	fmt.Println(string(c))
	subContent := ""
	for _, v := range matches {
		tmplFile := v[1]
		b, err := self.RawContent(tmplFile)
		if err != nil {
			return fmt.Sprintf("RenderTemplate %v read err: %s", tmplFile, err)
		}
		str := string(b)
		subContent += self.Tag(`define "`+tmplFile+`"`) + self.ContainsSubTpl(str) + self.Tag(`end`)
		content = strings.Replace(content, v[0], self.Tag(`template "`+tmplFile+`"`), -1)
	}
	content = subContent + content
	return content
}

func (self *TempateEx) Tag(content string) string {
	return self.DelimLeft + content + self.DelimRight
}

// Include method provide to template for {{include "xx.tmpl"}}
func (self *TempateEx) Include(tmplName string, funcMap htmlTpl.FuncMap, values interface{}) interface{} {
	tmpl, ok := self.CachedTemplate[tmplName]
	if !ok {
		b, err := self.RawContent(tmplName)
		if err != nil {
			return fmt.Sprintf("RenderTemplate %v read err: %s", tmplName, err)
		}

		content := string(b)
		if self.BeforeRender != nil {
			self.BeforeRender(&content)
		}

		t := htmlTpl.New(tmplName)
		t.Delims(self.DelimLeft, self.DelimRight)
		t.Funcs(funcMap)

		tmpl, err = t.Parse(content)
		if err != nil {
			return fmt.Sprintf("Parse %v err: %v", tmplName, err)
		}
		self.CachedTemplate[tmplName] = tmpl
		self.CachedRelation[tmplName] = tmplName
	}
	return self.Parse(tmpl, values)
}

func (self *TempateEx) Parse(tmpl *htmlTpl.Template, values interface{}) interface{} {
	newbytes := bytes.NewBufferString("")
	err := tmpl.Execute(newbytes, values)
	if err != nil {
		return fmt.Sprintf("Parse %v err: %v", tmpl.Name(), err)
	}

	b, err := ioutil.ReadAll(newbytes)
	if err != nil {
		return fmt.Sprintf("Parse %v err: %v", tmpl.Name(), err)
	}
	return htmlTpl.HTML(string(b))
}

func (self *TempateEx) RawContent(tmpl string) ([]byte, error) {
	if self.TemplateMgr != nil && self.TemplateMgr.Caches != nil {
		return self.TemplateMgr.GetTemplate(tmpl)
	}
	return ioutil.ReadFile(filepath.Join(self.TemplateDir, tmpl))
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

func FixDirSeparator(dir string) string {
	if runtime.GOOS == "windows" {
		return strings.Replace(dir, "\\", "/", -1)
	}
	return dir
}

func fileExists(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}

	return !info.IsDir()
}
