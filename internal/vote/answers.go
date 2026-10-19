package vote

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/api"
	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/config"
	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/prompt"
)

// ResolveAnswers は設定・プロフィールで埋まらない項目だけを対話で尋ねる。
// reask が true なら設定済みの項目も尋ね直す。asked は対話で答えた項目があったかどうか。
func ResolveAnswers(p *prompt.Prompter, cfg *config.Config, prof *api.CrossCampusProfile, reask bool) (a *Answers, asked bool, err error) {
	a = &Answers{}
	now := time.Now()

	if cfg.SupporterPublicationConsent != nil && !reask {
		a.PublicationConsent = *cfg.SupporterPublicationConsent
	} else {
		def := true
		if cfg.SupporterPublicationConsent != nil {
			def = *cfg.SupporterPublicationConsent
		}
		if a.PublicationConsent, err = p.Confirm("支援者として、お名前・大学との関係性の一般公開・支援先団体への共有に同意しますか？", def); err != nil {
			return nil, false, err
		}
		asked = true
	}

	fy := CurrentFiscalYear(now)
	if cfg.GraduationFiscalYear != 0 && !reask && cfg.GraduationFiscalYear >= fy {
		a.GraduationFiscalYear = cfg.GraduationFiscalYear
	} else {
		def := cfg.GraduationFiscalYear
		if def < fy {
			def = FiscalYearFromProfile(prof)
		}
		for {
			s, err := p.String(fmt.Sprintf("卒業/修了見込み年度（%d〜%d、例: 3月卒業なら卒業年の前年）", fy, fy+8), itoa(def))
			if err != nil {
				return nil, false, err
			}
			if n, err := strconv.Atoi(s); err == nil && n >= fy && n <= fy+8 {
				a.GraduationFiscalYear = n
				break
			}
		}
		asked = true
	}

	if validValue(VoteReasons, cfg.VoteReason) && !reask {
		a.VoteReason = cfg.VoteReason
	} else {
		def := cfg.VoteReason
		if def == "" {
			def = "campaign_connection"
		}
		if a.VoteReason, err = p.Select("投票先団体との関係性", VoteReasons, def); err != nil {
			return nil, false, err
		}
		asked = true
	}

	if len(cfg.Sources) > 0 && !reask {
		a.Sources = cfg.Sources
	} else {
		defs := cfg.Sources
		if len(defs) == 0 {
			defs = []string{"student"}
		}
		if a.Sources, err = p.MultiSelect("どのようにしてGiving Campaignを知りましたか？（「本学」は投票先の大学のこと）", Sources, defs); err != nil {
			return nil, false, err
		}
		asked = true
	}

	switch {
	case validValue(Grades, prof.Profile.Grade):
		a.Grade = prof.Profile.Grade
	case validValue(Grades, cfg.Grade) && !reask:
		a.Grade = cfg.Grade
	default:
		if a.Grade, err = p.Select("学年（Cross Campusのプロフィールに無いため）", Grades, cfg.Grade); err != nil {
			return nil, false, err
		}
		asked = true
	}

	ask := func(profVal, cfgVal, label string) (string, error) {
		if v := strings.TrimSpace(profVal); v != "" {
			return v, nil
		}
		if v := strings.TrimSpace(cfgVal); v != "" && !reask {
			return v, nil
		}
		asked = true
		for {
			v, err := p.String(label+"（Cross Campusのプロフィールに無いため）", cfgVal)
			if err != nil || v != "" {
				return v, err
			}
		}
	}
	if a.Faculty, err = ask(prof.Profile.Faculty, cfg.Faculty, "学部"); err != nil {
		return nil, false, err
	}
	if a.Department, err = ask(prof.Profile.Department, cfg.Department, "学科"); err != nil {
		return nil, false, err
	}
	return a, asked, nil
}

