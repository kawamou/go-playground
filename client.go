package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
	"google.golang.org/api/googleapi"
)

const baseURL = "https://api.foo.com"

// Client はCloud Run上に構築したAPIにアクセスするクライアントです
// 基本的にはHTTP/2で通信することを想定しています
// https://deeeet.com/writing/2016/11/01/go-api-client/
type Client struct {
	client *http.Client
}

// NewClient はClientのコンストラクタです
// Transportを指定しないとDefaultClientになり無限Timeoutなため非推奨です
// Transportを指定しForceAttemptHTTP2をtrueにすることでHTTP/2を試みます
// https://christina04.hatenablog.com/entry/go-http2-client
func NewClient() (*Client, error) {
	return &Client{
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   60 * time.Minute, // Cloud Runの最長タイムアウト時間
					KeepAlive: 10 * time.Second, // TODO: 値をどうするか
					DualStack: true,
				}).DialContext,
				ForceAttemptHTTP2: true,
			},
		},
	}, nil
}

// decodeResponseBody ...
// https://cloud.google.com/apis/design/errors#error_mapping
func decodeResponseBody(b []byte) (*googleapi.Error, error) {
	var gAPIErr *googleapi.Error
	if err := json.Unmarshal(b, &gAPIErr); err != nil {
		return nil, err
	}
	return gAPIErr, nil
}

func makeRequestBody(param interface{}) (io.Reader, error) {
	values, err := json.Marshal(param)
	if err != nil {
		return nil, nil
	}
	return bytes.NewBuffer(values), nil
}

func buildRequest(method, path string, body io.Reader) (req *http.Request, err error) {
	// NOTE: GETでもBodyが載る
	req, err = http.NewRequest(method, baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	return req, nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// isSpecificError ...
func isSpecificError(err error) bool {
	switch err.(type) {
	case http2.GoAwayError:
		// server sent GOAWAY and closed the connection...
		// 系のエラーが入った場合はこちらに入ります
		return true
	case http2.StreamError:
		// stream error: stream ID...
		// 系のエラーが生じた場合はこちらに入ります
		return true
	}
	return false
}

type Foo struct {
	ID string `json:"id"`
}

func (c *Client) CreateFoo(foo Foo) error {
	body, _ := makeRequestBody(foo)
	req, _ := buildRequest("POST", "/foos", body)
	resp, err := c.Do(req)
	if err != nil {
		if isSpecificError(err) {
			// 何かしらの処理
		}
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	decoded, _ := decodeResponseBody(b)
	// Run上のアプリがレスポンス返すときとロードバランサがレスポンス返すときがある
	// そのためステータスコードだけで判断するのが難しいのでdecodeして
	// 中身をもとに判断する必要がありそう
	// ここに何かしらの処理
	return nil
}

// func (c *Client) GetFoo(id string) {

// }

// func (c *Client) UpdateFoo(id string) {

// }

// func (c *Client) DeleteFoo(id string) {

// }
