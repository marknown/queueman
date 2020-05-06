// Package ohttp is own http client
package ohttp

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Request 请求
type Request http.Request

// Response 响应
type Response http.Response

// RequestSettings 请求设置结构体
type RequestSettings struct {
	Timeout           time.Duration // 请求超时时间
	IsRandomUserAgent bool          // 是否随机UserAgent
	UserAgent         string        // UserAgent 如果 IsRandomUserAgent 设置了，则随机
	Referer           string        // 设置 Referer
	IsFollowLocation  bool          // 是否跟随跳转
	IsAajx            bool          // 是否是 ajax 请求
	ContentType       string        // 内容类型
	Cookies           string        // Cookies 字符串
	Headers           [][2]string   // 额外的请求头
}

// defaultSetting 默认设置
var defaultSetting = RequestSettings{
	Timeout:           0 * time.Second,
	IsRandomUserAgent: false,
	UserAgent:         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36",
	Referer:           "",
	IsFollowLocation:  true,
	IsAajx:            false,
	ContentType:       "application/x-www-form-urlencoded",
	Cookies:           "",
	Headers:           [][2]string{},
}

// InitSetttings 重新初始化请求设置
func InitSetttings() *RequestSettings {
	s := defaultSetting

	return &s
}

// Get 执行 Get 请求
func (settings *RequestSettings) Get(url string) (string, *Response, error) {
	return settings.request("GET", url, nil)
}

// Post 执行 Post 请求
func (settings *RequestSettings) Post(url string, params interface{}) (string, *Response, error) {
	return settings.request("POST", url, params)
}

func (settings *RequestSettings) request(method string, url string, params interface{}) (string, *Response, error) {
	req, err := settings.NewRequest(method, url, params)

	if err != nil {
		return "", nil, err
	}

	resp, err := settings.Do(req)

	if err != nil {
		return "", nil, err
	}

	content, err := resp.ContentString()

	if err != nil {
		return "", nil, err
	}

	return content, resp, err
}

// NewRequest 创建一个 Request 请求
func (settings *RequestSettings) NewRequest(method string, url string, params interface{}) (*Request, error) {
	var req *http.Request
	var err error
	if _, ok := params.(string); ok {
		req, err = http.NewRequest(method, url, strings.NewReader(params.(string)))
	}
	if _, ok := params.(map[string]string); ok {
		req, err = http.NewRequest(method, url, BuildQueryReader(params.(map[string]string)))
	}
	if nil == params {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	if "" != settings.UserAgent {
		req.Header.Set("User-Agent", settings.UserAgent)
	}

	if "" != settings.Referer {
		req.Header.Set("Referer", settings.Referer)
	}

	if settings.IsAajx {
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
	}

	if "" != settings.ContentType {
		req.Header.Set("Content-Type", settings.ContentType) // 如果是 POST 一定要有这一行，不然接收不到数据
	}

	for _, v := range settings.Headers {
		req.Header.Set(v[0], v[1])
	}

	if "" != settings.Cookies {
		req.Header.Set("Cookie", settings.Cookies)
	}

	r := Request(*req)
	return &r, err
}

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, 10*time.Second)
}

// Do 执行网络请求
func (settings *RequestSettings) Do(req *Request) (*Response, error) {
	r := http.Request(*req)

	transport := http.Transport{
		Dial:              dialTimeout,
		DisableKeepAlives: true,
	}

	// 超时设置
	var c = &http.Client{
		Transport: &transport,
		Timeout:   settings.Timeout * time.Second,
	}

	resp, err := c.Do(&r)

	if err != nil {
		return nil, err
	}

	response := Response(*resp)
	return &response, err
}

// SetCookie 设置 cookie
func (req *Request) SetCookie(cookies string) {
	req.Header.Set("Cookie", cookies)
}

// ContentString 获取 response 的内容
func (resp *Response) ContentString() (string, error) {
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return string(body), nil
}

// CookieString 获取 response 的内容
func (resp *Response) CookieString() string {
	r := http.Response(*resp)
	Cookies := r.Cookies()
	cookieArray := []string{}

	for _, cookie := range Cookies {
		cookieArray = append(cookieArray, cookie.Name+"="+cookie.Value)
	}

	return strings.Join(cookieArray, "; ")
}

