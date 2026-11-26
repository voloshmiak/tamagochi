package config

import "testing"

func TestLoad_TTLLimits(t *testing.T) {
	tests := []struct {
		name    string
		access  string
		refresh string
		wantErr bool
	}{
		{"defaults", "", "", false},
		{"max allowed", "15m", "720h", false},
		{"access too long", "16m", "", true},
		{"refresh too long", "", "721h", true},
		{"access zero", "0s", "", true},
		{"refresh negative", "", "-1h", true},
		{"access garbage", "abc", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("APP_ENV", "dev")
			t.Setenv("ACCESS_TTL", tc.access)
			t.Setenv("REFRESH_TTL", tc.refresh)
			_, err := Load()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
