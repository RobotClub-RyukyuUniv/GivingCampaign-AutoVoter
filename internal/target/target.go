// Package target は告知文やURLから投票対象（大学サイトと団体）を取り出す。
package target

import (
	"errors"
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
	Pos     int    // 告知文中で何番目の URL か（表示順用）
}

// Origin は大学サイトのオリジン (https://<host>) を返す。
func (t Target) Origin() string { return "https://" + t.Host }

var (
	urlRe     = regexp.MustCompile(`https?://[^\s<>"'）)]+`)
	hostRe    = regexp.MustCompile(`^([a-z0-9-]+)\.(\d{4})\.giving-campaign\.jp$`)
	uuidRe    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	commentRe = regexp.MustCompile(`コメント欄に[^「]*「([^」]+)」`)
)

// Invalid は告知文中にあったが投票対象にできなかった URL。
type Invalid struct {
	Label string // 告知文で URL の直前にあった行
	URL   string
	Err   error
	Pos   int
}

// Year は対応しているキャンペーンの年度。
const Year = "2026"

// Parse は任意のテキスト（告知文の貼り付けなど）から投票対象を重複なく取り出す。
// 投票対象にできない URL は理由付きで invalid に入る。
func Parse(text string) (targets []Target, invalid []Invalid) {
	seen := map[string]bool{}
	prev := ""
	pos := 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		matches := urlRe.FindAllString(line, -1)
		if len(matches) == 0 {
			if line != "" {
				prev = line
			}
			continue
		}
		label := strings.TrimSpace(urlRe.ReplaceAllString(line, ""))
		if label == "" {
			label = prev
		}
		for _, raw := range matches {
			pos++
			t, err := ParseURL(raw)
			if err != nil {
				invalid = append(invalid, Invalid{Label: label, URL: raw, Err: err, Pos: pos})
				continue
			}
			key := t.Host + "/" + t.GroupID
			if seen[key] {
				continue
			}
			seen[key] = true
			t.Label = label
			t.Pos = pos
			targets = append(targets, t)
		}
		prev = ""
	}
	return targets, invalid
}

// ParseURL は団体ページ・投票フォームのURLを解析する。
// group / group-id / group_id のいずれかのクエリに団体IDが入っている必要がある。
func ParseURL(raw string) (Target, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Target{}, errors.New("URLの形式が正しくありません")
	}
	host := strings.ToLower(u.Hostname())
	m := hostRe.FindStringSubmatch(host)
	switch {
	case m == nil:
		return Target{}, errors.New("Giving Campaign の大学サイトのURLではありません")
	case m[2] != Year:
		return Target{}, fmt.Errorf("%s年のキャンペーンのURLです（対応しているのは%s年のみ）", m[2], Year)
	}
	q := u.Query()
	var gid, partial string
	for _, k := range []string{"group", "group-id", "group_id"} {
		v := strings.ToLower(q.Get(k))
		if uuidRe.MatchString(v) {
			gid = v
			break
		}
		if v != "" {
			partial = v
		}
	}
	switch {
	case gid != "":
		return Target{Host: host, Slug: m[1], GroupID: gid}, nil
	case partial != "":
		return Target{}, errors.New("団体IDが不完全です（URLが途中で切れていないか確認してください）")
	default:
		return Target{}, errors.New("団体を指定するURLではありません（団体ページの「シェア」などのURLを使ってください）")
	}
}

// SuggestMessage は告知文から「コメント欄に必ず「〜」と記入」の〜部分を取り出す。
func SuggestMessage(text string) string {
	if m := commentRe.FindStringSubmatch(text); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}
