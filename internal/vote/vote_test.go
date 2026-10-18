package vote

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/api"
	"github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/internal/target"
)

const (
	orgID   = "11111111-1111-1111-1111-111111111111"
	groupID = "22222222-2222-2222-2222-222222222222"
	partID  = "33333333-3333-3333-3333-333333333333"
	voteID  = "44444444-4444-4444-4444-444444444444"
)

type call struct {
	Method, Path, Origin, Session string
	Body                          map[string]any
}

// fakeAPI はサイトのフロントエンドから読み取った応答を返す偽サーバー。
func fakeAPI(t *testing.T, registered bool, voteStatus int) (*httptest.Server, *[]call) {
	var mu sync.Mutex
	var calls []call
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var body map[string]any
		if len(b) > 0 {
			if err := json.Unmarshal(b, &body); err != nil {
				t.Errorf("%s %s: body is not JSON: %s", r.Method, r.URL.Path, b)
			}
			if ct := r.Header.Get("Content-Type"); ct != "application/json;charset=utf-8" {
				t.Errorf("Content-Type = %q", ct)
			}
		}
		mu.Lock()
		calls = append(calls, call{r.Method, r.URL.Path, r.Header.Get("Origin"), r.Header.Get("X-CC-Session-ID"), body})
		mu.Unlock()
		base := "/gc2026/gc2026_organizations/" + orgID
		switch r.Method + " " + r.URL.Path {
		case "GET /gc2026/gc2026_organizations/by_slug/usp":
			json.NewEncoder(w).Encode(map[string]any{"id": orgID, "slug": "usp", "name": "テスト大学"})
		case "GET " + base + "/gc2026_groups":
			json.NewEncoder(w).Encode([]map[string]any{{"id": groupID, "name": "テスト部", "total_vote_count": 3}})
		case "POST " + base + "/gc2026_participants/other_university_bachelor_student":
			if registered {
				t.Errorf("should not create participant when already registered")
			}
			json.NewEncoder(w).Encode(map[string]any{"id": partID})
		case "POST " + base + "/gc2026_participants/me":
			if !registered {
				w.WriteHeader(404)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"id": partID})
		case "POST " + base + "/gc2026_participants/" + partID + "/gc2026_votes":
			w.WriteHeader(voteStatus)
			if voteStatus == 201 {
				json.NewEncoder(w).Encode(map[string]any{"gc2026_vote_id": voteID})
			} else {
				json.NewEncoder(w).Encode(map[string]any{"code": voteStatus, "message": "Conflict"})
			}
		case "POST " + base + "/gc2026_participants/" + partID + "/gc2026_votes/" + voteID + "/gc2026_message":
			w.WriteHeader(201)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, &calls
}

func testProfile() *api.CrossCampusProfile {
	var p api.CrossCampusProfile
	p.Profile.LastName, p.Profile.FirstName = "山田", "太郎"
	p.Profile.Email = "taro@example.com"
	p.Profile.Gender = "prefer_not_to_say"
	p.Profile.PrefectureID = "pref-okinawa"
	p.Profile.DateOfBirth = "2005-04-01"
	p.Profile.PhoneNumber = "+819000000000"
	p.Profile.PhoneNumberVerified = true
	p.Profile.Faculty = "工学部"
	p.Profile.Department = "工学科"
	p.Profile.Grade = "bachelor_2"
	p.UniversityID = "univ-1"
	p.OrganizationID = "home-org"
	p.AuthToken = "tok"
	return &p
}

func testAnswers() *Answers {
	return &Answers{
		PublicationConsent:   true,
		GraduationFiscalYear: CurrentFiscalYear(time.Now()) + 1,
		VoteReason:           "campaign_connection",
		Sources:              []string{"student"},
		Grade:                "bachelor_2",
		Faculty:              "工学部",
		Department:           "工学科",
		Message:              "応援しています",
	}
}

func client(srv *httptest.Server) *api.Client {
	c := api.New()
	c.BaseURL = srv.URL
	c.SessionID = "sess"
	return c
}

