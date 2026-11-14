package block

import "strings"

var titleTranslations = map[string]map[string]string{
	"ru": {
		"Best sellers":                 "Лидеры продаж",
		"Deals worth discovering":      "Выгодные предложения",
		"New releases":                 "Новые релизы",
		"Explore the marketplace":      "Откройте мир игр",
		"Discover your next adventure": "Твоя следующая игровая вселенная",
		"Explore this week's deals":    "Большие игры. Маленькие цены.",
		"Favorites from our players":   "Игры, которые остаются с тобой",
	},
	"lt": {
		"Best sellers":                 "Perkamiausi žaidimai",
		"Deals worth discovering":      "Verti dėmesio pasiūlymai",
		"New releases":                 "Naujausi leidimai",
		"Explore the marketplace":      "Atraskite žaidimų pasaulį",
		"Discover your next adventure": "Atraskite naują nuotykį",
		"Explore this week's deals":    "Atraskite šios savaitės pasiūlymus",
		"Favorites from our players":   "Žaidėjų mėgstamiausi",
	},
}

func LocalizedTitle(title, acceptLanguage string) string {
	language := strings.ToLower(strings.TrimSpace(strings.SplitN(acceptLanguage, ",", 2)[0]))
	language = strings.SplitN(strings.SplitN(language, ";", 2)[0], "-", 2)[0]
	if translated, ok := titleTranslations[language][title]; ok {
		return translated
	}
	return title
}
