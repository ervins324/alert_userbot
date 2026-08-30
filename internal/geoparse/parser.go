package geoparse

import (
	"regexp"
	"strings"
)

// RaionID identifies one of the 10 administrative raions of Kyiv city.
type RaionID string

const (
	RaionHolosiivskyi RaionID = "holosiivskyi"
	RaionDarnytskyi   RaionID = "darnytskyi"
	RaionDesnyanskyi  RaionID = "desnyanskyi"
	RaionDniprovskyi  RaionID = "dniprovskyi"
	RaionObolonskyi   RaionID = "obolonskyi"
	RaionPecherskyi   RaionID = "pecherskyi"
	RaionPodilskyi    RaionID = "podilskyi"
	RaionSviatoshyn   RaionID = "sviatoshynskyi"
	RaionSolomyanskyi RaionID = "solomyanskyi"
	RaionShevchenko   RaionID = "shevchenkivskyi"
)

// RaionInfo holds display names and metadata for a Kyiv district.
type RaionInfo struct {
	ID        RaionID
	NameUA    string
	NameEN    string
	CenterLat float64
	CenterLon float64
}

// AllRaions maps RaionID to RaionInfo.
var AllRaions = map[RaionID]RaionInfo{
	RaionHolosiivskyi: {ID: RaionHolosiivskyi, NameUA: "Голосіївський район", NameEN: "Holosiivskyi District", CenterLat: 50.3800, CenterLon: 30.5150},
	RaionDarnytskyi:   {ID: RaionDarnytskyi, NameUA: "Дарницький район", NameEN: "Darnytskyi District", CenterLat: 50.4100, CenterLon: 30.6600},
	RaionDesnyanskyi:  {ID: RaionDesnyanskyi, NameUA: "Деснянський район", NameEN: "Desnyanskyi District", CenterLat: 50.5100, CenterLon: 30.6000},
	RaionDniprovskyi:  {ID: RaionDniprovskyi, NameUA: "Дніпровський район", NameEN: "Dniprovskyi District", CenterLat: 50.4500, CenterLon: 30.6000},
	RaionObolonskyi:   {ID: RaionObolonskyi, NameUA: "Оболонський район", NameEN: "Obolonskyi District", CenterLat: 50.5050, CenterLon: 30.4900},
	RaionPecherskyi:   {ID: RaionPecherskyi, NameUA: "Печерський район", NameEN: "Pecherskyi District", CenterLat: 50.4300, CenterLon: 30.5500},
	RaionPodilskyi:    {ID: RaionPodilskyi, NameUA: "Подільський район", NameEN: "Podilskyi District", CenterLat: 50.4800, CenterLon: 30.4400},
	RaionSviatoshyn:   {ID: RaionSviatoshyn, NameUA: "Святошинський район", NameEN: "Sviatoshynskyi District", CenterLat: 50.4550, CenterLon: 30.3650},
	RaionSolomyanskyi: {ID: RaionSolomyanskyi, NameUA: "Солом'янський район", NameEN: "Solomyanskyi District", CenterLat: 50.4300, CenterLon: 30.4450},
	RaionShevchenko:   {ID: RaionShevchenko, NameUA: "Шевченківський район", NameEN: "Shevchenkivskyi District", CenterLat: 50.4600, CenterLon: 30.4700},
}

// LocationResult represents the parsed geographic result.
type LocationResult struct {
	MatchedRaions    []RaionID
	NeighborhoodName string
	Latitude         float64
	Longitude        float64
	HasSpecificPoint bool
	Description      string
}

type pointRule struct {
	names     []string
	raion     RaionID
	nameUA    string
	lat       float64
	lon       float64
}

