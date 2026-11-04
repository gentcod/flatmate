package ai

import (
	"context"
	"testing"
)

func TestDraftAgreement(t *testing.T) {
	stub := &stubLLM{replies: []string{
		`{"sections":[
			{"heading":"House rules","points":["Quiet after 10pm.","Chore rota on Sundays."]},
			{"heading":"Bills","points":["Split power equally.","Settle monthly."]},
			{"heading":"Guests","points":["Notice for overnights.","Max two nights a week."]},
			{"heading":"Moving out","points":["30 days notice.","Help find a replacement."]}
		]}`,
	}}
	got, ok := DraftAgreement(context.Background(), stub, "Ada", "Bo", []string{"noise"})
	if !ok || len(got) != 4 {
		t.Fatalf("ok=%v sections=%d", ok, len(got))
	}
	if got[0].Heading != "House rules" || len(got[0].Points) != 2 {
		t.Errorf("first section: %+v", got[0])
	}
	bad := &stubLLM{replies: []string{`{"sections":[]}`}}
	if _, ok := DraftAgreement(context.Background(), bad, "Ada", "Bo", nil); ok {
		t.Error("empty sections: want ok=false")
	}
	tmpl := TemplateAgreement()
	if len(tmpl) != 4 || tmpl[1].Heading != "Bills" {
		t.Errorf("template: %+v", tmpl)
	}
}

func TestParseSearch(t *testing.T) {
	stub := &stubLLM{replies: []string{
		`{"area":"yaba","max_rent":200000,"no_smoking":true,"no_pets":false,"wfh_max":2}`,
	}}
	got, ok := ParseSearch(context.Background(), stub, "quiet non-smoker in yaba under 200k, wfh max 2 days")
	if !ok {
		t.Fatal("want ok=true")
	}
	if got.Area != "yaba" || got.MaxRent != 200000 || got.Smoking != "no" || got.Pets != "" || got.WFHMax != 2 {
		t.Errorf("filters: %+v", got)
	}
	bad := &stubLLM{replies: []string{`{"area":"yaba","max_rent":-5,"no_smoking":false,"no_pets":false,"wfh_max":-1}`}}
	if _, ok := ParseSearch(context.Background(), bad, "x"); ok {
		t.Error("negative rent: want ok=false")
	}
	kw := KeywordSearch("  Yaba ")
	if kw.Area != "Yaba" || kw.WFHMax != -1 || kw.MaxRent != 0 {
		t.Errorf("keyword: %+v", kw)
	}
}
