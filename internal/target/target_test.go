package target

import "testing"

const announcement = `【Giving Campaign 相互投票のお願い】

①滋賀県立大学ロボット部コンピュータ研究会
https://usp.2026.giving-campaign.jp/groups?image-type=group-ad&group-id=7e7f87d1-2319-40f9-b2d7-910c4e2f5230&group=7e7f87d1-2319-40f9-b2d7-910c4e2f5230

②東北大学学友会人力飛行部Windnauts
https://tohoku.2026.giving-campaign.jp/groups?image-type=group-ad&group-id=2a9d8f02-c36a-4bc0-ad4f-74d9b6ecb788&group=2a9d8f02-c36a-4bc0-ad4f-74d9b6ecb788

③ https://saitama-u.2026.giving-campaign.jp/form?group_id=1af7fdd1-34a7-42fd-86de-e8a7777c890a
https://usp.2026.giving-campaign.jp/groups?group=7e7f87d1-2319-40f9-b2d7-910c4e2f5230
https://example.com/groups?group=7e7f87d1-2319-40f9-b2d7-910c4e2f5230

【投票時の注意】
投票の際は、コメント欄に必ず「琉球大学 Robotサークル」と記入してください。`

func TestParse(t *testing.T) {
	got, errs := Parse(announcement)
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want 1 error (example.com)", errs)
	}
	want := []Target{
		{Host: "usp.2026.giving-campaign.jp", Slug: "usp", GroupID: "7e7f87d1-2319-40f9-b2d7-910c4e2f5230", Label: "①滋賀県立大学ロボット部コンピュータ研究会"},
		{Host: "tohoku.2026.giving-campaign.jp", Slug: "tohoku", GroupID: "2a9d8f02-c36a-4bc0-ad4f-74d9b6ecb788", Label: "②東北大学学友会人力飛行部Windnauts"},
		{Host: "saitama-u.2026.giving-campaign.jp", Slug: "saitama-u", GroupID: "1af7fdd1-34a7-42fd-86de-e8a7777c890a", Label: "③"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d targets, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("target[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestSuggestMessage(t *testing.T) {
	if got := SuggestMessage(announcement); got != "琉球大学 Robotサークル" {
		t.Errorf("SuggestMessage = %q", got)
	}
}