// Kyiv neighborhoods, landmarks, bridges, and metro areas with their parent raion and coordinates.
var kyivPoints = []pointRule{
	// Darnytskyi
	{names: []string{"позняки", "позняках", "позняків", "позняками"}, raion: RaionDarnytskyi, nameUA: "Позняки", lat: 50.3980, lon: 30.6340},
	{names: []string{"осокорки", "осокорках", "осокорків"}, raion: RaionDarnytskyi, nameUA: "Осокорки", lat: 50.3920, lon: 30.6180},
	{names: []string{"харківський масив", "харківського масиву", "харківській", "харківська площа"}, raion: RaionDarnytskyi, nameUA: "Харківський масив", lat: 50.4070, lon: 30.6650},
	{names: []string{"дврз"}, raion: RaionDarnytskyi, nameUA: "ДВРЗ", lat: 50.4480, lon: 30.6860},
	{names: []string{"бортничі", "бортничах"}, raion: RaionDarnytskyi, nameUA: "Бортничі", lat: 50.3750, lon: 30.7000},
	{names: []string{"червоний хутір"}, raion: RaionDarnytskyi, nameUA: "Червоний хутір", lat: 50.4080, lon: 30.6920},
	{names: []string{"дарницький вокзал", "дарницька площа", "дарницької площі", "нова дарниця"}, raion: RaionDarnytskyi, nameUA: "Нова Дарниця", lat: 50.4350, lon: 30.6450},

	// Desnyanskyi
	{names: []string{"троєщина", "троєщині", "троєщину", "троєщини"}, raion: RaionDesnyanskyi, nameUA: "Троєщина", lat: 50.5180, lon: 30.6020},
	{names: []string{"лісовий масив", "лісовому масиві", "лісового масиву", "лісовий"}, raion: RaionDesnyanskyi, nameUA: "Лісовий масив", lat: 50.4780, lon: 30.6350},
	{names: []string{"биківня", "биківні"}, raion: RaionDesnyanskyi, nameUA: "Биківня", lat: 50.4720, lon: 30.6880},

	// Dniprovskyi
	{names: []string{"воскресенка", "воскресенці", "воскресенку"}, raion: RaionDniprovskyi, nameUA: "Воскресенка", lat: 50.4850, lon: 30.5900},
	{names: []string{"русанівка", "русанівці", "русанівку", "русанівські сади"}, raion: RaionDniprovskyi, nameUA: "Русанівка", lat: 50.4380, lon: 30.5970},
	{names: []string{"березняки", "березняках", "березняків"}, raion: RaionDniprovskyi, nameUA: "Березняки", lat: 50.4280, lon: 30.6010},
	{names: []string{"лівобережна", "лівобережної", "лівобережний"}, raion: RaionDniprovskyi, nameUA: "Лівобережна", lat: 50.4520, lon: 30.5980},
	{names: []string{"райдужний", "райдужному", "райдужного"}, raion: RaionDniprovskyi, nameUA: "Райдужний масив", lat: 50.4890, lon: 30.5820},
	{names: []string{"гідропарк", "гідропарку"}, raion: RaionDniprovskyi, nameUA: "Гідропарк", lat: 50.4430, lon: 30.5770},
	{names: []string{"соцмісто", "соцмісті"}, raion: RaionDniprovskyi, nameUA: "Соцмісто", lat: 50.4490, lon: 30.6200},

	// Obolonskyi
	{names: []string{"оболонь", "оболоні", "оболонська набережна"}, raion: RaionObolonskyi, nameUA: "Оболонь", lat: 50.5050, lon: 30.5010},
	{names: []string{"мінський масив", "мінському масиві", "мінського масиву", "мінська"}, raion: RaionObolonskyi, nameUA: "Мінський масив", lat: 50.5180, lon: 30.4630},
	{names: []string{"пріорка", "пріорці"}, raion: RaionObolonskyi, nameUA: "Пріорка", lat: 50.4950, lon: 30.4580},
	{names: []string{"пуща-водиця", "пущі-водиці", "пуща водиця"}, raion: RaionObolonskyi, nameUA: "Пуща-Водиця", lat: 50.5400, lon: 30.3550},
	{names: []string{"почайна", "почайні", "петрівка", "петрівці"}, raion: RaionObolonskyi, nameUA: "Почайна", lat: 50.4870, lon: 30.4980},

	// Podilskyi
	{names: []string{"поділ", "подолі", "подолу", "контрактова площа"}, raion: RaionPodilskyi, nameUA: "Поділ", lat: 50.4680, lon: 30.5170},
	{names: []string{"виноградар", "виноградарі", "виноградаря"}, raion: RaionPodilskyi, nameUA: "Виноградар", lat: 50.5080, lon: 30.4150},
	{names: []string{"вітряні гори", "вітряних горах"}, raion: RaionPodilskyi, nameUA: "Вітряні Гори", lat: 50.5000, lon: 30.4350},
	{names: []string{"куренівка", "куренівці", "куренівку"}, raion: RaionPodilskyi, nameUA: "Куренівка", lat: 50.4880, lon: 30.4700},
	{names: []string{"мостицький", "мостицькому"}, raion: RaionPodilskyi, nameUA: "Мостицький масив", lat: 50.4980, lon: 30.4370},

	// Pecherskyi
	{names: []string{"печерськ", "печерську", "печерська", "печерські пагорби"}, raion: RaionPecherskyi, nameUA: "Печерськ", lat: 50.4310, lon: 30.5450},
	{names: []string{"липки", "липках"}, raion: RaionPecherskyi, nameUA: "Липки", lat: 50.4430, lon: 30.5330},
	{names: []string{"звіринець", "звіринці"}, raion: RaionPecherskyi, nameUA: "Звіринець", lat: 50.4190, lon: 30.5550},
	{names: []string{"лаврська", "печерська лавра", "лаври", "арсенальна"}, raion: RaionPecherskyi, nameUA: "Арсенальна / Лавра", lat: 50.4420, lon: 30.5520},
	{names: []string{"дружби народів", "звіринецька"}, raion: RaionPecherskyi, nameUA: "Звіринецька площа", lat: 50.4180, lon: 30.5450},

	// Shevchenkivskyi
	{names: []string{"лук'янівка", "лукянівка", "лук'янівці", "лукянівці", "лук'янівська"}, raion: RaionShevchenko, nameUA: "Лук'янівка", lat: 50.4620, lon: 30.4820},
	{names: []string{"сирець", "сирці", "сирця"}, raion: RaionShevchenko, nameUA: "Сирець", lat: 50.4740, lon: 30.4350},
	{names: []string{"шулявка", "шулявці", "шулявку"}, raion: RaionShevchenko, nameUA: "Шулявка", lat: 50.4540, lon: 30.4450},
	{names: []string{"нивки", "нивках"}, raion: RaionShevchenko, nameUA: "Нивки", lat: 50.4610, lon: 30.4040},
	{names: []string{"хрещатик", "хрещатику", "майдан", "майдані", "центр києва", "в центрі"}, raion: RaionShevchenko, nameUA: "Центр / Хрещатик", lat: 50.4470, lon: 30.5220},
	{names: []string{"кпі", "політех"}, raion: RaionShevchenko, nameUA: "КПІ", lat: 50.4500, lon: 30.4570},
	{names: []string{"татарка", "татарці"}, raion: RaionShevchenko, nameUA: "Татарка", lat: 50.4700, lon: 30.4900},

	// Solomyanskyi
	{names: []string{"солом'янка", "соломянка", "солом'янці", "соломянці", "солом'янська площа"}, raion: RaionSolomyanskyi, nameUA: "Солом'янка", lat: 50.4310, lon: 30.4700},
	{names: []string{"відрадний", "відрадному", "відрадного"}, raion: RaionSolomyanskyi, nameUA: "Відрадний", lat: 50.4350, lon: 30.4200},
	{names: []string{"жуляни", "жулянах", "аеропорт київ", "аеропорт жуляни"}, raion: RaionSolomyanskyi, nameUA: "Жуляни", lat: 50.4020, lon: 30.4490},
	{names: []string{"чоколівка", "чоколівці", "севастопольська площа"}, raion: RaionSolomyanskyi, nameUA: "Чоколівка", lat: 50.4190, lon: 30.4570},
	{names: []string{"караваєві дачі", "кардачі"}, raion: RaionSolomyanskyi, nameUA: "Караваєві Дачі", lat: 50.4370, lon: 30.4460},
	{names: []string{"батиєва гора", "батиєвій горі"}, raion: RaionSolomyanskyi, nameUA: "Батиєва Гора", lat: 50.4300, lon: 30.4920},

	// Sviatoshynskyi
	{names: []string{"борщагівка", "борщагівці", "борщагівку", "південна борщагівка", "микільська борщагівка"}, raion: RaionSviatoshyn, nameUA: "Борщагівка", lat: 50.4150, lon: 30.3750},
	{names: []string{"святошин", "святошині", "святошино", "святошинська"}, raion: RaionSviatoshyn, nameUA: "Святошин", lat: 50.4570, lon: 30.3700},
	{names: []string{"академмістечко", "академмістечку"}, raion: RaionSviatoshyn, nameUA: "Академмістечко", lat: 50.4680, lon: 30.3550},
	{names: []string{"біличі", "біличах", "новобіличі"}, raion: RaionSviatoshyn, nameUA: "Біличі", lat: 50.4630, lon: 30.3400},

	// Holosiivskyi
	{names: []string{"голосієво", "голосіїв", "голосіївський парк"}, raion: RaionHolosiivskyi, nameUA: "Голосієво", lat: 50.3920, lon: 30.5050},
	{names: []string{"теремки", "теремках", "теремків", "теремки-1", "теремки-2", "теремки 1", "теремки 2"}, raion: RaionHolosiivskyi, nameUA: "Теремки", lat: 50.3660, lon: 30.4550},
	{names: []string{"корчувате", "корчуватому"}, raion: RaionHolosiivskyi, nameUA: "Корчувате", lat: 50.3670, lon: 30.5600},
	{names: []string{"деміївка", "деміївці", "деміївська", "автовокзал"}, raion: RaionHolosiivskyi, nameUA: "Деміївка", lat: 50.4050, lon: 30.5180},
	{names: []string{"пирогів", "пирогово", "пирогові"}, raion: RaionHolosiivskyi, nameUA: "Пирогів", lat: 50.3520, lon: 30.5100},
	{names: []string{"феофанія", "феофанії"}, raion: RaionHolosiivskyi, nameUA: "Феофанія", lat: 50.3420, lon: 30.4850},
	{names: []string{"вднг", "виставковий центр"}, raion: RaionHolosiivskyi, nameUA: "ВДНГ", lat: 50.3780, lon: 30.4780},
	{names: []string{"либідська", "либідської"}, raion: RaionHolosiivskyi, nameUA: "Либідська", lat: 50.4130, lon: 30.5240},

	// Bridges
	{names: []string{"південний міст", "південному мосту", "південного мосту"}, raion: RaionDarnytskyi, nameUA: "Південний міст", lat: 50.3980, lon: 30.5880},
	{names: []string{"північний міст", "північному мосту", "північного мосту", "московський міст"}, raion: RaionDesnyanskyi, nameUA: "Північний міст", lat: 50.4910, lon: 30.5380},
	{names: []string{"міст патона", "мосту патона", "мостом патона"}, raion: RaionPecherskyi, nameUA: "Міст Патона", lat: 50.4280, lon: 30.5750},
	{names: []string{"міст метро", "мосту метро"}, raion: RaionDniprovskyi, nameUA: "Міст Метро", lat: 50.4430, lon: 30.5690},
	{names: []string{"дарницький міст", "дарницькому мосту"}, raion: RaionDarnytskyi, nameUA: "Дарницький міст", lat: 50.4160, lon: 30.5890},
	{names: []string{"подільсько-воскресенський", "подільський міст"}, raion: RaionPodilskyi, nameUA: "Подільський міст", lat: 50.4720, lon: 30.5350},
}