func TestResolveAndSubmit(t *testing.T) {
	srv, calls := fakeAPI(t, false, 201)
	c := client(srv)
	targets := []target.Target{
		{Host: "usp.2026.giving-campaign.jp", Slug: "usp", GroupID: groupID},
		{Host: "usp.2026.giving-campaign.jp", Slug: "usp", GroupID: "55555555-5555-5555-5555-555555555555"},
	}
	plans := Resolve(context.Background(), c, targets)
	if plans[0].Err != nil || plans[0].Group.Name != "テスト部" {
		t.Fatalf("plan[0] = %+v", plans[0])
	}
	if plans[1].Err == nil {
		t.Fatalf("unknown group should fail")
	}

	r := Submit(context.Background(), c, plans[0], testProfile(), testAnswers(), "tok")
	if r.Err != nil || !r.Voted || !r.Message {
		t.Fatalf("result = %+v", r)
	}

	var post []call
	for _, cl := range *calls {
		if cl.Method == "POST" && !strings.HasSuffix(cl.Path, "/me") {
			post = append(post, cl)
		}
		if cl.Origin != "https://usp.2026.giving-campaign.jp" {
			t.Errorf("%s: Origin = %q", cl.Path, cl.Origin)
		}
	}
	if len(post) != 3 {
		t.Fatalf("POST calls = %d, want 3", len(post))
	}
	for _, cl := range post {
		if cl.Session != "sess" {
			t.Errorf("%s: X-CC-Session-ID = %q", cl.Path, cl.Session)
		}
		if cl.Body["auth_token"] != "tok" {
			t.Errorf("%s: auth_token = %v", cl.Path, cl.Body["auth_token"])
		}
	}

	prof := post[0].Body["profile"].(map[string]any)
	wantProf := map[string]any{
		"name": "山田 太郎", "name_kana": " ", "gender": "prefer_not_to_say", "birthday": "2005-04-01",
		"email": "taro@example.com", "phone_number": "+819000000000", "hometown_prefecture_id": "pref-okinawa",
		"university_id": "univ-1", "faculty": "工学部", "department": "工学科", "bvp_shown": true, "bvp_interest": false,
	}
	for k, v := range wantProf {
		if prof[k] != v {
			t.Errorf("profile.%s = %#v, want %#v", k, prof[k], v)
		}
	}
	if _, ok := prof["custom_profile_row"]; ok {
		t.Errorf("custom_profile_row should be omitted")
	}
	q := prof["questionnaire"].(map[string]any)
	if q["version"] != "ksr97-v1" || q["support_target_group_id"] != groupID || q["grade"] != "bachelor_2" ||
		q["vote_reason"] != "campaign_connection" || q["publication_consent"] != true || q["faculty"] != "工学部" {
		t.Errorf("questionnaire = %#v", q)
	}
	if post[0].Body["cross_campus_opt_in"] != false {
		t.Errorf("cross_campus_opt_in = %v", post[0].Body["cross_campus_opt_in"])
	}
	if post[1].Body["gc2026_group_id"] != groupID {
		t.Errorf("vote body = %#v", post[1].Body)
	}
	if post[2].Body["message"] != "応援しています" {
		t.Errorf("message body = %#v", post[2].Body)
	}
}

func TestSubmitRegisteredButNotVoted(t *testing.T) {
	srv, _ := fakeAPI(t, true, 201)
	c := client(srv)
	plans := Resolve(context.Background(), c, []target.Target{{Host: "usp.2026.giving-campaign.jp", Slug: "usp", GroupID: groupID}})
	r := Submit(context.Background(), c, plans[0], testProfile(), testAnswers(), "tok")
	if r.Err != nil || !r.Voted || r.Note == "" {
		t.Fatalf("result = %+v", r)
	}
}

func TestSubmitAlreadyVoted(t *testing.T) {
	srv, _ := fakeAPI(t, true, 409)
	c := client(srv)
	plans := Resolve(context.Background(), c, []target.Target{{Host: "usp.2026.giving-campaign.jp", Slug: "usp", GroupID: groupID}})
	r := Submit(context.Background(), c, plans[0], testProfile(), testAnswers(), "tok")
	if r.Err != nil || r.Voted || !r.AlreadyVoted {
		t.Fatalf("result = %+v", r)
	}
}

func TestOneVotePerUniversity(t *testing.T) {
	srv, _ := fakeAPI(t, false, 201)
	plans := Resolve(context.Background(), client(srv), []target.Target{
		{Host: "usp.2026.giving-campaign.jp", Slug: "usp", GroupID: groupID},
		{Host: "usp.2026.giving-campaign.jp", Slug: "usp", GroupID: groupID + "x"},
	})
	if plans[0].Err != nil {
		t.Fatal(plans[0].Err)
	}
	if plans[1].Err == nil {
		t.Errorf("second target in same university should be rejected")
	}
}

func TestFiscalYear(t *testing.T) {
	if got := CurrentFiscalYear(time.Date(2026, 3, 31, 12, 0, 0, 0, jst)); got != 2025 {
		t.Errorf("FY(2026-03-31) = %d", got)
	}
	if got := CurrentFiscalYear(time.Date(2026, 4, 1, 0, 0, 0, 0, jst)); got != 2026 {
		t.Errorf("FY(2026-04-01) = %d", got)
	}
	var p api.CrossCampusProfile
	p.Profile.GraduationYear, p.Profile.GraduationMonth = float64(2028), float64(3)
	if got := FiscalYearFromProfile(&p); got != 2027 {
		t.Errorf("FiscalYearFromProfile(2028/3) = %d", got)
	}
	p.Profile.GraduationYear, p.Profile.GraduationMonth = "2028-09", nil
	if got := FiscalYearFromProfile(&p); got != 2028 {
		t.Errorf("FiscalYearFromProfile(2028-09) = %d", got)
	}
}

func TestBirthday(t *testing.T) {
	for in, want := range map[string]string{
		"2005-04-01":                "2005-04-01",
		"2005-03-31T15:00:00Z":      "2005-04-01",
		"2005-04-01T00:00:00+09:00": "2005-04-01",
	} {
		if got, err := birthday(in); err != nil || got != want {
			t.Errorf("birthday(%q) = %q, %v", in, got, err)
		}
	}
}
