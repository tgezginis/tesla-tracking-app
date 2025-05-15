package i18n

import (
	"os"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)


const (
	LangEnglish = "en"
	LangTurkish = "tr"
)

var (
	
	CurrentLang = LangEnglish

	
	textMap = map[string]map[string]string{
		LangEnglish: EnglishTexts,
		LangTurkish: TurkishTexts,
	}
)


func Init() {
	
	locale := getSystemLocale()
	
	
	if strings.HasPrefix(locale, "tr") {
		CurrentLang = LangTurkish
	}
}


func Text(key string) string {
	if text, exists := textMap[CurrentLang][key]; exists {
		return text
	}
	
	
	if text, exists := EnglishTexts[key]; exists {
		return text
	}
	
	
	return key
}


func SetLanguage(langCode string) {
	if _, exists := textMap[langCode]; exists {
		CurrentLang = langCode
	}
}


func GetAvailableLanguages() map[string]string {
	return map[string]string{
		LangEnglish: display.English.Languages().Name(language.English),
		LangTurkish: display.English.Languages().Name(language.Turkish),
	}
}


func getSystemLocale() string {
	
	for _, envVar := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if locale := os.Getenv(envVar); locale != "" {
			
			if pos := strings.Index(locale, "."); pos > 0 {
				locale = locale[:pos]
			}
			return strings.ToLower(locale)
		}
	}
	
	
	return "en"
} 