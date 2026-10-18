package vote

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/api"
	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/prompt"
	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/target"
)

// Plan は1件の投票対象を解決した結果。
type Plan struct {
	Target target.Target
	Org    *api.Organization
	Group  *api.Group
	Err    error // 解決できなかった・投票できない理由
}

func (pl *Plan) Title() string {
	if pl.Org != nil && pl.Group != nil {
		return pl.Org.Name + " / " + pl.Group.Name
	}
	if pl.Target.Label != "" {
		return pl.Target.Label
	}
	return pl.Target.Host + " " + pl.Target.GroupID
}

// Resolve は大学（組織）と団体を公開APIで引き当てる。
func Resolve(ctx context.Context, c *api.Client, targets []target.Target) []*Plan {
	orgs := map[string]*api.Organization{}
	groups := map[string][]api.Group{}
	var plans []*Plan
	for _, t := range targets {
		pl := &Plan{Target: t}
		plans = append(plans, pl)
		cc := c.WithOrigin(t.Origin())
		org, ok := orgs[t.Slug]
		if !ok {
			o, err := cc.OrganizationBySlug(ctx, t.Slug)
			if err != nil {
				pl.Err = fmt.Errorf("大学情報を取得できません: %w", err)
				continue
			}
			org, orgs[t.Slug] = o, o
		}
		pl.Org = org
		gs, ok := groups[org.ID]
		if !ok {
			var err error
			if gs, err = cc.Groups(ctx, org.ID); err != nil {
				pl.Err = fmt.Errorf("団体一覧を取得できません: %w", err)
				continue
			}
			groups[org.ID] = gs
		}
		for i := range gs {
			if gs[i].ID == t.GroupID {
				pl.Group = &gs[i]
				break
			}
		}
		if pl.Group == nil {
			pl.Err = fmt.Errorf("%s に団体ID %s が見つかりません", org.Name, t.GroupID)
		}
	}
	// 応援は1人1大学につき1回まで。同じ大学の団体が複数あれば2件目以降は投票できない。
	seen := map[string]*Plan{}
	for _, pl := range plans {
		if pl.Err != nil || pl.Org == nil {
			continue
		}
		if first, ok := seen[pl.Org.ID]; ok {
			pl.Err = fmt.Errorf("応援は1つの大学につき1人1回までです（%s に投票するため対象外）", first.Group.Name)
			continue
		}
		seen[pl.Org.ID] = pl
	}
	return plans
}

// Result は1件の投票結果。
type Result struct {
	Plan    *Plan
	Voted   bool
	Message bool
	Note    string
	Err     error
}

// Submit は参加登録 → 投票 → 応援メッセージ送信を行う。
func Submit(ctx context.Context, c *api.Client, pl *Plan, prof *api.CrossCampusProfile, a *Answers, authToken string) Result {
	r := Result{Plan: pl}
	cc := c.WithOrigin(pl.Target.Origin())
	orgID := pl.Org.ID

	req, err := BuildRequest(prof, a, pl.Org, pl.Group.ID, authToken)
	if err != nil {
		r.Err = err
		return r
	}
	pid, err := cc.CreateOtherUniversityStudent(ctx, orgID, req)
	if api.IsStatus(err, http.StatusConflict) {
		// このアカウント（または電話番号）で参加登録済み。既存の参加者で続行する。
		var meErr error
		if pid, meErr = cc.MyParticipant(ctx, orgID, authToken); meErr != nil {
			r.Err = fmt.Errorf("この大学には参加登録済みですが、登録情報を取得できませんでした: %w", errors.Join(err, meErr))
			return r
		}
		r.Note = "参加登録済みのため既存の登録を使用"
	} else if err != nil {
		r.Err = fmt.Errorf("参加登録に失敗しました: %w", err)
		return r
	}

	voteID, err := cc.CreateVote(ctx, orgID, pid, pl.Group.ID, authToken)
	if err != nil {
		if api.IsStatus(err, http.StatusConflict) || api.IsStatus(err, http.StatusForbidden) {
			r.Err = fmt.Errorf("投票できませんでした（この大学では投票済みの可能性があります）: %w", err)
		} else {
			r.Err = fmt.Errorf("投票に失敗しました: %w", err)
		}
		return r
	}
	r.Voted = true

	if strings.TrimSpace(a.Message) == "" {
		return r
	}
	if err := cc.PostMessage(ctx, orgID, pid, voteID, a.Message, authToken); err != nil {
		r.Err = fmt.Errorf("投票は完了しましたが、応援メッセージの送信に失敗しました: %w", err)
		return r
	}
	r.Message = true
	return r
}

// SMSAuth は Cross Campus で電話番号が未認証のときに SMS 認証で auth_token を得る。
func SMSAuth(ctx context.Context, c *api.Client, p *prompt.Prompter, phone string) (string, error) {
	e164, err := phoneE164(phone)
	if err != nil {
		return "", err
	}
	for {
		vid, err := c.SendSMS(ctx, e164)
		if err != nil {
			return "", fmt.Errorf("SMSの送信に失敗しました: %w", err)
		}
		p.Printf("%s に8桁の認証コードをSMSで送りました。\n", e164)
		for {
			code, err := p.Line("認証コード（再送するには r）: ")
			if err != nil {
				return "", err
			}
			if strings.EqualFold(code, "r") {
				break
			}
			code = strings.ToUpper(strings.ReplaceAll(code, " ", ""))
			if len(code) != 8 {
				p.Printf("8桁で入力してください。\n")
				continue
			}
			tok, err := c.VerifySMS(ctx, vid, code)
			switch {
			case err == nil && tok != "":
				return tok, nil
			case api.IsStatus(err, http.StatusGone):
				p.Printf("認証コードの有効期限が切れました。再送します。\n")
			case err != nil:
				p.Printf("認証に失敗しました（%v）。もう一度入力してください。\n", err)
				continue
			default:
				return "", errors.New("認証トークンが返されませんでした")
			}
			break
		}
	}
}
