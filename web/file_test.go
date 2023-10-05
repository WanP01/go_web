package web

import (
	"bytes"
	"html/template"
	"mime/multipart"
	"path/filepath"
	"testing"
)

func TestFileUp(t *testing.T) {
	s := NewServerEngine("test")
	s.Get("/upload_page", func(ctx *Context) {
		tpl := template.New("upload")
		tpl, err := tpl.Parse(`
<html>
<body>
	<form action="/upload" method="post" enctype="multipart/form-data">
		 <input type="file" name="myfile" />
		 <button type="submit">上传</button>
	</form>
</body>
<html>
`)
		if err != nil {
			t.Fatal(err)
		}

		page := &bytes.Buffer{}
		err = tpl.Execute(page, nil)
		if err != nil {
			t.Fatal(err)
		}

		ctx.RespStatusCode = 200
		ctx.RespData = page.Bytes()
	})

	s.Post("/upload", NewFileUpLoader("myfile", WithDefaultFileUpLoader(func(fh *multipart.FileHeader) string {
		return filepath.Join("C:\\Users\\wp199\\Desktop\\go_pro\\go_web\\testdata", "upload", fh.Filename)
	})).Handle())
	s.Start(":8081")
}

func TestFileDownloader_Handle(t *testing.T) {
	s := NewServerEngine("test")
	s.Get("/download", (&FileDownloader{
		// 下载的文件所在目录
		Dir: "C:\\Users\\wp199\\Desktop\\go_pro\\go_web\\testdata\\download",
	}).Handle())
	// 在浏览器里面输入 localhost:8081/download?file=test.txt
	s.Start(":8081")
}

func TestStaticResourceHandler_Handle(t *testing.T) {
	s := NewServerEngine("test")
	handler := NewStaticResourceHandler()
	s.Get("/img/:file", handler.handle())
	// 在浏览器里面输入 localhost:8081/img/come_on_baby.jpg
	s.Start(":8081")
}
