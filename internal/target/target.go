// Package target は告知文やURLから投票対象（大学サイトと団体）を取り出す。
package target

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Target は1件の投票対象。
type Target struct {
	Host    string // 例: usp.2026.giving-campaign.jp
	Slug    string // 例: usp
	GroupID string // 団体ID (UUID)
	Label   string // 告知文中で URL の直前にあった行（表示用、無ければ空）
}

// Origin は大学サイトのオリジン (https://<host>) を返す。
func (t Target) Origin() string { return "https://" + t.Host }

var (
	urlRe     = regexp.MustCompile(`https?://[^\s<>"'）)]+`)
	hostRe    = regexp.MustCompile(`^([a-z0-9-]+)\.(\d{4})\.giving-campaign\.jp$`)
	uuidRe    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	commentRe = regexp.MustCompile(`コメント欄に[^「]*「([^」]+)」`)
)

// Parse は任意のテキスト（告知文の貼り付けなど）から投票対象を重複なく取り出す。
func Parse(text string) ([]Target, []error) {
	var (
		targets []Target
		errs    []error
		seen    = map[string]bool{}
		prev    string
	)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		matches := urlRe.FindAllString(line, -1)
		if len(matches) == 0 {
			if line != "" {
				prev = line
			}
			continue
		}
		for _, raw := range matches {
			t, err := ParseURL(raw)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			key := t.Host + "/" + t.GroupID
			if seen[key] {
				continue
			}
			seen[key] = true
			if label := strings.TrimSpace(urlRe.ReplaceAllString(line, "")); label != "" {
				t.Label = label
			} else {
				t.Label = prev
			}
			targets = append(targets, t)
		}
		prev = ""
	}
	return targets, errs
}

// ParseURL は団体ページ・投票フォームのURLを解析する。
// group / group-id / group_id のいずれかのクエリに団体IDが入っている必要がある。
func ParseURL(raw string) (Target, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Target{}, fmt.Errorf("URLを解析できません: %s", raw)
	}
	host := strings.ToLower(u.Hostname())
	m := hostRe.FindStringSubmatch(host)
	if m == nil {
		return Target{}, fmt.Errorf("Giving CampaignのURLではありません: %s", raw)
	}
	q := u.Query()
	var gid string
	for _, k := range []string{"group", "group-id", "group_id"} {
		if v := strings.ToLower(q.Get(k)); uuidRe.MatchString(v) {
			gid = v
			break
		}
	}
	if gid == "" {
		return Target{}, fmt.Errorf("URLに団体ID(group=...)が含まれていません: %s", raw)
	}
	return Target{Host: host, Slug: m[1], GroupID: gid}, nil
}

// SuggestMessage は告知文から「コメント欄に必ず「〜」と記入」の〜部分を取り出す。
func SuggestMessage(text string) string {
	if m := commentRe.FindStringSubmatch(text); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}
