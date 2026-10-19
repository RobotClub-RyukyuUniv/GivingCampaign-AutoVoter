package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

// 受付状況（サイトの eventStatus）。
const (
	EventPre    = "pre"
	EventOpened = "opened"
	EventClosed = "closed"
)

// SiteStatus は大学サイトに埋め込まれたキャンペーンの受付状況と期間。
type SiteStatus struct {
	Event string // pre / opened / closed
	Start time.Time
	End   time.Time
}

var nextDataRe = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__"[^>]*>(.*?)</script>`)

// SiteStatus は大学サイトのページから受付状況を読む。API に受付状況のエンドポイントが無いため、
// サイトが描画に使っている __NEXT_DATA__ を見る（小さい /form ページを使う）。
func (c *Client) SiteStatus(ctx context.Context, origin string) (*SiteStatus, error) {
	if c.SiteURLOverride != "" {
		origin = c.SiteURLOverride
	}
	req, err := http.NewRequestWithContext(ctx, "GET", origin+"/form", nil)
	if err != nil {
		return nil, err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s/form: HTTP %d", origin, res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	m := nextDataRe.FindSubmatch(b)
	if m == nil {
		return nil, errors.New("ページに __NEXT_DATA__ がありません")
	}
	var d struct {
		Props struct {
			PageProps struct {
				EventStatus string `json:"eventStatus"`
			} `json:"pageProps"`
		} `json:"props"`
		RuntimeConfig struct {
			Start string `json:"GC_SCHEDULE_START"`
			End   string `json:"GC_SCHEDULE_END"`
		} `json:"runtimeConfig"`
	}
	if err := json.Unmarshal(m[1], &d); err != nil {
		return nil, err
	}
	st := &SiteStatus{Event: d.Props.PageProps.EventStatus}
	st.Start, _ = time.Parse(time.RFC3339, d.RuntimeConfig.Start)
	st.End, _ = time.Parse(time.RFC3339, d.RuntimeConfig.End)
	if st.Event == "" {
		return nil, errors.New("受付状況が見つかりません")
	}
	return st, nil
}
