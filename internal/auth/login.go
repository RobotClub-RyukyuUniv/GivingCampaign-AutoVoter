// Package auth は Cross Campus ログインとセッションの保存を扱う。
package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/api"
	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/prompt"
)

var sessionIDRe = regexp.MustCompile(`^[A-Za-z0-9._~+/=-]{16,}$`)

// EnsureSession は保存済みセッションが有効ならそれを使い、無効ならログインを行う。
// 成功すると c.SessionID が設定される。
func EnsureSession(ctx context.Context, c *api.Client, p *prompt.Prompter, forceLogin bool) (*api.Me, error) {
	if !forceLogin {
		if s, err := LoadSession(); err == nil && s != nil {
			c.SessionID = s.ID
			me, err := c.Me(ctx)
			if err == nil && me.CrossCampusAuthenticated {
				return me, nil
			}
			c.SessionID = ""
		}
	}
	id, err := Login(ctx, c, p)
	if err != nil {
		return nil, err
	}
	c.SessionID = id
	me, err := c.Me(ctx)
	if err != nil {
		return nil, fmt.Errorf("ログイン後のセッション確認に失敗しました: %w", err)
	}
	if !me.CrossCampusAuthenticated {
		return nil, errors.New("Cross Campus の認証が完了していません")
	}
	if err := SaveSession(id); err != nil {
		p.Printf("（セッションの保存に失敗しました: %v）\n", err)
	}
	return me, nil
}

// Login はブラウザで Cross Campus にログインしてもらい、セッションIDを得る。
//
// 手元に http://127.0.0.1:<port> のサーバーを立て、それを front_url にしてログインURLを発行する。
// ログイン後ブラウザが /auth/callback?token=... に戻ってきたら token をセッションIDに交換する。
// 自動で戻ってこない場合に備えて、callback のURL・token・セッションIDの貼り付けも受け付ける。
func Login(ctx context.Context, c *api.Client, p *prompt.Prompter) (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer ln.Close()
	front := "http://" + ln.Addr().String()

	tokens := make(chan string, 1)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := r.URL.Query().Get("token")
		if r.URL.Path != "/auth/callback" || tok == "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!doctype html><meta charset="utf-8"><title>gcvote</title><p style="font-family:sans-serif">ログインできました。このタブを閉じてターミナルに戻ってください。</p>`)
		select {
		case tokens <- tok:
		default:
		}
	})}
	go srv.Serve(ln)
	defer srv.Close()

	loginURL, err := c.LoginURL(ctx, front)
	if err != nil {
		return "", fmt.Errorf("ログインURLの取得に失敗しました: %w", err)
	}

	p.Printf("\nブラウザで Cross Campus にログインしてください。\n  %s\n", loginURL)
	if err := OpenBrowser(loginURL); err != nil {
		p.Printf("（ブラウザを自動で開けませんでした。上のURLを開いてください）\n")
	}
	p.Printf("ログインが終わると自動で続行します。\n")
	p.Printf("自動で戻らない場合は、ブラウザのアドレスバーのURL（…/auth/callback?token=…）か\n")
	p.Printf("セッションIDをここに貼り付けてEnter（詳しくは README の「ログインがうまくいかないとき」）: ")

	timeout := time.NewTimer(10 * time.Minute)
	defer timeout.Stop()
	lines := p.Lines()
	for {
		select {
		case tok := <-tokens:
			p.Printf("\n")
			return exchange(ctx, c, tok)
		case line, ok := <-lines:
			if !ok {
				lines = nil // 標準入力が閉じてもブラウザからのコールバックは待ち続ける
				continue
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if tok := tokenFromInput(line); tok != "" {
				return exchange(ctx, c, tok)
			}
			if sessionIDRe.MatchString(line) {
				return line, nil
			}
			p.Printf("認識できませんでした。callback のURLかセッションIDを貼り付けてください: ")
		case <-timeout.C:
			return "", errors.New("ログインがタイムアウトしました")
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func exchange(ctx context.Context, c *api.Client, token string) (string, error) {
	id, err := c.ExchangeToken(ctx, token)
	if err != nil {
		return "", fmt.Errorf("トークンの交換に失敗しました: %w", err)
	}
	if id == "" {
		return "", errors.New("トークンの交換に失敗しました（session_id が空）")
	}
	return id, nil
}

// tokenFromInput は貼り付けられた callback URL から token を取り出す。
func tokenFromInput(s string) string {
	if !strings.Contains(s, "token=") {
		return ""
	}
	if u, err := url.Parse(s); err == nil {
		if t := u.Query().Get("token"); t != "" {
			return t
		}
	}
	if q, err := url.ParseQuery(strings.TrimPrefix(s[strings.Index(s, "token="):], "?")); err == nil {
		return q.Get("token")
	}
	return ""
}
