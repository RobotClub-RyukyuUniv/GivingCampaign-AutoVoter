// Package vote は Cross Campus のプロフィールと設定から投票リクエストを組み立てて送信する。
package vote

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/api"
)

// Answers はフォームのうち Cross Campus のプロフィールから埋まらない項目への回答。
type Answers struct {
	PublicationConsent   bool
	GraduationFiscalYear int
	VoteReason           string
	Sources              []string
	Grade                string
	Faculty              string
	Department           string
	Message              string
}

// MaxMessageLen はサイトの入力制限（JSの文字列長＝UTF-16単位で1000）。
const MaxMessageLen = 1000

var jst = time.FixedZone("JST", 9*60*60)

// CurrentFiscalYear は日本の年度（4月始まり）。
func CurrentFiscalYear(now time.Time) int {
	now = now.In(jst)
	if now.Month() >= time.April {
		return now.Year()
	}
	return now.Year() - 1
}

// FiscalYearFromProfile は Cross Campus の卒業年月から卒業見込み年度を推定する（3月卒業→前年度）。
func FiscalYearFromProfile(p *api.CrossCampusProfile) int {
	y := toInt(p.Profile.GraduationYear)
	m := toInt(p.Profile.GraduationMonth)
	if s, ok := p.Profile.GraduationYear.(string); ok && len(s) >= 7 && s[4] == '-' {
		y, _ = strconv.Atoi(s[:4])
		m, _ = strconv.Atoi(s[5:7])
	}
	if y == 0 {
		return 0
	}
	if m >= 1 && m < 4 {
		return y - 1
	}
	return y
}

func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(x))
		return n
	}
	return 0
}

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// birthday はプロフィールの date_of_birth を YYYY-MM-DD にする（日時ならJSTの日付）。
func birthday(s string) (string, error) {
	s = strings.TrimSpace(s)
	if dateRe.MatchString(s) {
		return s, nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006/01/02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.In(jst).Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("生年月日の形式が不明です: %q", s)
}

// phoneE164 は +81... 形式の電話番号にする。
func phoneE164(s string) (string, error) {
	s = strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(s))
	switch {
	case strings.HasPrefix(s, "+"):
		return s, nil
	case regexp.MustCompile(`^0\d{9,10}$`).MatchString(s):
		return "+81" + s[1:], nil
	}
	return "", fmt.Errorf("電話番号の形式が不明です: %q", s)
}

// MissingProfileFields は Cross Campus 側で埋めてもらう必要がある必須項目を返す。
func MissingProfileFields(p *api.CrossCampusProfile) []string {
	var miss []string
	pr := p.Profile
	check := func(ok bool, name string) {
		if !ok {
			miss = append(miss, name)
		}
	}
	check(pr.LastName != "" && pr.FirstName != "", "お名前")
	check(Genders[pr.Gender] != "", "性別")
	check(pr.DateOfBirth != "", "生年月日")
	check(pr.Email != "", "メールアドレス")
	check(pr.PhoneNumber != "", "携帯電話番号")
	check(pr.PrefectureID != "", "出身地（都道府県）")
	check(p.UniversityID != "", "大学")
	return miss
}

// BuildRequest は「他大学の学生」としての参加登録リクエストを組み立てる。
func BuildRequest(p *api.CrossCampusProfile, a *Answers, org *api.Organization, groupID, authToken string) (*api.CreateParticipantRequest, error) {
	if miss := MissingProfileFields(p); len(miss) > 0 {
		return nil, fmt.Errorf("Cross Campus のプロフィールに未入力の項目があります: %s", strings.Join(miss, "・"))
	}
	if err := a.Validate(time.Now()); err != nil {
		return nil, err
	}
	bd, err := birthday(p.Profile.DateOfBirth)
	if err != nil {
		return nil, err
	}
	phone, err := phoneE164(p.Profile.PhoneNumber)
	if err != nil {
		return nil, err
	}
	pr := p.Profile
	prof := api.OtherUniversityStudentProfile{
		BVPShown:             true,
		BVPInterest:          false,
		Name:                 pr.LastName + " " + pr.FirstName,
		NameKana:             pr.LastNameKana + " " + pr.FirstNameKana,
		Gender:               pr.Gender,
		Birthday:             bd,
		Email:                pr.Email,
		PhoneNumber:          phone,
		Sources:              a.Sources,
		HometownPrefectureID: pr.PrefectureID,
		UniversityID:         p.UniversityID,
		Faculty:              a.Faculty,
		Department:           a.Department,
		Questionnaire: api.Questionnaire{
			Version:              api.QuestionnaireVersion,
			PublicationConsent:   a.PublicationConsent,
			SupportTargetGroupID: groupID,
			GraduationFiscalYear: a.GraduationFiscalYear,
			Faculty:              a.Faculty,
			Grade:                a.Grade,
			VoteReason:           a.VoteReason,
		},
	}
	if org.CustomProfileColumn != nil && org.CustomProfileColumn.ID != "" {
		prof.CustomProfileRow = &api.CustomProfileRow{ProfileColumnID: org.CustomProfileColumn.ID, Value: true}
	}
	return &api.CreateParticipantRequest{Profile: prof, AuthToken: authToken, CrossCampusOptIn: false}, nil
}

// Validate はサイト側と同じ入力チェックを行う。
func (a *Answers) Validate(now time.Time) error {
	var errs []error
	fy := CurrentFiscalYear(now)
	if a.GraduationFiscalYear < fy || a.GraduationFiscalYear > fy+8 {
		errs = append(errs, fmt.Errorf("卒業/修了見込み年度は %d〜%d の範囲で指定してください（現在: %d）", fy, fy+8, a.GraduationFiscalYear))
	}
	if !validValue(VoteReasons, a.VoteReason) {
		errs = append(errs, fmt.Errorf("投票先団体との関係性が不正です: %q", a.VoteReason))
	}
	if len(a.Sources) == 0 {
		errs = append(errs, errors.New("どのようにしてGiving Campaignを知ったかを1つ以上選んでください"))
	}
	for _, s := range a.Sources {
		if !validValue(Sources, s) {
			errs = append(errs, fmt.Errorf("知ったきっかけの値が不正です: %q", s))
		}
	}
	if !validValue(Grades, a.Grade) {
		errs = append(errs, fmt.Errorf("学年が不正です: %q", a.Grade))
	}
	if strings.TrimSpace(a.Faculty) == "" {
		errs = append(errs, errors.New("学部が未入力です"))
	}
	if strings.TrimSpace(a.Department) == "" {
		errs = append(errs, errors.New("学科が未入力です"))
	}
	if strings.TrimSpace(a.Message) == "" {
		errs = append(errs, errors.New("応援メッセージが未入力です"))
	} else if err := checkMessageLen(a.Message); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
