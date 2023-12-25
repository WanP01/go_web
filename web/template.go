package web

import (
	"bytes"
	"html/template"
	"io/fs"
)

type TemplateEngine interface {
	// Render 渲染页面
	// data 是渲染页面所需要的数据
	Render(telname string, data any) ([]byte, error)
}

type GOTemplateEngine struct {
	T *template.Template
	// 也可以考虑设计为 map[string]*template.Template
	// 但是其实没太大必要，因为 template.Template 本身就提供了按名索引的功能
	//ParseGlob,ParseFiles可以提供解析多个template的集合，然后用ExecuteTemplate()指定template解析
}

func (g *GOTemplateEngine) Render(telname string, data any) ([]byte, error) {
	res := &bytes.Buffer{}
	err := g.T.ExecuteTemplate(res, telname, data)
	return res.Bytes(), err
}

// LoadFromGlob 按照模式导入解析模板
func (g *GOTemplateEngine) LoadFromGlob(pattern string) error {
	var err error
	g.T, err = template.ParseGlob(pattern)
	return err
}

// LoadFromFiles 按照文件名导入并解析模板
func (g *GOTemplateEngine) LoadFromFiles(filenames ...string) error {
	var err error
	g.T, err = template.ParseFiles(filenames...)
	return err
}

// LoadFromFS  按照文件系统导入并解析模板
func (g *GOTemplateEngine) LoadFromFS(fs fs.FS, patterns ...string) error {
	var err error
	g.T, err = template.ParseFS(fs, patterns...)
	return err
}
