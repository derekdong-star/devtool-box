package service

import "testing"

func TestExtractSessionCookie(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantName   string
		wantValue  string
		wantErr    bool
	}{
		{
			name:      "cookie header with session",
			input:     "Cookie: foo=bar; session=abc123; token=xyz",
			wantName:  "session",
			wantValue: "abc123",
		},
		{
			name:      "set cookie header",
			input:     "Set-Cookie: session=abc123; Path=/; HttpOnly; SameSite=Lax",
			wantName:  "session",
			wantValue: "abc123",
		},
		{
			name:      "common session alias",
			input:     "foo=bar; connect.sid=s%3Atest.signature; theme=dark",
			wantName:  "connect.sid",
			wantValue: "s%3Atest.signature",
		},
		{
			name:    "single non session cookie should error",
			input:   "remember_token=abc123",
			wantErr: true,
		},
		{
			name:      "raw session value fallback",
			input:     "MTIzfGFiY3xzaWc",
			wantName:  "session",
			wantValue: "MTIzfGFiY3xzaWc",
		},
		{
			name:    "multiple cookies without session candidate",
			input:   "foo=bar; token=xyz",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotName, gotValue, err := extractSessionCookie(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotName != tc.wantName || gotValue != tc.wantValue {
				t.Fatalf("got (%q, %q), want (%q, %q)", gotName, gotValue, tc.wantName, tc.wantValue)
			}
		})
	}
}
