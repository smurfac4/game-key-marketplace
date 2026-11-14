package account

import "testing"

func TestProfileValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		p     Profile
		valid bool
	}{
		{"empty", Profile{}, true}, {"unicode", Profile{DisplayName: "Игрок 12", CountryCode: "LT", Avatar: "orbit"}, true},
		{"short", Profile{DisplayName: "a"}, false}, {"markup", Profile{DisplayName: "<script>"}, false}, {"unknown country", Profile{CountryCode: "ZZ"}, false}, {"avatar URL", Profile{Avatar: "https://foreign.invalid/avatar"}, false}, {"long name", Profile{DisplayName: "abcdefghijklmnopqrstuvwxyzabcdefg"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Validate(tc.p)
			if (err == nil) != tc.valid {
				t.Fatal("unexpected validation result")
			}
		})
	}
	p, err := Validate(Profile{DisplayName: "  Player 12  "})
	if err != nil || p.DisplayName != "Player 12" {
		t.Fatal("name not normalized")
	}
}
func TestCompletion(t *testing.T) {
	for _, tc := range []struct {
		p                  Profile
		completed, percent int
	}{{Profile{}, 2, 33}, {Profile{DisplayName: "Player"}, 3, 50}, {Profile{DisplayName: "Player", CountryCode: "LT", Avatar: "orbit", PreferredPlatform: "Steam"}, 6, 100}} {
		p := Progress(tc.p)
		if p.Completed != tc.completed || p.Percent != tc.percent || p.Total != 6 {
			t.Fatal("incorrect completion")
		}
	}
}
