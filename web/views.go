package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// // 自定义错误格式(暂弃)
// type Myerror struct {
// 	Err  error
// 	Eerr error
// }

// func (m *Myerror) Error() string {
// 	return m.Err.Error() + "\n" + m.Eerr.Error()
// }

// 基于code 复用，封装一部分代码用于w,r的读写操作
type Context struct {
	W http.ResponseWriter
	R *http.Request
}

// 读取
func (c *Context) ReadJson(data interface{}) error {
	body, err := io.ReadAll(c.R.Body)

	//测试body
	// fmt.Printf("c.R.Body: %v\n", c.R.Body)
	// fmt.Printf("body: %v\n", string(body))

	if err != nil {
		errjson := errors.Join(err, errors.New("json deserialized fail"))
		return errjson
	}
	return json.Unmarshal(body, data)
}

// 写入
func (c *Context) WriterJson(status int, data interface{}) error {
	c.W.WriteHeader(status)
	dataJSON, err := json.Marshal(data)

	if err != nil {
		errjson := errors.Join(err, errors.New("json serialized fail"))
		return errjson
		// return err
	}
	_, err = c.W.Write([]byte(dataJSON))
	return err
}

// 新建
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		W: w,
		R: r,
	}
}

// 不导出的内部结构，用于json 反序列化存放 Sign 相关信息
type signUpReq struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	ConfirmedPassword string `json:"confirmed_password"`
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

//——————————————————————————————————————————————————————————————
//函数主体
//——————————————————————————————————————————————————————————————

// 登录实验的具体函数 Sign
func Sign(c *Context) {

	req := new(signUpReq)
	// 从r（request）读取数据
	err := c.ReadJson(req)

	//测试用例~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// example := signUpReq{
	// 	Email:             "1",
	// 	Password:          "2",
	// 	ConfirmedPassword: "3",
	// }
	// exjson, _ := json.Marshal(&example)
	// fmt.Printf("exjson: %v\n", string(exjson))
	// fmt.Printf("req: %v\n", req)
	//~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	//返回客户的错误（response）
	if err != nil {
		fmt.Fprintf(c.W, "invalid request:%v", err)
		return
	}

	resp := NewcommonResponse(4, "输入的信息为", req)

	err = c.WriterJson(http.StatusOK, resp)
	//写入失败的情况下，无法返回给客户信息，应当输出日志
	if err != nil {
		fmt.Printf("写入响应失败：%v", err)
	}
}

// 进一步封装以便提供更便捷的方法
func (c *Context) OKRequestJson(data interface{}) error {
	return c.WriterJson(http.StatusOK, data) //status code:200
}
func (c *Context) BadRequestJson(data interface{}) error {
	return c.WriterJson(http.StatusBadRequest, data) //status code:400
}
