package vote

import "github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/prompt"

// フォームの選択肢（値はAPIに送る値、ラベルはサイト上の表示）。

var VoteReasons = []prompt.Option{
	{Value: "target_group_member", Label: "この団体のメンバーであるため"},
	{Value: "prior_connection", Label: "Giving Campaign以前から交流があるため"},
	{Value: "campaign_connection", Label: "Giving Campaignをきっかけに交流が活発化したため"},
	{Value: "support_project", Label: "特に交流等はないが、応援したいと思うプロジェクトであるため"},
	{Value: "other", Label: "その他"},
}

var Sources = []prompt.Option{
	{Value: "student", Label: "本学の学生からの紹介"},
	{Value: "other_individual", Label: "本学の学生以外の個人からの紹介"},
	{Value: "university_official_announcement", Label: "本学による発信（ホームページ・公式SNSなど）"},
	{Value: "giving_campaign_official_announcement", Label: "Giving Campaign運営による発信（プレスリリース・公式SNSなど）"},
	{Value: "media", Label: "テレビ・新聞などのメディア"},
	{Value: "other", Label: "その他"},
}

var Grades = []prompt.Option{
	{Value: "bachelor_1", Label: "学部1年"},
	{Value: "bachelor_2", Label: "学部2年"},
	{Value: "bachelor_3", Label: "学部3年"},
	{Value: "bachelor_4", Label: "学部4年"},
	{Value: "bachelor_5", Label: "学部5年"},
	{Value: "bachelor_6", Label: "学部6年"},
	{Value: "master_1", Label: "修士1年"},
	{Value: "master_2", Label: "修士2年"},
	{Value: "doctor_1", Label: "博士1年"},
	{Value: "doctor_2", Label: "博士2年"},
	{Value: "doctor_3", Label: "博士3年"},
}

var Genders = map[string]string{
	"male":              "男性",
	"female":            "女性",
	"other":             "その他",
	"prefer_not_to_say": "無回答",
}

func validValue(opts []prompt.Option, v string) bool {
	for _, o := range opts {
		if o.Value == v {
			return true
		}
	}
	return false
}

func label(opts []prompt.Option, v string) string {
	for _, o := range opts {
		if o.Value == v {
			return o.Label
		}
	}
	return v
}
