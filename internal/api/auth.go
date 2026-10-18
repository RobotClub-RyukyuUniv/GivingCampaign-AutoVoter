package api

import (
	"context"
	"net/url"
)

// LoginURL は Cross Campus のログインページURLを取得する。
// ログイン後、ブラウザは <frontURL>/auth/callback?token=... に戻される。
func (c *Client) LoginURL(ctx context.Context, frontURL string) (string, error) {
	var res struct {
		LoginURL string `json:"login_url"`
	}
	err := c.do(ctx, "GET", "/gc2026/auth/login", url.Values{"front_url": {frontURL}}, nil, &res)
	return res.LoginURL, err
}

// ExchangeToken はコールバックで受け取った token をセッションIDに交換する。
func (c *Client) ExchangeToken(ctx context.Context, token string) (string, error) {
	var res struct {
		SessionID string `json:"session_id"`
	}
	err := c.do(ctx, "POST", "/gc2026/auth/exchange_token", nil, map[string]string{"token": token}, &res)
	return res.SessionID, err
}

type Me struct {
	CrossCampusAuthenticated bool   `json:"cross_campus_authenticated"`
	Email                    string `json:"email"`
}

// Me はセッションの状態を返す。未ログインなら HTTP 401。
func (c *Client) Me(ctx context.Context) (*Me, error) {
	var res Me
	if err := c.do(ctx, "GET", "/gc2026/auth/me", nil, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
