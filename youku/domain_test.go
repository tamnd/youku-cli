package youku

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "youku" {
		t.Errorf("Scheme = %q, want youku", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "youku" {
		t.Errorf("Identity.Binary = %q, want youku", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		in      string
		wantTyp string
		wantID  string
		wantErr bool
	}{
		{"XNjI5MjA1NzA0", "show", "XNjI5MjA1NzA0", false},
		{"https://v.youku.com/v_show/id_XNjI5MjA1NzA0.html", "show", "XNjI5MjA1NzA0", false},
		{"", "", "", true},
		{"!!invalid!!", "", "", true},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("Classify(%q): expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("Classify(%q): unexpected error %v", tc.in, err)
			continue
		}
		if typ != tc.wantTyp || id != tc.wantID {
			t.Errorf("Classify(%q) = (%q, %q), want (%q, %q)",
				tc.in, typ, id, tc.wantTyp, tc.wantID)
		}
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("show", "XNjI5MjA1NzA0")
	want := ShowBase + "XNjI5MjA1NzA0.html"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}

	_, err = Domain{}.Locate("unknown", "x")
	if err == nil {
		t.Error("expected error for unknown uriType")
	}
}

func TestExtractShowID(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://v.youku.com/v_show/id_XNjI5MjA1NzA0.html", "XNjI5MjA1NzA0"},
		{"https://v.youku.com/v_show/id_abc123", "abc123"},
		{"other", ""},
	}
	for _, tc := range cases {
		got := extractShowID(tc.in)
		if got != tc.want {
			t.Errorf("extractShowID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
