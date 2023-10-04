package cookie

import (
	"net/http"
)

type PropagatorOption func(propagator *Propagator)

// 用于修改cookie option，比如max_age
func WithCookieOption(opt func(c *http.Cookie)) PropagatorOption {
	return func(propagator *Propagator) {
		propagator.cookieOpt = opt
	}
}

// 用户赋值
type Propagator struct {
	cookieName string               //cookies的名字
	cookieOpt  func(c *http.Cookie) //更改cookie的方法
}

func NewPropagator(cookieName string, opts ...PropagatorOption) *Propagator {
	res := &Propagator{
		cookieName: cookieName,
		cookieOpt:  func(c *http.Cookie) {},
	}

	for _, opt := range opts {
		opt(res)
	}
	return res
}

func (p *Propagator) Inject(id string, writer http.ResponseWriter) error {
	cookie := &http.Cookie{
		Name:  p.cookieName,
		Value: id, // session_id
	}
	p.cookieOpt(cookie)
	http.SetCookie(writer, cookie)
	return nil
}

func (p *Propagator) Extract(req *http.Request) (string, error) {
	cookie, err := req.Cookie(p.cookieName)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func (p *Propagator) Remove(writer http.ResponseWriter) error {
	cookie := &http.Cookie{
		Name:   p.cookieName,
		MaxAge: -1,
	}
	p.cookieOpt(cookie)
	http.SetCookie(writer, cookie)
	return nil
}
