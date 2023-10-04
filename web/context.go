package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

// ——————————————————————————————————————————————————————————————
// 封装的结构，避免给框架使用者暴露太多
// ——————————————————————————————————————————————————————————————
// 基于code 复用，封装一部分代码用于w,r的读写操作
type Context struct {
	//request info
	R                *http.Request
	PathParams       map[string]string // 路由匹配参数（正则或参数路径）
	cacheQueryValues url.Values        // query value的缓存
	MatchRoute       string            //匹配到的完整路由节点

	//response info
	// Resp 原生的 ResponseWriter没有对外开放的接口方法
	//如果直接使用 W ，相当于绕开了 RespStatusCode 和 RespData，这时候响应数据直接被发送到前端，其它中间件将无法修改响应
	// 增加了RespStatusCode 和 RespData后，其实可以考虑将W做成私有的
	W http.ResponseWriter
	// 缓存的响应部分,这部分数据会在最后刷新
	RespStatusCode int    //响应码（供middleware调用）
	RespData       []byte //保存的响应内容（json格式，供middleware调用）

	// 万一将来有需求，可以考虑支持这个，但是需要复杂一点的机制
	// Body []byte 用户返回的响应
	// Err error 用户执行的 Error

	// 页面渲染的引擎
	tplEngine TemplateEngine

	// 用户可以自由决定在这里存储什么，
	// 主要用于解决在不同 Middleware 之间数据传递的问题
	// 但是要注意
	// 1. UserValues 在初始状态的时候总是 nil，你需要自己手动初始化
	// 2. 不要在New预定义,这样在判定是否nil的时候会失败，即map[string]any{} != nil)
	UserValues map[string]any
}

// 新建
func NewContext(w http.ResponseWriter, r *http.Request, tplEngine TemplateEngine) *Context {
	return &Context{
		W:         w,
		R:         r,
		tplEngine: tplEngine, //由客户初始化模板引擎
	}
}

//处理输入要解决的问题：
// • 反序列化输入：将 Body 字节流转换成一个具体的类型
// • 处理表单输入：可以看做是一个和 JSON 或者 XML 差不多的一种特殊序列化方式
// • 处理查询参数：指从 URL 中的查询参数中读取值，并且转化为对应的类型
// • 处理路径参数：读取路径参数的值，并且转化为具体的类型
// • 重复读取 body：http.Request 的 Body 默认是只能读取一次，不能重复读取的
// • 读取 Header：从 Header 里面读取出来特定的值，并且转化为对应的类型
// X 模糊读取：按照一定的顺序，尝试从 Body、Header、路径参数或者 Cookie 里面读取值，并且转化为特定类型 (不支持)

// 反序列化输入：将 Body 字节流转换成一个具体的类型(解析body中的数据为json格式,输入到val中)
// 重复读取 body：http.Request 的 Body 默认是只能读取一次，不能重复读取的
func (c *Context) BindJSON(val any) error {
	if c.R.Body == nil {
		return errors.New("web:body为nil")
	}
	decoder := json.NewDecoder(c.R.Body)
	decoder.DisallowUnknownFields() // val中含有无法匹配上的字段时会报错
	return decoder.Decode(val)
}

// 处理表单输入：表单+URL数据按key取值,也是只取 From[key][0]第一个，一般存在表单的情况下取得是表单赋的值
func (c *Context) FormValue(key string) StringValue {
	err := c.R.ParseForm() // 解析传输过来的数据
	if err != nil {
		return StringValue{err: err}
	}
	return StringValue{val: c.R.FormValue(key), err: err} //原生FromValue有ParseForm,这里重复一遍是为了确认err
	// 不担心FromValue重复 ParseForm,内部有缓存 From, 确认From 有数据就会不做ParseForm
}

