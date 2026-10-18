package api

import (
	"context"
	"fmt"
	"net/url"
)

type Organization struct {
	ID                  string `json:"id"`
	Slug                string `json:"slug"`
	Name                string `json:"name"`
	CampaignStatus      string `json:"campaign_status"`
	IsDebug             bool   `json:"is_debug"`
	CustomProfileColumn *struct {
		ID string `json:"id"`
	} `json:"custom_profile_column"`
}

func (c *Client) OrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	var o Organization
	if err := c.do(ctx, "GET", "/gc2026/gc2026_organizations/by_slug/"+url.PathEscape(slug), nil, nil, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

type Group struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	TotalVoteCount int    `json:"total_vote_count"`
	OrganizationID string `json:"gc2026_organization_id"`
}

func (c *Client) Groups(ctx context.Context, orgID string) ([]Group, error) {
	var gs []Group
	err := c.do(ctx, "GET", orgPath(orgID)+"/gc2026_groups", nil, nil, &gs)
	return gs, err
}

// CrossCampusProfile は Cross Campus に登録済みのプロフィール。
type CrossCampusProfile struct {
	Profile struct {
		LastName              string `json:"last_name"`
		FirstName             string `json:"first_name"`
		LastNameKana          string `json:"last_name_kana"`
		FirstNameKana         string `json:"first_name_kana"`
		Email                 string `json:"email"`
		Gender                string `json:"gender"`
		PrefectureID          string `json:"prefecture_id"`
		DateOfBirth           string `json:"date_of_birth"`
		PhoneNumber           string `json:"phone_number"`
		PhoneNumberVerified   bool   `json:"phone_number_verified"`
		AcademicMajorCategory string `json:"academic_major_category"`
		Faculty               string `json:"faculty"`
		Department            string `json:"department"`
		GraduationYear        any    `json:"graduation_year"`
		GraduationMonth       any    `json:"graduation_month"`
		Grade                 string `json:"grade"`
		UniversityName        string `json:"university_name"`
	} `json:"profile"`
	OrganizationID string `json:"gc2026_organization_id"`
	UniversityID   string `json:"gc2026_university_id"`
	AuthToken      string `json:"auth_token"`
}

func (c *Client) CrossCampusProfile(ctx context.Context) (*CrossCampusProfile, error) {
	var p CrossCampusProfile
	if err := c.do(ctx, "GET", "/gc2026/cross-campus/profile", nil, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// --- SMS認証（Cross Campus で電話番号が未認証のときだけ使う） ---

func (c *Client) SendSMS(ctx context.Context, phoneE164 string) (string, error) {
	var res struct {
		ID string `json:"gc2026_phone_number_verification_id"`
	}
	err := c.do(ctx, "POST", "/gc2026/gc2026_phone_number_verifications", nil, map[string]string{"phone_number": phoneE164}, &res)
	return res.ID, err
}

func (c *Client) VerifySMS(ctx context.Context, verificationID, code string) (string, error) {
	var res struct {
		AuthToken string `json:"authToken"`
	}
	err := c.do(ctx, "POST", "/gc2026/gc2026_phone_number_verifications/"+url.PathEscape(verificationID)+"/verify", nil, map[string]string{"sms_code": code}, &res)
	return res.AuthToken, err
}

// --- 参加登録・投票・メッセージ ---

// Questionnaire は2026年版フォームのアンケート部分。
type Questionnaire struct {
	Version              string `json:"version"`
	PublicationConsent   bool   `json:"publication_consent"`
	SupportTargetGroupID string `json:"support_target_group_id"`
	GraduationFiscalYear int    `json:"graduation_fiscal_year"`
	Faculty              string `json:"faculty,omitempty"`
	Grade                string `json:"grade"`
	VoteReason           string `json:"vote_reason"`
}

const QuestionnaireVersion = "ksr97-v1"

type CustomProfileRow struct {
	ProfileColumnID string `json:"profile_column_id"`
	Value           bool   `json:"value"`
}

// OtherUniversityStudentProfile は「他大学の学生」として参加登録するときのプロフィール。
type OtherUniversityStudentProfile struct {
	BVPShown             bool              `json:"bvp_shown"`
	BVPInterest          bool              `json:"bvp_interest"`
	Name                 string            `json:"name"`
	NameKana             string            `json:"name_kana"`
	Gender               string            `json:"gender"`
	Birthday             string            `json:"birthday"`
	Email                string            `json:"email"`
	PhoneNumber          string            `json:"phone_number"`
	Sources              []string          `json:"gc2026_sources"`
	HometownPrefectureID string            `json:"hometown_prefecture_id"`
	CustomProfileRow     *CustomProfileRow `json:"custom_profile_row,omitempty"`
	UniversityID         string            `json:"university_id"`
	Faculty              string            `json:"faculty"`
	Department           string            `json:"department"`
	Questionnaire        Questionnaire     `json:"questionnaire"`
}

type CreateParticipantRequest struct {
	Profile          OtherUniversityStudentProfile `json:"profile"`
	AuthToken        string                        `json:"auth_token"`
	CrossCampusOptIn bool                          `json:"cross_campus_opt_in"`
}

func (c *Client) CreateOtherUniversityStudent(ctx context.Context, orgID string, req *CreateParticipantRequest) (string, error) {
	var res struct {
		ID string `json:"id"`
	}
	err := c.do(ctx, "POST", orgPath(orgID)+"/gc2026_participants/other_university_bachelor_student", nil, req, &res)
	return res.ID, err
}

// MyParticipant はこの大学に参加登録済みならその参加者IDを返す。未登録なら HTTP 404。
func (c *Client) MyParticipant(ctx context.Context, orgID, authToken string) (string, error) {
	var res struct {
		ID string `json:"id"`
	}
	err := c.do(ctx, "POST", orgPath(orgID)+"/gc2026_participants/me", nil, map[string]string{"auth_token": authToken}, &res)
	return res.ID, err
}

func (c *Client) CreateVote(ctx context.Context, orgID, participantID, groupID, authToken string) (string, error) {
	var res struct {
		ID string `json:"gc2026_vote_id"`
	}
	body := map[string]string{"gc2026_group_id": groupID, "auth_token": authToken}
	err := c.do(ctx, "POST", participantPath(orgID, participantID)+"/gc2026_votes", nil, body, &res)
	return res.ID, err
}

func (c *Client) PostMessage(ctx context.Context, orgID, participantID, voteID, message, authToken string) error {
	body := map[string]string{"message": message, "auth_token": authToken}
	return c.do(ctx, "POST", fmt.Sprintf("%s/gc2026_votes/%s/gc2026_message", participantPath(orgID, participantID), url.PathEscape(voteID)), nil, body, nil)
}

func orgPath(orgID string) string {
	return "/gc2026/gc2026_organizations/" + url.PathEscape(orgID)
}

func participantPath(orgID, participantID string) string {
	return orgPath(orgID) + "/gc2026_participants/" + url.PathEscape(participantID)
}
