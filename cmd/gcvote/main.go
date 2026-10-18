// gcvote は Giving Campaign 2026 の応援（投票）を API 直接で行う CLI。
//
//	gcvote                       告知文やURLを貼り付けて投票
//	gcvote <URL|告知文ファイル>...  引数で指定して投票
//	gcvote login | logout        Cross Campus へのログイン / ログアウト
//	gcvote config                保存済みの回答を設定し直す
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/api"
	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/auth"
	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/config"
	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/prompt"
	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/target"
	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/vote"
)

type options struct {
	configPath string
	message    string
	yes        bool
	dryRun     bool
	relogin    bool
	verbose    bool
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "エラー:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	defPath, err := config.DefaultPath()
	if err != nil {
		return err
	}
	var o options
	fs := flag.NewFlagSet("gcvote", flag.ContinueOnError)
	fs.StringVar(&o.configPath, "config", defPath, "設定ファイルのパス")
	fs.StringVar(&o.message, "m", "", "応援メッセージ（告知文の指定より優先）")
	fs.BoolVar(&o.yes, "y", false, "確認を省略して送信する（規約・プライバシーポリシーに同意したものとみなす）")
	fs.BoolVar(&o.dryRun, "dry-run", false, "送信内容を表示するだけで送信しない")
	fs.BoolVar(&o.relogin, "relogin", false, "保存済みセッションを使わずにログインし直す")
	fs.BoolVar(&o.verbose, "v", false, "APIの通信内容を表示する（個人情報が表示されるので注意）")
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), `使い方:
  gcvote [オプション]                 告知文やURLを貼り付けて投票
  gcvote [オプション] <URL|ファイル>... 引数で指定して投票（- で標準入力）
  gcvote login                        Cross Campus にログイン
  gcvote logout                       保存済みのセッションを削除
  gcvote config                       保存済みの回答を設定し直す

オプション:
`)
		fs.PrintDefaults()
	}

	sub := ""
	if len(args) > 0 {
		switch args[0] {
		case "login", "logout", "config", "vote":
			sub, args = args[0], args[1:]
		}
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	p := prompt.Default()
	c := api.New()
	if o.verbose {
		c.Debug = func(format string, a ...any) { fmt.Fprintf(os.Stderr, format+"\n", a...) }
	}

	switch sub {
	case "logout":
		if err := auth.DeleteSession(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "ログアウトしました（保存済みセッションを削除）。")
		return nil
	case "login":
		me, err := auth.EnsureSession(ctx, c, p, true)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "ログインしました: %s\n", me.Email)
		return nil
	case "config":
		return runConfig(ctx, c, p, o)
	}
	return runVote(ctx, c, p, o, fs.Args())
}

// readInput は引数（URL・ファイル・-）か対話入力から告知文を得る。
func readInput(p *prompt.Prompter, args []string) (string, error) {
	if len(args) == 0 {
		return p.Block("投票ページのURL、または告知文をそのまま貼り付けてください（空行を2回で確定）:")
	}
	var b strings.Builder
	for _, a := range args {
		switch {
		case a == "-":
			// 標準入力は prompt が読んでいるので、そこから EOF まで受け取る
			for l := range p.Lines() {
				b.WriteString(l)
				b.WriteString("\n")
			}
		case strings.HasPrefix(a, "http://") || strings.HasPrefix(a, "https://"):
			b.WriteString(a)
		default:
			data, err := os.ReadFile(a)
			if err != nil {
				return "", fmt.Errorf("URLでもファイルでもありません: %s", a)
			}
			b.Write(data)
		}
		b.WriteString("\n")
	}
	return b.String(), nil
}