// 处理查询参数：query URL 查询数据（url ?后面的部分）按key取值，只取第一个 Values[k][0]
func (c *Context) QueryValue(key string) StringValue {
	if c.cacheQueryValues == nil {
		c.cacheQueryValues = c.R.URL.Query()
	}
	val, ok := c.cacheQueryValues[key]
	if !ok {
		return StringValue{err: errors.New("web: 找不到这个 key")}
	}
	return StringValue{val: val[0]}
}

// 处理路径参数：查询路由匹配路径的参数（不是url,是url与路由的参数匹配值）
func (c *Context) PathValue(key string) StringValue {
	val, ok := c.PathParams[key]
	if !ok {
		return StringValue{err: errors.New("web: 找不到这个 key")}
	}
	return StringValue{val: val}
}

// 读取 Header：从 Header 里面读取出来特定的值，并且转化为对应的类型（格式转换可以用StringValue）
func (c *Context) HeaderJson(key string) StringValue {
	val := c.R.Header.Get(key)
	if val == "" {
		return StringValue{err: errors.New("web: 找不到这个 key")}
	}
	return StringValue{val: val}
}

// 格式转换：方便用户自己选择对应格式，示例用法 c.FromValue().ToInt64()=> ToInt64()可变其他格式
type StringValue struct {
	val string
	err error
}

// 解包数据结构体
func (s StringValue) ToString() (string, error) {
	return s.val, s.err
}

// 解包数据结构体 + 转换Int64
func (s StringValue) ToInt64() (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return strconv.ParseInt(s.val, 10, 64)
}

// 避免如下一种格式写一个复杂函数对应
// func (c *Context) QueryValueAsInt64(key string) (int64, error) {
// 	val, err := c.QueryValue(key)
// 	if err != nil {
// 		return 0, err
// 	}
// 	return strconv.ParseInt(val, 10, 64)
// }

//*********************************************************************************************
// 处理输出要解决的问题：
// • 序列化输出：按照某种特定的格式输出数据，例如 JSON 或者 XML
// • 渲染页面：要考虑模板定位、命名和渲染的问题
// x 处理状态码：允许用户返回特定状态码的响应，例如 HTTP 404
// • 错误页面：特定 HTTP Status 或者 error 的时候，能够重定向到一个错误页面，例如404 被重定向到首页
// • 设置 Cookie ：设置 Cookie 的值
// • 设置 Header：往 Header放东西

// 序列化输出：按照某种特定的格式输出数据，例如 JSON 或者 XML
func (c *Context) RespJson(code int, data any) error {
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.RespStatusCode = code
	c.RespData = val
	return err
}

// 设置 Cookie ：设置 Cookie 的值
func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.W, cookie)
	//c.W.Header().Set("Set-Cookie",cookie.String())
	// c.R.Cookie(key),c.R.Cookies(key) 查询cookie
	// c.R.AddCookie(cookie) 增加cookie
}

// • 设置 Header：往 Header放东西
func (c *Context) setHeader(key, val string) {
	c.W.Header().Add(key, val)
}

// 进一步封装以便提供更便捷的方法
func (c *Context) OKRequestJson(data interface{}) error {
	return c.RespJson(http.StatusOK, data) //status code:200
}
func (c *Context) BadRequestJson(data interface{}) error {
	return c.RespJson(http.StatusBadRequest, data) //status code:400
}

// 不导出的内部结构，用于json 反序列化存放 Sign 相关信息
type signUpReq struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	ConfirmedPassword string `json:"confirmed_password"`
}

func NewsignUpReq() *signUpReq {
	return &signUpReq{}
}

// 创建统一的回复结构体格式
type commmonResponse struct {
	BizCode int         `json:"bizcode"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data"`
}

func NewcommonResponse(b int, m string, d interface{}) *commmonResponse {
	return &commmonResponse{
		BizCode: b,
		Msg:     m,
		Data:    d,
	}
}

func (c *Context) Render(tplName string, data any) error {
	var err error
	c.RespData, err = c.tplEngine.Render(tplName, data)
	c.RespStatusCode = 200
	if err != nil {
		c.RespStatusCode = 500
	}
	return err
}
