package web

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	lru "github.com/hashicorp/golang-lru"
)

//>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

type FileUpLoader struct {
	// FileField 对应于文件在表单中的字段名字
	FileField string
	// DstPathFunc 用于计算目标路径(可以用户自定义，也可以用默认值)
	DstPathFunc func(fh *multipart.FileHeader) string
}

// 选项模式设置默认值
type DstOPT func(f *FileUpLoader)

func NewFileUpLoader(FileField string, OPT ...DstOPT) *FileUpLoader {
	f := &FileUpLoader{
		FileField: FileField, // FileFiled 由前端决定
		DstPathFunc: func(fh *multipart.FileHeader) string {
			return filepath.Join("testdata", "upload", fh.Filename)
		}}

	for _, opt := range OPT {
		opt(f)
	}

	return f
}

// 更换默认目的文件地址函数
func WithDefaultFileUpLoader(dst func(fh *multipart.FileHeader) string) DstOPT {
	return func(f *FileUpLoader) {
		f.DstPathFunc = dst
	}
}

func (f *FileUpLoader) Handle() HandleFunc {
	return func(ctx *Context) {
		//获得上传文件
		srcFile, srcHeader, err := ctx.R.FormFile(f.FileField)
		if err != nil {
			ctx.RespData = []byte("上传失败，未找到数据" + err.Error())
			ctx.RespStatusCode = http.StatusBadRequest //400 客户端请求报文语法错误
			//log.Fatalln(err)
			return
		}
		defer srcFile.Close()
		//获得目标地址，并创建和打开对应文件
		dstPath := f.DstPathFunc(srcHeader)

		//注意创建沿途的dir
		//os.MkdirAll()
		dstFile, err := os.OpenFile(dstPath, os.O_APPEND|os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o666)
		if err != nil {
			ctx.RespData = []byte("上传失败" + err.Error())
			ctx.RespStatusCode = http.StatusInternalServerError //500 服务端内部错误
			//log.Fatalln(err)
			return
		}
		defer dstFile.Close()
		//保存到目标地址
		_, err = io.CopyBuffer(dstFile, srcFile, nil)
		if err != nil {
			ctx.RespData = []byte("上传失败" + err.Error())
			ctx.RespStatusCode = http.StatusInternalServerError //500 服务端内部错误
			//log.Fatalln(err)
			return
		}
		//上传成功返回提示即可
		ctx.RespData = []byte("上传成功" + fmt.Sprintf("%v", dstPath))
		ctx.RespStatusCode = http.StatusOK
	}
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
// FileDownloader 直接操作了 http.ResponseWriter
// 所以在 Middleware 里面将不能使用 RespData
// 因为没有赋值
type FileDownloader struct {
	Dir string //下载文件的地址（需要下载的文件存放位置）
}

// 假定下载地址的格式是restfull ：..... /XXXX?file=xxxx
func (f *FileDownloader) Handle() HandleFunc {
	return func(ctx *Context) {
		//拿到要下载的文件名字
		reqFile, _ := ctx.QueryValue("file").ToString()
		//拼接下载地址
		// filepath.Clean（）会消除不安全的路径，比如 ../../xxx.go 这种会被客户拿到不该传出的文件
		path := filepath.Join(f.Dir, filepath.Clean(reqFile))
		fn := filepath.Base(path)
		//构建Response HTTP header 包
		header := ctx.W.Header()
		header.Set("Content-Disposition", "attachment;filename="+fn)
		header.Set("Content-Description", "File Transfer")
		header.Set("Content-Type", "application/octet-stream")
		header.Set("Content-Transfer-Encoding", "binary")
		header.Set("Expires", "0")
		header.Set("Cache-Control", "must-revalidate")
		header.Set("Pragma", "public")

		//ServeFile用指定文件或目录的内容响应请求。
		//如果提供的文件或目录名是一个相对路径，它将相对于当前目录进行解释，并可能上升到父目录。如果提供的名称是根据用户输入构造的，则应该在调用ServeFile之前对其进行清理。
		http.ServeFile(ctx.W, ctx.R, path)
	}
}

//>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

// option模式更改默认值
type StaticResourceHandlerOption func(s *StaticResourceHandler)

// 静态资源请求
// 通过缓存数量与每个缓存大小控制避免大文件缓存
type StaticResourceHandler struct {
	dir                     string            //文件存储目录
	extensionContentTypeMap map[string]string //根据不同文件尾缀填写不同文件header content-type
	cache                   *lru.Cache        //文件缓存
	maxFileSize             int               //控制文件缓存大小小于maxFileSzie
}

// 缓存的文件信息
type fileCacheItem struct {
	fileName    string //缓存文件名
	fileSize    int    //缓存文件大小
	contentType string //缓存文件type
	data        []byte //缓存文件
}

func NewStaticResourceHandler(options ...StaticResourceHandlerOption) *StaticResourceHandler {
	c, _ := lru.New(1000) //缓存键值对数目小于1000
	res := &StaticResourceHandler{
		dir: "C:\\Users\\wp199\\Desktop\\go_pro\\go_web\\testdata\\img", //静态文件存放地址
		extensionContentTypeMap: map[string]string{
			// 这里根据自己的需要不断添加
			"jpeg": "image/jpeg",
			"jpe":  "image/jpeg",
			"jpg":  "image/jpeg",
			"png":  "image/png",
			"pdf":  "image/pdf",
		},
		cache: c,
	}

	for _, opt := range options {
		opt(res)
	}
	return res
}

func WithMoreExtension(extMap map[string]string) StaticResourceHandlerOption {
	return func(s *StaticResourceHandler) {
		for ext, contentType := range extMap {
			s.extensionContentTypeMap[ext] = contentType
		}
	}
}

func WithSetDir(dir string) StaticResourceHandlerOption {
	return func(s *StaticResourceHandler) {
		s.dir = dir
	}
}

// 静态文件有固定存放地点，http URL 格式统一化，注册router方式 img/:file (参数路径匹配)
// 静态文件请求地址格式举例 xxxxx/img/come_on_bady.jpg

func (s *StaticResourceHandler) handle() HandleFunc {
	return func(ctx *Context) {
		reqFile, _ := ctx.PathValue("file").ToString() //获取文件名：注册router方式 img/:file (参数路径匹配)，所以用pathValue取

		//先看缓存有没有
		if item, ok := s.readFileFromData(reqFile); ok {
			log.Printf("从缓存中读取数据...")
			s.writeItemAsResponse(item, ctx.W) //存在就直接写回
			return
		}

		//打开对应文件
		path := filepath.Join(s.dir, reqFile) //拼接文件存放地址
		fdata, err := os.ReadFile(path)       //打开对应文件,获取数据
		fmt.Printf("%v", fdata)
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError // 500,服务器内部资源问题
			//log.Fatalln(err)
			return
		}

		//获取文件尾缀，好写回http header content-type
		ext := getFileExt(reqFile)
		t, ok := s.extensionContentTypeMap[ext]
		if !ok {
			ctx.RespStatusCode = http.StatusBadRequest
			return
		}

		//保存缓存记录
		filecache := &fileCacheItem{
			fileName:    reqFile,
			fileSize:    len(fdata),
			contentType: t,
			data:        fdata,
		}
		if len(fdata) < s.maxFileSize { //缓存文件大小过大避免缓存
			s.cache.Add(reqFile, filecache)
		}

		s.writeItemAsResponse(filecache, ctx.W)

	}
}

func (h *StaticResourceHandler) readFileFromData(fileName string) (*fileCacheItem, bool) {
	if h.cache != nil {
		if item, ok := h.cache.Get(fileName); ok {
			return item.(*fileCacheItem), true
		}
	}
	return nil, false
}

func getFileExt(fn string) string {
	ext := strings.Split(fn, ".")
	if len(ext) == 1 {
		return ""
	}
	return ext[1]
}

// 直接操作了 http.ResponseWriter,绕过了ctx.Respdata flush
func (h *StaticResourceHandler) writeItemAsResponse(item *fileCacheItem, writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusOK)
	writer.Header().Set("Content-Type", item.contentType)
	writer.Header().Set("Content-Length", fmt.Sprintf("%d", item.fileSize))
	_, _ = writer.Write(item.data)

}