// Raion keyword stems for matching Ukrainian inflections
var raionStems = []struct {
	id    RaionID
	stems []string
}{
	{RaionDarnytskyi, []string{"дарниц"}},
	{RaionDesnyanskyi, []string{"деснянськ"}},
	{RaionDniprovskyi, []string{"дніпровськ"}},
	{RaionObolonskyi, []string{"оболон"}},
	{RaionPecherskyi, []string{"печерськ", "печерськ"}},
	{RaionPodilskyi, []string{"подільськ", "поділ"}},
	{RaionSviatoshyn, []string{"святошин"}},
	{RaionSolomyanskyi, []string{"солом'ян", "соломян"}},
	{RaionShevchenko, []string{"шевченківськ"}},
	{RaionHolosiivskyi, []string{"голосіїв", "голосієв", "голосіївськ"}},
}

// ExtractLocation parses text and finds any mentioned Kyiv raions, neighborhoods, bridges or landmarks.
func ExtractLocation(text string) *LocationResult {
	norm := strings.ToLower(text)
	var matchedRaions []RaionID
	raionSet := make(map[RaionID]bool)

	var specificPoint *pointRule

	// 1. Check specific points and landmarks first
	for _, p := range kyivPoints {
		for _, alias := range p.names {
			if strings.Contains(norm, alias) {
				specificPoint = &p
				raionSet[p.raion] = true
				break
			}
		}
		if specificPoint != nil {
			break
		}
	}

	// 2. Check raion direct names
	for _, rs := range raionStems {
		for _, stem := range rs.stems {
			if strings.Contains(norm, stem) {
				raionSet[rs.id] = true
				break
			}
		}
	}

	// 3. Check Left / Right bank mentions if no explicit raion
	if strings.Contains(norm, "лівий берег") || strings.Contains(norm, "лівому березі") || strings.Contains(norm, "лівобережж") {
		raionSet[RaionDarnytskyi] = true
		raionSet[RaionDniprovskyi] = true
		raionSet[RaionDesnyanskyi] = true
	} else if strings.Contains(norm, "правий берег") || strings.Contains(norm, "правому березі") || strings.Contains(norm, "правобережж") {
		raionSet[RaionObolonskyi] = true
		raionSet[RaionPodilskyi] = true
		raionSet[RaionShevchenko] = true
		raionSet[RaionPecherskyi] = true
		raionSet[RaionSolomyanskyi] = true
		raionSet[RaionSviatoshyn] = true
		raionSet[RaionHolosiivskyi] = true
	}

	for id := range raionSet {
		matchedRaions = append(matchedRaions, id)
	}

	if len(matchedRaions) == 0 && specificPoint == nil {
		// Generic Kyiv check
		if strings.Contains(norm, "київ") || strings.Contains(norm, "києві") || strings.Contains(norm, "києва") {
			return &LocationResult{
				Description: "м. Київ (загально)",
			}
		}
		return nil
	}

	res := &LocationResult{
		MatchedRaions: matchedRaions,
	}

	if specificPoint != nil {
		res.NeighborhoodName = specificPoint.nameUA
		res.Latitude = specificPoint.lat
		res.Longitude = specificPoint.lon
		res.HasSpecificPoint = true
		raionInfo := AllRaions[specificPoint.raion]
		res.Description = raionInfo.NameUA + " (" + specificPoint.nameUA + ")"
	} else if len(matchedRaions) == 1 {
		raionInfo := AllRaions[matchedRaions[0]]
		res.Latitude = raionInfo.CenterLat
		res.Longitude = raionInfo.CenterLon
		res.Description = raionInfo.NameUA
	} else if len(matchedRaions) > 1 {
		var names []string
		for _, id := range matchedRaions {
			names = append(names, AllRaions[id].NameUA)
		}
		res.Description = strings.Join(names, ", ")
	}

	return res
}
