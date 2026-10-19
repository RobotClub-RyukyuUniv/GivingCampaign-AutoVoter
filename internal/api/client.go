// Package api は Giving Campaign 2026 のバックエンド (api.alumnote.jp/gc2026) を直接呼ぶクライアント。
// エンドポイントや形式はサイトのフロントエンド実装から読み取ったもので、公式に公開されたAPIではない。
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.alumnote.jp"

type Client struct {
	BaseURL   string
	HTTP      *http.Client
	SessionID string // Cross Campus ログインのセッション (X-CC-Session-ID)
	Origin    string // 呼び出し元の大学サイト (https://xxx.2026.giving-campaign.jp)
	Debug     func(format string, a ...any)
	// SiteURLOverride はテスト用。設定すると大学サイトの代わりにこのURLを読む。
	SiteURLOverride string
}

func New() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{BaseURL: DefaultBaseURL, HTTP: &http.Client{Timeout: 30 * time.Second, Jar: jar}}
}

// WithOrigin は Origin だけ差し替えたコピーを返す（大学ごとに Origin/Referer を合わせるため）。
func (c *Client) WithOrigin(origin string) *Client {
	cp := *c
	cp.Origin = origin
	return &cp
}

// Error はAPIが2xx以外を返したときのエラー。
type Error struct {
	Method, Path string
	Status       int
	Body         string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s %s: HTTP %d: %s", e.Method, e.Path, e.Status, e.Body)
}

// IsStatus は err が指定ステータスの API エラーかどうか。
func IsStatus(err error, status int) bool {
	ae, ok := err.(*Error)
	return ok && ae.Status == status
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, in, out any) error {
	u := strings.TrimRight(c.BaseURL, "/") + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var body io.Reader
	var reqBody []byte
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		reqBody = b
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json;charset=utf-8")
	}
	if c.Origin != "" {
		req.Header.Set("Origin", c.Origin)
		req.Header.Set("Referer", c.Origin+"/")
	}
	if c.SessionID != "" {
		req.Header.Set("X-CC-Session-ID", c.SessionID)
	}
	if c.Debug != nil {
		c.Debug("→ %s %s %s", method, u, reqBody)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	resBody, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return err
	}
	if c.Debug != nil {
		c.Debug("← %d %s", res.StatusCode, truncate(string(resBody), 2000))
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return &Error{Method: method, Path: path, Status: res.StatusCode, Body: truncate(string(resBody), 500)}
	}
	if out != nil && len(resBody) > 0 {
		if err := json.Unmarshal(resBody, out); err != nil {
			return fmt.Errorf("%s %s: レスポンスを解析できません: %w", method, path, err)
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
