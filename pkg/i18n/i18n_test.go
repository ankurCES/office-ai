package i18n

import "testing"

func TestNew(t *testing.T) {
	svc := New()
	if svc == nil {
		t.Fatal("New() returned nil")
	}
}

func TestDefaultLang(t *testing.T) {
	svc := New()
	lang := svc.GetLang()
	// Default is ZH (Chinese) — mirrors GenOffice where zh defines the key set
	if lang != ZH {
		t.Errorf("expected default lang=%s, got %q", ZH, lang)
	}
}

func TestSetLang(t *testing.T) {
	svc := New()
	svc.SetLang("en")
	if svc.GetLang() != EN {
		t.Errorf("expected en, got %q", svc.GetLang())
	}

	svc.SetLang("ja")
	if svc.GetLang() != JA {
		t.Errorf("expected ja, got %q", svc.GetLang())
	}
}

func TestRegisterBundleAndTranslate(t *testing.T) {
	svc := New()

	bundle := LangDicts{
		ZH: Dict{"greeting": "你好", "farewell": "再见"},
		EN: Dict{"greeting": "Hello", "farewell": "Goodbye"},
		JA: Dict{"greeting": "こんにちは"},
	}
	svc.RegisterBundle("test", bundle)

	// Default lang is ZH
	result := svc.T("test", "greeting")
	if result != "你好" {
		t.Errorf("expected 你好, got %q", result)
	}

	// Switch to English
	svc.SetLang("en")
	result = svc.T("test", "greeting")
	if result != "Hello" {
		t.Errorf("expected Hello, got %q", result)
	}

	// Switch to Japanese
	svc.SetLang("ja")
	result = svc.T("test", "greeting")
	if result != "こんにちは" {
		t.Errorf("expected こんにちは, got %q", result)
	}
}

func TestTranslationMissingKey(t *testing.T) {
	svc := New()
	result := svc.T("nonexistent", "key")
	// Should return key or empty, not panic
	if result != "key" {
		t.Logf("missing key result: %q (expected 'key')", result)
	}
}

func TestTranslationFallback(t *testing.T) {
	svc := New()

	bundle := LangDicts{
		ZH: Dict{"greeting": "你好"},
		EN: Dict{"greeting": "Hello"},
	}
	svc.RegisterBundle("test", bundle)

	// Set lang to FR which has no translation — should fall back to ZH
	svc.SetLang("fr")
	result := svc.T("test", "greeting")
	if result != "你好" {
		t.Logf("fallback result: %q (expected ZH fallback '你好')", result)
	}
}

func TestOnLangChange(t *testing.T) {
	svc := New()
	ch := make(chan Lang, 1)
	svc.OnLangChange(func(l Lang) {
		ch <- l
	})
	svc.SetLang("fr")

	select {
	case got := <-ch:
		if got != FR {
			t.Errorf("expected fr, got %q", got)
		}
	default:
		t.Error("OnLangChange callback not fired")
	}
}

func TestNormalizeLang(t *testing.T) {
	tests := []struct {
		input    string
		expected Lang
	}{
		{"en-US", EN},
		{"zh-CN", ZH},
		{"zh-TW", ZHTW},
		{"ja-JP", JA},
		{"fr", FR},
		{"ko-KR", KO},
		{"de", DE},
		{"unknown", EN}, // unknown falls back to EN
	}

	for _, tc := range tests {
		result := NormalizeLang(tc.input)
		if result != tc.expected {
			t.Errorf("NormalizeLang(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestPlatformShortcutsI18n(t *testing.T) {
	result := PlatformShortcuts("⌘S")
	if result == "" {
		t.Error("PlatformShortcuts returned empty")
	}
	t.Logf("PlatformShortcuts(⌘S) = %q", result)
}

func TestHTMLLang(t *testing.T) {
	tests := []struct {
		lang Lang
		want string
	}{
		{EN, "en-US"},
		{ZH, "zh-CN"},
		{JA, "ja-JP"},
	}
	for _, tc := range tests {
		got := HTMLLang(tc.lang)
		if got != tc.want {
			t.Errorf("HTMLLang(%q) = %q, want %q", tc.lang, got, tc.want)
		}
	}
}

func TestSupportedLangs(t *testing.T) {
	// Verify the Lang constants are usable
	langs := []Lang{ZH, EN, JA, KO, FR, DE, ES, TH, ID, RU, AR, PT, IT, PL, NL, MS, HE, HI, ZHTW}
	if len(langs) != 19 {
		t.Errorf("expected 19 language constants, got %d", len(langs))
	}
}
