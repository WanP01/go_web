package view

import (
	"fmt"
	"go_web/web"
	"net/http"
)

//——————————————————————————————————————————————————————————————
//函数主体 down
//——————————————————————————————————————————————————————————————

// 登录实验的具体函数 Sign
func Sign(c *web.Context) {

	req := web.NewsignUpReq()
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

	resp := web.NewcommonResponse(4, "输入的信息为", req)

	err = c.WriterJson(http.StatusOK, resp)
	//写入失败的情况下，无法返回给客户信息，应当输出日志
	if err != nil {
		fmt.Printf("写入响应失败：%v", err)
	}
}

//——————————————————————————————————————————————————————————————
//函数主体 up
//——————————————————————————————————————————————————————————————
