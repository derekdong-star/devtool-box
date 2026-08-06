package service

import "testing"

func TestNormalizeBusinessIdentifier(t *testing.T) {
	tests := []struct {
		name           string
		raw            string
		wantType       string
		wantIdentifier string
		wantErr        bool
	}{
		{
			name:           "email",
			raw:            "  Test.User@Example.COM ",
			wantType:       "email",
			wantIdentifier: "test.user@example.com",
		},
		{
			name:           "uuid",
			raw:            "550e8400-e29b-41d4-a716-446655440000",
			wantType:       "uuid",
			wantIdentifier: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:    "invalid",
			raw:     "not-a-user",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotIdentifier, err := normalizeBusinessIdentifier(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotType != tt.wantType {
				t.Fatalf("type = %q, want %q", gotType, tt.wantType)
			}
			if gotIdentifier != tt.wantIdentifier {
				t.Fatalf("identifier = %q, want %q", gotIdentifier, tt.wantIdentifier)
			}
		})
	}
}

func TestClampSessionLimit(t *testing.T) {
	tests := []struct {
		limit int
		want  int
	}{
		{limit: 0, want: defaultRecentSessionLimit},
		{limit: -1, want: defaultRecentSessionLimit},
		{limit: 50, want: 50},
		{limit: 500, want: maxRecentSessionLimit},
	}

	for _, tt := range tests {
		if got := clampSessionLimit(tt.limit); got != tt.want {
			t.Fatalf("clampSessionLimit(%d) = %d, want %d", tt.limit, got, tt.want)
		}
	}
}