// HeaderString 获取 Response 的 header
func (resp *Response) HeaderString() string {
	Headers := resp.Header

	HeaderString := ""

	for k, v := range Headers {
		for _, vv := range v {
			HeaderString += k + ": " + vv + "\n"
		}
	}

	return "Response Header:\n\n" + HeaderString + "\n"
}

// RequestCookieString 获取 request 的 cookie
func (resp *Response) RequestCookieString() string {
	Cookies := resp.Request.Cookies()
	cookieArray := []string{}

	for _, cookie := range Cookies {
		cookieArray = append(cookieArray, cookie.Name+"="+cookie.Value)
	}

	return strings.Join(cookieArray, "; ")
}

// RequestHeaderString 获取 request 的 header
func (resp *Response) RequestHeaderString() string {
	Headers := resp.Request.Header

	HeaderString := ""

	for k, v := range Headers {
		for _, vv := range v {
			HeaderString += k + ": " + vv + "\n"
		}
	}

	return "Request Header:\n\n" + HeaderString + "\n"
}

// 以下为公共函数

// BuildQuery 构造 GET 或者 POST 请求参数
func BuildQuery(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values[k] = []string{v}
	}
	result := values.Encode()

	// todo 以下代码针对淘宝秒杀做的限制
	result = strings.Replace(result, "%27", `'`, -1) // ' 转成 \' 不能转成 %27
	result = strings.Replace(result, "+", `%20`, -1) // + 转成 %20
	result = strings.Replace(result, "%28", `(`, -1) // ( 不转成 %28
	result = strings.Replace(result, "%29", `)`, -1) // ) 不转成 %29
	// tododel
	result = strings.Replace(result, "info%5C%5C%5C%22%3A%5B%5D", `info%5C%5C%5C%22%3A%7B%7D`, -1) // ) 不转成 %29

	return result
}

// BuildQueryReader 构造 GET 或者 POST 请求参数 io.Reader 形式
func BuildQueryReader(params map[string]string) io.Reader {
	return strings.NewReader(BuildQuery(params))
}

// URLValuesToStringMap 把 url.Values 转成 map[string]string
func URLValuesToStringMap(values url.Values) map[string]string {
	m := make(map[string]string)
	for k, v := range values {
		if len(v) > 0 {
			m[k] = v[0]
		} else {
			m[k] = ""
		}
	}

	return m
}

// IsAjaxRequest 判断是否是 ajax 请求
func IsAjaxRequest(r *http.Request) bool {
	return "XMLHttpRequest" == r.Header.Get("X-Requested-With")
}

// IsPostRequest 判断是否是 ajax 请求
func IsPostRequest(r *http.Request) bool {
	return "POST" == r.Method
}

// IsGetRequest 判断是否是 ajax 请求
func IsGetRequest(r *http.Request) bool {
	return "GET" == r.Method
}

// MapCookies 把字符串 cookie 转成 map 键值对
func MapCookies(cookies string) map[string]string {
	cookieArray := strings.Split(cookies, ";")
	result := map[string]string{}

	for _, cookie := range cookieArray {
		kv := strings.Split(strings.TrimSpace(cookie), "=")
		if len(kv) >= 2 && "" != kv[0] && "" != kv[1] {
			result[kv[0]] = kv[1]
		}
	}

	return result
}

// MapCookiesToString 把 map 键值对字符串转成 字符串 cookie
func MapCookiesToString(m map[string]string) string {
	cookieArray := []string{}

	for k, v := range m {
		cookieArray = append(cookieArray, k+"="+v)
	}

	return strings.Join(cookieArray, "; ")
}

// AppendCookies 在老的 cookie 字符串中添加新的 cookie 字符串，如果名称相同，则替换
func AppendCookies(oldCookie string, appendCookie string) string {
	oldCookieMap := MapCookies(oldCookie)
	appendCookieMap := MapCookies(appendCookie)

	for k, v := range appendCookieMap {
		oldCookieMap[k] = v
	}

	return MapCookiesToString(oldCookieMap)
}
