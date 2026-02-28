package publisher

import "testing"

func TestLoadPublishFlow_Zhihu(t *testing.T) {
	flow, err := LoadPublishFlow("zhihu")
	if err != nil {
		t.Fatalf("LoadPublishFlow zhihu: %v", err)
	}
	if flow.SchemaVersion != 1 {
		t.Fatalf("schemaVersion: want 1 got %d", flow.SchemaVersion)
	}
	if flow.Platform != "zhihu" {
		t.Fatalf("platform: want zhihu got %s", flow.Platform)
	}
	if len(flow.Steps) == 0 {
		t.Fatalf("steps should not be empty")
	}
	if flow.Steps[0].Action == "" {
		t.Fatalf("first step missing action")
	}
}

func TestLoadPublishFlow_Baijiahao(t *testing.T) {
	flow, err := LoadPublishFlow("baijiahao")
	if err != nil {
		t.Fatalf("LoadPublishFlow baijiahao: %v", err)
	}
	if flow.Platform != "baijiahao" {
		t.Fatalf("platform: want baijiahao got %s", flow.Platform)
	}
	if len(flow.Steps) == 0 {
		t.Fatalf("steps should not be empty")
	}
}

func TestLoadPublishFlow_Qie(t *testing.T) {
	flow, err := LoadPublishFlow("qie")
	if err != nil {
		t.Fatalf("LoadPublishFlow qie: %v", err)
	}
	if flow.Platform != "qie" {
		t.Fatalf("platform: want qie got %s", flow.Platform)
	}
	if len(flow.Steps) == 0 {
		t.Fatalf("steps should not be empty")
	}
}

func TestInterpolateValue(t *testing.T) {
	article := Article{Title: "T", Content: "C", CoverImage: "I"}
	out := interpolateValue("{{title}}-{{content}}-{{cover_image}}", article, nil)
	if out != "T-C-I" {
		t.Fatalf("unexpected interpolation: %s", out)
	}
}

func TestInterpolateValue_TempVars(t *testing.T) {
	article := Article{}
	tempVars := map[string]string{"temp_cover": "/tmp/cover.png"}
	out := interpolateValue("{{temp_cover}}", article, tempVars)
	if out != "/tmp/cover.png" {
		t.Fatalf("unexpected tempVars interpolation: %s", out)
	}
}
