/**
 * 模板扩展
 * @author swh <swh@admpub.com>
 */
package tplex

import (
	"bytes"
	"fmt"
	htmlTpl "html/template"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/coscms/xweb/lib/log"
)

func New(logger *log.Logger, templateDir string, cached ...bool) *TemplateEx {
	t := &TemplateEx{
		CachedRelation: make(map[string]*CcRel),
		TemplateDir:    templateDir,
		TemplateMgr:    new(TemplateMgr),
		DelimLeft:      "{{",
		DelimRight:     "}}",
		IncludeTag:     "Include",
		Ext:            ".html",
	}
	mgrCtlLen := len(cached)
	if mgrCtlLen > 0 && cached[0] {
		reloadTemplates := true
		if mgrCtlLen > 1 {
			reloadTemplates = cached[1]
		}
		t.TemplateMgr.OnChangeCallback = func(name, typ, event string) {
			//logger.Infof("OnChange: %v %v %v", name, typ, event)
			//dump(t.CachedRelation)
			switch event {
			case "create":
			case "delete", "modify", "rename":
				if typ == "dir" {
					return
				}
				if cs, ok := t.CachedRelation[name]; ok {
					for key, _ := range cs.Rel {
						if name == key {
							logger.Infof("remove cached template object: %v", key)
							continue
						}
						if _, ok := t.CachedRelation[key]; ok {
							logger.Infof("remove cached template object: %v", key)
							delete(t.CachedRelation, key)
						}
					}
					delete(t.CachedRelation, name)
				}
			}
		}
		t.TemplateMgr.Init(logger, templateDir, reloadTemplates)
	}
	return t
}

type CcRel struct {
	Rel map[string]uint8
	Tpl *htmlTpl.Template
}

type TemplateEx struct {
	CachedRelation   map[string]*CcRel
	TemplateDir      string
	TemplateMgr      *TemplateMgr
	BeforeRender     func(*string)
	DelimLeft        string
	DelimRight       string
	incTagRegex      *regexp.Regexp
	cachedRegexIdent string
	IncludeTag       string
	Ext              string
}

func (self *TemplateEx) Fetch(tmplName string, fn func() htmlTpl.FuncMap, values interface{}) string {
	tmplName += self.Ext
	var tmpl *htmlTpl.Template
	cv, ok := self.CachedRelation[tmplName]
	if !ok {
		b, err := self.RawContent(tmplName)
		if err != nil {
			return fmt.Sprintf("RenderTemplate %v read err: %s", tmplName, err)
		}

		content := string(b)
		if self.BeforeRender != nil {
			self.BeforeRender(&content)
		}
		subcs := make(map[string]string, 0) //子模板内容

		ident := self.DelimLeft + self.IncludeTag + self.DelimRight
		if self.cachedRegexIdent != ident || self.incTagRegex == nil {
			self.incTagRegex = regexp.MustCompile(regexp.QuoteMeta(self.DelimLeft) + self.IncludeTag + `[\s]+"([^"]+)"(?:[\s]+([^` + regexp.QuoteMeta(self.DelimRight[0:1]) + `]+))?[\s]*` + regexp.QuoteMeta(self.DelimRight))
		}

		content = self.ContainsSubTpl(content, &subcs)
		//fmt.Println("====>", content)
		t := htmlTpl.New(tmplName)
		t.Delims(self.DelimLeft, self.DelimRight)
		var funcMap htmlTpl.FuncMap
		if fn != nil {
			funcMap = fn()
			t.Funcs(funcMap)
		}

		tmpl, err = t.Parse(content)
		if err != nil {
			return fmt.Sprintf("Parse %v err: %v", tmplName, err)
		}
		for name, subc := range subcs {
			if _, ok := self.CachedRelation[name]; ok {
				self.CachedRelation[name].Rel[tmplName] = 0
				continue
			}
			//fmt.Println("====>", name)
			var t *htmlTpl.Template
			if tmpl == nil {
				tmpl = htmlTpl.New(name)
				tmpl.Delims(self.DelimLeft, self.DelimRight)
				tmpl.Funcs(funcMap)
			}
			if name == tmpl.Name() {
				t = tmpl
			} else {
				t = tmpl.New(name)
			}
			if self.BeforeRender != nil {
				self.BeforeRender(&subc)
			}
			_, err = t.Parse(subc)
			if err != nil {
				return fmt.Sprintf("Parse %v err: %v", name, err)
			}
			self.CachedRelation[name] = &CcRel{
				Rel: map[string]uint8{tmplName: 0},
				Tpl: t,
			}
		}
		self.CachedRelation[tmplName] = &CcRel{
			Rel: map[string]uint8{tmplName: 0},
			Tpl: tmpl,
		}
	} else {
		tmpl = cv.Tpl
	}
	return self.Parse(tmpl, values)
}

func (self *TemplateEx) ContainsSubTpl(content string, subcs *map[string]string) string {
	matches := self.incTagRegex.FindAllStringSubmatch(content, -1)
	//dump(matches)
	for _, v := range matches {
		matched := v[0]
		tmplFile := v[1]
		passObject := v[2]
		tmplFile += self.Ext
		if _, ok := (*subcs)[tmplFile]; !ok {
			if _, ok := self.CachedRelation[tmplFile]; ok {
				(*subcs)[tmplFile] = ""
				continue
			}
			b, err := self.RawContent(tmplFile)
			if err != nil {
				return fmt.Sprintf("RenderTemplate %v read err: %s", tmplFile, err)
			}
			str := string(b)
			str = self.ContainsSubTpl(str, subcs)
			sub := self.Tag(`define "`+tmplFile+`"`) + str + self.Tag(`end`)
			(*subcs)[tmplFile] = sub
		}
		if passObject == "" {
			passObject = "."
		}
		content = strings.Replace(content, matched, self.Tag(`template "`+tmplFile+`" `+passObject), -1)
	}
	return content
}

func (self *TemplateEx) Tag(content string) string {
	return self.DelimLeft + content + self.DelimRight
}

// Include method provide to template for {{include "about"}}
func (self *TemplateEx) Include(tmplName string, fn func() htmlTpl.FuncMap, values interface{}) interface{} {
	return htmlTpl.HTML(self.Fetch(tmplName, fn, values))
}

func (self *TemplateEx) Parse(tmpl *htmlTpl.Template, values interface{}) string {
	newbytes := bytes.NewBufferString("")
	err := tmpl.Execute(newbytes, values)
	if err != nil {
		return fmt.Sprintf("Parse %v err: %v", tmpl.Name(), err)
	}

	b, err := ioutil.ReadAll(newbytes)
	if err != nil {
		return fmt.Sprintf("Parse %v err: %v", tmpl.Name(), err)
	}
	return string(b)
}

func (self *TemplateEx) RawContent(tmpl string) ([]byte, error) {
	if self.TemplateMgr != nil && self.TemplateMgr.Caches != nil {
		return self.TemplateMgr.GetTemplate(tmpl)
	}
	return ioutil.ReadFile(filepath.Join(self.TemplateDir, tmpl))
}
