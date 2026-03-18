package i18n

const (
	LangRU = "ru"
	LangTK = "tk"

	DefaultLang = LangRU
)

var translations = map[string]map[string]string{
	LangRU: messagesRU,
	LangTK: messagesTK,
}

// Translate returns the localized version of msg.
// Fallback chain: requested lang → Russian → original string.
func Translate(lang, msg string) string {
	if dict, ok := translations[lang]; ok {
		if t, ok := dict[msg]; ok {
			return t
		}
	}
	if lang != LangRU {
		if t, ok := translations[LangRU][msg]; ok {
			return t
		}
	}
	return msg
}

// TranslateDetails translates all values in a map[string]string (validation details).
func TranslateDetails(lang string, details any) any {
	m, ok := details.(map[string]string)
	if !ok {
		return details
	}
	translated := make(map[string]string, len(m))
	for field, msg := range m {
		translated[field] = Translate(lang, msg)
	}
	return translated
}