// DefaultMessage は Enter だけで送れる応援メッセージの既定値を決める。
// 設定済みのメッセージを優先し、告知文に指定の文言があってそれを含まない場合だけ指定に合わせた文面にする。
func DefaultMessage(suggested, cfgMsg, university string) string {
	switch {
	case cfgMsg != "" && (suggested == "" || containsLoose(cfgMsg, suggested)):
		return cfgMsg
	case suggested != "":
		return suggested + "です。応援しています！"
	case university != "":
		return university + "です。応援しています！"
	}
	return ""
}

// ResolveMessage は応援メッセージを決める（必須）。-m 指定があればそれを使い、
// なければ既定値を表示して Enter でそのまま使えるようにする。yes なら既定値を確認なしで使う。
func ResolveMessage(p *prompt.Prompter, flagMsg, suggested, cfgMsg, university string, yes bool) (string, error) {
	if m := strings.TrimSpace(flagMsg); m != "" {
		return m, checkMessageLen(m)
	}
	def := DefaultMessage(suggested, cfgMsg, university)
	if yes {
		if def == "" {
			return "", errors.New("応援メッセージが未設定です。-m で指定するか gcvote config で設定してください")
		}
		return def, checkMessageLen(def)
	}
	if suggested != "" {
		p.Printf("告知文の指定: コメント欄に「%s」\n", suggested)
	}
	for {
		m, err := p.String("応援メッセージ（Enterでそのまま）", def)
		if err != nil {
			return "", err
		}
		m = strings.TrimSpace(m)
		if m == "" {
			p.Printf("応援メッセージは必須です。\n")
			continue
		}
		if err := checkMessageLen(m); err != nil {
			p.Printf("%v\n", err)
			continue
		}
		if suggested != "" && !containsLoose(m, suggested) {
			ok, err := p.Confirm(fmt.Sprintf("告知文で指定された「%s」が含まれていません。このまま送りますか？", suggested), false)
			if err != nil {
				return "", err
			}
			if !ok {
				continue
			}
		}
		return m, nil
	}
}

func checkMessageLen(m string) error {
	if n := len(utf16.Encode([]rune(m))); n > MaxMessageLen {
		return fmt.Errorf("応援メッセージは%d文字以下にしてください（%d文字）", MaxMessageLen, n)
	}
	return nil
}

// containsLoose は空白（全角含む）の有無を無視して部分一致を調べる。
func containsLoose(s, sub string) bool {
	strip := func(x string) string { return strings.Join(strings.Fields(strings.ReplaceAll(x, "\u3000", " ")), "") }
	return strings.Contains(strip(s), strip(sub))
}

// ApplyToConfig は回答を設定に書き戻す（保存用）。プロフィール由来の値は保存しない。
func ApplyToConfig(cfg *config.Config, a *Answers, prof *api.CrossCampusProfile) {
	consent := a.PublicationConsent
	cfg.SupporterPublicationConsent = &consent
	cfg.GraduationFiscalYear = a.GraduationFiscalYear
	cfg.VoteReason = a.VoteReason
	cfg.Sources = a.Sources
	if prof.Profile.Grade == "" {
		cfg.Grade = a.Grade
	}
	if strings.TrimSpace(prof.Profile.Faculty) == "" {
		cfg.Faculty = a.Faculty
	}
	if strings.TrimSpace(prof.Profile.Department) == "" {
		cfg.Department = a.Department
	}
	if cfg.Message == "" {
		cfg.Message = a.Message
	}
}

func itoa(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// Summary は確認表示用の回答一覧。
func (a *Answers) Summary() []string {
	consent := "いいえ"
	if a.PublicationConsent {
		consent = "はい"
	}
	var src []string
	for _, s := range a.Sources {
		src = append(src, label(Sources, s))
	}
	msg := a.Message
	return []string{
		"名前等の一般公開・共有への同意: " + consent,
		"卒業/修了見込み年度: " + strconv.Itoa(a.GraduationFiscalYear),
		"学年: " + label(Grades, a.Grade),
		"学部・学科: " + a.Faculty + " " + a.Department,
		"投票先団体との関係性: " + label(VoteReasons, a.VoteReason),
		"知ったきっかけ: " + strings.Join(src, "、"),
		"応援メッセージ: " + msg,
	}
}
