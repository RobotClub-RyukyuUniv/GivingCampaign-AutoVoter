package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/api"
	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/prompt"
)

func fakeAuthAPI(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/gc2026/auth/login":
			front := r.URL.Query().Get("front_url")
			json.NewEncoder(w).Encode(map[string]string{"login_url": "https://login.example/authorize?front_url=" + url.QueryEscape(front)})
		case "/gc2026/auth/exchange_token":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["token"] != "tok123" {
				w.WriteHeader(400)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"session_id": "session-from-token"})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLoginViaLocalCallback(t *testing.T) {
	srv := fakeAuthAPI(t)
	c := api.New()
	c.BaseURL = srv.URL
	OpenBrowser = func(loginURL string) error {
		// ログイン後に Cross Campus 側から front_url/auth/callback へ戻される動きを再現
		u, _ := url.Parse(loginURL)
		go http.Get(u.Query().Get("front_url") + "/auth/callback?token=tok123")
		return nil
	}
	t.Cleanup(func() { OpenBrowser = openBrowser })

	p := prompt.New(strings.NewReader(""), &strings.Builder{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := Login(ctx, c, p)
	if err != nil || id != "session-from-token" {
		t.Fatalf("Login = %q, %v", id, err)
	}
}

func TestLoginViaPaste(t *testing.T) {
	srv := fakeAuthAPI(t)
	c := api.New()
	c.BaseURL = srv.URL
	OpenBrowser = func(string) error { return nil }
	t.Cleanup(func() { OpenBrowser = openBrowser })

	in := "https://usp.2026.giving-campaign.jp/auth/callback?token=tok123\n"
	p := prompt.New(strings.NewReader(in), &strings.Builder{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := Login(ctx, c, p)
	if err != nil || id != "session-from-token" {
		t.Fatalf("Login = %q, %v", id, err)
	}
}