func runVote(ctx context.Context, c *api.Client, p *prompt.Prompter, o options, args []string) error {
	text, err := readInput(p, args)
	if err != nil {
		return err
	}
	targets, perrs := target.Parse(text)
	for _, e := range perrs {
		p.Printf("スキップ: %v\n", e)
	}
	if len(targets) == 0 {
		return errors.New("投票先のURLが見つかりませんでした")
	}

	p.Printf("\n投票先を確認しています…\n")
	plans := vote.Resolve(ctx, c, targets)

	cfg, err := config.Load(o.configPath)
	if err != nil {
		return fmt.Errorf("設定ファイルを読めません: %w", err)
	}

	c = c.WithOrigin(targets[0].Origin())
	me, err := auth.EnsureSession(ctx, c, p, o.relogin)
	if err != nil {
		return err
	}
	prof, err := c.CrossCampusProfile(ctx)
	if err != nil {
		return fmt.Errorf("Cross Campus のプロフィールを取得できません: %w", err)
	}
	if miss := vote.MissingProfileFields(prof); len(miss) > 0 {
		return fmt.Errorf("Cross Campus のプロフィールに未入力の項目があります（%s）。Cross Campus で入力してから再実行してください", strings.Join(miss, "・"))
	}
	for _, pl := range plans {
		if pl.Err == nil && pl.Org.ID == prof.OrganizationID {
			pl.Err = errors.New("自分の大学への投票（「学生」区分）はこのツールでは未対応です。サイトから投票してください")
		}
	}
	var ok []*vote.Plan
	for _, pl := range plans {
		if pl.Err == nil {
			ok = append(ok, pl)
		}
	}
	if len(ok) == 0 {
		printPlans(p, plans)
		return errors.New("投票できる対象がありません")
	}

	// auth_token: Cross Campus で電話番号認証済みならSMS認証は不要
	authToken := prof.AuthToken
	if !prof.Profile.PhoneNumberVerified || authToken == "" {
		authToken = ""
		if o.dryRun {
			p.Printf("（Cross Campus で電話番号が未認証のため、実際の送信時はSMS認証が必要です）\n")
		} else {
			p.Printf("\nCross Campus で電話番号が未認証のため、SMS認証を行います。\n")
			if authToken, err = vote.SMSAuth(ctx, c, p, prof.Profile.PhoneNumber); err != nil {
				return err
			}
		}
	}

	// 参加登録済み（＝投票済み）の大学を先に調べ、全部済みなら何も聞かずに終わる
	pending := 0
	for _, pl := range ok {
		if authToken == "" {
			pending++
			continue
		}
		pid, err := vote.Registered(ctx, c, pl, authToken)
		if err != nil {
			p.Printf("（%s の参加登録状況を確認できませんでした: %v）\n", pl.Title(), err)
			pending++
			continue
		}
		reg := pid != ""
		pl.Registered = &reg
		if !reg {
			pending++
		}
	}
	if pending == 0 {
		printPlans(p, plans)
		p.Printf("\nすべて投票済みです。\n")
		return nil
	}

	answers, asked, err := vote.ResolveAnswers(p, cfg, prof, false)
	if err != nil {
		return err
	}
	if answers.Message, err = vote.ResolveMessage(p, o.message, target.SuggestMessage(text), cfg.Message, o.yes); err != nil {
		return err
	}
	if err := answers.Validate(timeNow()); err != nil {
		return err
	}
	if asked && !o.yes {
		if save, err := p.Confirm(fmt.Sprintf("今回の回答を %s に保存して次回から省略しますか？", o.configPath), true); err != nil {
			return err
		} else if save {
			vote.ApplyToConfig(cfg, answers, prof)
			if err := config.Save(o.configPath, cfg); err != nil {
				p.Printf("（保存に失敗しました: %v）\n", err)
			}
		}
	}

	pr := prof.Profile
	p.Printf("\n===== 送信内容 =====\n")
	p.Printf("投票者: %s %s（%s・他大学の学生 / %s）\n", pr.LastName, pr.FirstName, pr.UniversityName, me.Email)
	for _, l := range answers.Summary() {
		p.Printf("  %s\n", l)
	}
	printPlans(p, plans)

	if o.dryRun {
		for _, pl := range ok {
			if _, err := vote.BuildRequest(prof, answers, pl.Org, pl.Group.ID, "<auth_token>"); err != nil {
				p.Printf("✗ %s: %v\n", pl.Title(), err)
			}
		}
		p.Printf("\n--dry-run のため送信しません。\n")
		return nil
	}
	if !o.yes {
		p.Printf("\n各大学のキャンペーン規約・プライバシーポリシー: <大学サイト>/terms-of-service , /privacy-policy\n")
		yes, err := p.Confirm(fmt.Sprintf("キャンペーン規約及びプライバシーポリシーに同意し、%d件の投票を送信しますか？", pending), false)
		if err != nil {
			return err
		}
		if !yes {
			return errors.New("中止しました")
		}
	}

	p.Printf("\n")
	failed := 0
	for _, pl := range ok {
		// 参加登録済みでも投票だけ失敗していた場合に備えて送る（投票済みなら 409 でスキップ扱い）
		r := vote.Submit(ctx, c, pl, prof, answers, authToken)
		switch {
		case r.AlreadyVoted:
			p.Printf("- %s: この大学では投票済みのためスキップ\n", pl.Title())
		case r.Err != nil && r.Voted:
			p.Printf("△ %s: %v\n", pl.Title(), r.Err)
		case r.Err != nil:
			failed++
			p.Printf("✗ %s: %v\n", pl.Title(), r.Err)
		default:
			extra := ""
			if r.Message {
				extra = "（メッセージ送信済み）"
			}
			if r.Note != "" {
				extra += "（" + r.Note + "）"
			}
			p.Printf("✓ %s: 投票しました%s\n", pl.Title(), extra)
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d件の投票に失敗しました", failed)
	}
	return nil
}

var timeNow = time.Now

func printPlans(p *prompt.Prompter, plans []*vote.Plan) {
	p.Printf("投票先:\n")
	for i, pl := range plans {
		if pl.Err != nil {
			p.Printf("  %d. ✗ %s — %v\n", i+1, pl.Title(), pl.Err)
			continue
		}
		status := ""
		if pl.Registered != nil && *pl.Registered {
			status = " — 投票済み"
		}
		p.Printf("  %d. %s（現在 %d票）%s\n", i+1, pl.Title(), pl.Group.TotalVoteCount, status)
	}
}

func runConfig(ctx context.Context, c *api.Client, p *prompt.Prompter, o options) error {
	cfg, err := config.Load(o.configPath)
	if err != nil {
		return err
	}
	if _, err := auth.EnsureSession(ctx, c, p, o.relogin); err != nil {
		return err
	}
	prof, err := c.CrossCampusProfile(ctx)
	if err != nil {
		return fmt.Errorf("Cross Campus のプロフィールを取得できません: %w", err)
	}
	a, _, err := vote.ResolveAnswers(p, cfg, prof, true)
	if err != nil {
		return err
	}
	msg, err := p.String("応援メッセージの既定値（告知文に指定があればそちらを優先）", cfg.Message)
	if err != nil {
		return err
	}
	vote.ApplyToConfig(cfg, a, prof)
	cfg.Message = msg
	if err := config.Save(o.configPath, cfg); err != nil {
		return err
	}
	p.Printf("%s に保存しました。\n", o.configPath)
	return nil
}
