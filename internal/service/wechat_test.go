package service

import (
	"strings"
	"testing"
)

func TestWeChatServiceFormat(t *testing.T) {
	svc := NewWeChatService()

	got, err := svc.Format("# 标题\n\n正文包含 `code`。\n\n- 条目", "tech")
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	if got.Theme != "tech" {
		t.Fatalf("Theme = %q, want tech", got.Theme)
	}
	for _, want := range []string{"<section", "<h1", "style=", "标题", "正文包含", "<ul"} {
		if !strings.Contains(got.HTML, want) {
			t.Fatalf("HTML missing %q:\n%s", want, got.HTML)
		}
	}
	if !strings.Contains(got.FullHTML, "<!doctype html>") {
		t.Fatalf("FullHTML missing doctype:\n%s", got.FullHTML)
	}
}

func TestWeChatServiceFallsBackToDefaultTheme(t *testing.T) {
	svc := NewWeChatService()

	got, err := svc.Format("content", "missing")
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if got.Theme != defaultWeChatThemeID {
		t.Fatalf("Theme = %q, want %q", got.Theme, defaultWeChatThemeID)
	}
}

func TestWeChatServiceRejectsEmptyMarkdown(t *testing.T) {
	svc := NewWeChatService()

	if _, err := svc.Format("  \n\t", "default"); err == nil {
		t.Fatal("Format() error = nil, want error")
	}
}
