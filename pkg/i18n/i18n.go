// Package i18n provides internationalization support matching GenOffice's 19-language system.
// zh defines the key set; all other languages must cover exactly the same keys.
package i18n

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// Lang represents a supported UI language (BCP-47 subset).
type Lang string

const (
	ZH   Lang = "zh"
	EN   Lang = "en"
	JA   Lang = "ja"
	KO   Lang = "ko"
	FR   Lang = "fr"
	DE   Lang = "de"
	ES   Lang = "es"
	TH   Lang = "th"
	ID   Lang = "id"
	RU   Lang = "ru"
	AR   Lang = "ar"
	PT   Lang = "pt"
	IT   Lang = "it"
	PL   Lang = "pl"
	NL   Lang = "nl"
	MS   Lang = "ms"
	HE   Lang = "he"
	HI   Lang = "hi"
	ZHTW Lang = "zh-TW"
)

// Langs is the list of all supported languages.
var Langs = []Lang{ZH, EN, JA, KO, FR, DE, ES, TH, ID, RU, AR, PT, IT, PL, NL, MS, HE, HI, ZHTW}

// Dict is a map of translation keys to their translated strings.
type Dict map[string]string

// LangDicts maps each language to its dictionary. zh defines the canonical key set.
type LangDicts map[Lang]Dict

// Service holds the current UI language and provides translation methods.
type Service struct {
	mu        sync.RWMutex
	lang      Lang
	listeners []func(Lang)
	bundles   map[string]LangDicts // registered translation bundles by namespace
}

// New creates a new i18n Service with default language zh.
func New() *Service {
	return &Service{
		lang:    ZH,
		bundles: make(map[string]LangDicts),
	}
}

// GetLang returns the current UI language.
func (s *Service) GetLang() Lang {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lang
}

// SetLang changes the UI language and notifies listeners.
func (s *Service) SetLang(lang Lang) {
	s.mu.Lock()
	if s.lang == lang {
		s.mu.Unlock()
		return
	}
	s.lang = lang
	listeners := make([]func(Lang), len(s.listeners))
	copy(listeners, s.listeners)
	s.mu.Unlock()
	for _, fn := range listeners {
		fn(lang)
	}
}

// OnLangChange registers a callback for language changes.
func (s *Service) OnLangChange(fn func(Lang)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, fn)
}

// RegisterBundle registers a named set of translations.
func (s *Service) RegisterBundle(namespace string, dicts LangDicts) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bundles[namespace] = dicts
}

// T translates a key in the given namespace using the current language.
func (s *Service) T(namespace, key string, params ...string) string {
	s.mu.RLock()
	lang := s.lang
	bundle, ok := s.bundles[namespace]
	s.mu.RUnlock()

	if !ok {
		return key
	}
	dict, ok := bundle[lang]
	if !ok {
		dict = bundle[ZH] // fallback to zh
	}
	text, ok := dict[key]
	if !ok {
		return key
	}
	return formatParams(text, params)
}

// NormalizeLang maps a raw locale string to a supported Lang.
func NormalizeLang(raw string) Lang {
	val := strings.TrimSpace(strings.ToLower(raw))
	if val == "" {
		return EN
	}
	// Traditional Chinese
	if matched, _ := regexp.MatchString(`^zh[-_](tw|hk|mo|hant)`, val); matched {
		return ZHTW
	}
	for _, lang := range Langs {
		if lang != EN && lang != ZHTW && strings.HasPrefix(val, string(lang)) {
			return lang
		}
	}
	// Legacy codes
	if strings.HasPrefix(val, "in") {
		return ID
	}
	if strings.HasPrefix(val, "iw") {
		return HE
	}
	return EN
}

// HTMLLang returns the BCP-47 tag for document.documentElement.lang.
func HTMLLang(lang Lang) string {
	m := map[Lang]string{
		ZH: "zh-CN", EN: "en-US", JA: "ja-JP", KO: "ko-KR",
		FR: "fr-FR", DE: "de-DE", ES: "es-ES", TH: "th-TH",
		ID: "id-ID", RU: "ru-RU", AR: "ar-SA", PT: "pt-BR",
		IT: "it-IT", PL: "pl-PL", NL: "nl-NL", MS: "ms-MY",
		HE: "he-IL", HI: "hi-IN", ZHTW: "zh-TW",
	}
	if v, ok := m[lang]; ok {
		return v
	}
	return "en-US"
}

// PlatformShortcuts converts Mac shortcut notation to platform-appropriate form.
func PlatformShortcuts(text string) string {
	if runtime.GOOS == "darwin" {
		return text
	}
	return MacShortcutsToWin(text)
}

// MacShortcutsToWin rewrites Mac shortcut notation (⌘⌥⇧) to Ctrl/Alt/Shift form.
func MacShortcutsToWin(text string) string {
	// Simplified conversion - full implementation would use regex like the TS version
	r := strings.NewReplacer("⌘", "Ctrl+", "⌥", "Alt+", "⇧", "Shift+", "⌃", "Ctrl+")
	return r.Replace(text)
}

func formatParams(template string, params []string) string {
	if len(params) == 0 || len(params)%2 != 0 {
		return template
	}
	result := template
	for i := 0; i < len(params); i += 2 {
		result = strings.ReplaceAll(result, fmt.Sprintf("{%s}", params[i]), params[i+1])
	}
	return result
}
