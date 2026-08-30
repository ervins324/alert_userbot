package geoparse

import (
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
	RaionHolosiivskyi: {ID: RaionHolosiivskyi, NameUA: "Голосіївський район", NameEN: "Holosiivskyi District", CenterLat: 50.3750, CenterLon: 30.5050},
	RaionDarnytskyi:   {ID: RaionDarnytskyi, NameUA: "Дарницький район", NameEN: "Darnytskyi District", CenterLat: 50.4050, CenterLon: 30.6600},
	RaionDesnyanskyi:  {ID: RaionDesnyanskyi, NameUA: "Деснянський район", NameEN: "Desnyanskyi District", CenterLat: 50.5150, CenterLon: 30.6150},
	RaionDniprovskyi:  {ID: RaionDniprovskyi, NameUA: "Дніпровський район", NameEN: "Dniprovskyi District", CenterLat: 50.4500, CenterLon: 30.6000},
	RaionObolonskyi:   {ID: RaionObolonskyi, NameUA: "Оболонський район", NameEN: "Obolonskyi District", CenterLat: 50.5100, CenterLon: 30.4850},
	RaionPecherskyi:   {ID: RaionPecherskyi, NameUA: "Печерський район", NameEN: "Pecherskyi District", CenterLat: 50.4300, CenterLon: 30.5450},
	RaionPodilskyi:    {ID: RaionPodilskyi, NameUA: "Подільський район", NameEN: "Podilskyi District", CenterLat: 50.4850, CenterLon: 30.4350},
	RaionSviatoshyn:   {ID: RaionSviatoshyn, NameUA: "Святошинський район", NameEN: "Sviatoshynskyi District", CenterLat: 50.4550, CenterLon: 30.3600},
	RaionSolomyanskyi: {ID: RaionSolomyanskyi, NameUA: "Солом'янський район", NameEN: "Solomyanskyi District", CenterLat: 50.4250, CenterLon: 30.4450},
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
	names  []string
	raion  RaionID
	nameUA string
	lat    float64
	lon    float64
}

// Kyiv points of interest, energy facilities, shopping malls, transport hubs, bridges & neighborhoods
var kyivPoints = []pointRule{
	// Energy Facilities (Critical targets)
	{names: []string{"тец-5", "тец 5", "тец5"}, raion: RaionPecherskyi, nameUA: "ТЕЦ-5", lat: 50.3950, lon: 30.5650},
	{names: []string{"тец-6", "тец 6", "тец6"}, raion: RaionDesnyanskyi, nameUA: "ТЕЦ-6", lat: 50.5310, lon: 30.6540},

	// Shopping Malls & Commercial Centers
	{names: []string{"рівер мол", "рівермол", "river mall"}, raion: RaionDarnytskyi, nameUA: "ТРЦ River Mall", lat: 50.4000, lon: 30.6180},
	{names: []string{"трц республіка", "республіка", "республіки", "республіку"}, raion: RaionHolosiivskyi, nameUA: "ТРЦ Республіка", lat: 50.3680, lon: 30.4550},
	{names: []string{"трц лавіна", "лавіна", "лавіну", "лавіни", "lavina"}, raion: RaionSviatoshyn, nameUA: "ТРЦ Лавіна", lat: 50.4950, lon: 30.3580},
	{names: []string{"трц ретровіль", "ретровіль", "ретровиль", "retroville"}, raion: RaionPodilskyi, nameUA: "ТРЦ Retroville", lat: 50.5050, lon: 30.4180},
	{names: []string{"трц проспект", "тц проспект"}, raion: RaionDniprovskyi, nameUA: "ТРЦ Проспект", lat: 50.4550, lon: 30.6340},
	{names: []string{"трц блокбастер", "блокбастер", "blockbuster"}, raion: RaionObolonskyi, nameUA: "ТРЦ Блокбастер", lat: 50.4890, lon: 30.5280},
	{names: []string{"епіцентр"}, raion: RaionHolosiivskyi, nameUA: "Епіцентр (Кільцева)", lat: 50.3720, lon: 30.4500},

	// University / Hubs / Logistics
	{names: []string{"гуртожитків імені тараса шевченка", "гуртожитки шевченка", "гуртожитки кну"}, raion: RaionHolosiivskyi, nameUA: "Гуртожитки КНУ", lat: 50.3840, lon: 30.4920},
	{names: []string{"термінал нової пошти", "нової пошти"}, raion: RaionSolomyanskyi, nameUA: "Термінал Нової Пошти", lat: 50.3800, lon: 30.4400},
	{names: []string{"вокзал", "вокзалу", "вокзалі", "центральний вокзал", "залізничний вокзал"}, raion: RaionSolomyanskyi, nameUA: "Центральний вокзал", lat: 50.4400, lon: 30.4890},
	{names: []string{"дарницький вокзал"}, raion: RaionDarnytskyi, nameUA: "Дарницький вокзал", lat: 50.4350, lon: 30.6450},
	{names: []string{"аеропорт бориспіль", "бориспільський аеропорт"}, raion: RaionDarnytskyi, nameUA: "Аеропорт Бориспіль", lat: 50.3450, lon: 30.8950},
	{names: []string{"жуляни", "жулянах", "аеропорт київ", "аеропорт жуляни"}, raion: RaionSolomyanskyi, nameUA: "Аеропорт Жуляни", lat: 50.4020, lon: 30.4490},
	{names: []string{"київське водосховище", "водосховище", "водосховищі", "київське море"}, raion: RaionObolonskyi, nameUA: "Київське водосховище", lat: 50.5850, lon: 30.5100},
	{names: []string{"труханів острів", "труханів", "трухановому"}, raion: RaionDniprovskyi, nameUA: "Труханів острів", lat: 50.4600, lon: 30.5400},
	{names: []string{"святошинський ліс"}, raion: RaionSviatoshyn, nameUA: "Святошинський ліс", lat: 50.5100, lon: 30.3300},
	{names: []string{"житомирськ", "стоянка", "стоянці"}, raion: RaionSviatoshyn, nameUA: "Стоянка / Житомирська траса", lat: 50.4480, lon: 30.2300},
	{names: []string{"бориспільська траса"}, raion: RaionDarnytskyi, nameUA: "Бориспільська траса", lat: 50.3950, lon: 30.7500},

	// Darnytskyi
	{names: []string{"позняки", "позняках", "позняків", "позняками"}, raion: RaionDarnytskyi, nameUA: "Позняки", lat: 50.3980, lon: 30.6340},
	{names: []string{"осокорки", "осокорках", "осокорків"}, raion: RaionDarnytskyi, nameUA: "Осокорки", lat: 50.3920, lon: 30.6180},
	{names: []string{"нижні сади", "нижніх садах"}, raion: RaionDarnytskyi, nameUA: "Нижні Сади", lat: 50.3550, lon: 30.6050},
	{names: []string{"харківський масив", "харківського масиву", "харківській", "харківська площа"}, raion: RaionDarnytskyi, nameUA: "Харківський масив", lat: 50.4070, lon: 30.6650},
	{names: []string{"рембаза", "рембази", "рембазу", "рембазі"}, raion: RaionDarnytskyi, nameUA: "Рембаза", lat: 50.4250, lon: 30.6900},
	{names: []string{"дарницький масив", "дарницького масиву", "нова дарниця", "дарницька площа"}, raion: RaionDarnytskyi, nameUA: "Дарницький масив", lat: 50.4350, lon: 30.6450},
	{names: []string{"дврз"}, raion: RaionDarnytskyi, nameUA: "ДВРЗ", lat: 50.4480, lon: 30.6860},
	{names: []string{"бортничі", "бортничах"}, raion: RaionDarnytskyi, nameUA: "Бортничі", lat: 50.3750, lon: 30.7000},
	{names: []string{"червоний хутір"}, raion: RaionDarnytskyi, nameUA: "Червоний хутір", lat: 50.4080, lon: 30.6920},

	// Desnyanskyi
	{names: []string{"троєщина", "троєщині", "троєщину", "троєщини", "трою"}, raion: RaionDesnyanskyi, nameUA: "Троєщина", lat: 50.5180, lon: 30.6020},
	{names: []string{"лісовий масив", "лісовому масиві", "лісового масиву", "лісовий"}, raion: RaionDesnyanskyi, nameUA: "Лісовий масив", lat: 50.4780, lon: 30.6350},
	{names: []string{"биківня", "биківні"}, raion: RaionDesnyanskyi, nameUA: "Биківня", lat: 50.4720, lon: 30.6880},
	{names: []string{"погреби", "погребах"}, raion: RaionDesnyanskyi, nameUA: "Погреби", lat: 50.5500, lon: 30.6380},

	// Dniprovskyi
	{names: []string{"воскресенка", "воскресенці", "воскресенку"}, raion: RaionDniprovskyi, nameUA: "Воскресенка", lat: 50.4850, lon: 30.5900},
	{names: []string{"русанівка", "русанівці", "русанівку"}, raion: RaionDniprovskyi, nameUA: "Русанівка", lat: 50.4380, lon: 30.5970},
	{names: []string{"русанівські сади", "русанівських садах"}, raion: RaionDniprovskyi, nameUA: "Русанівські Сади", lat: 50.4650, lon: 30.5750},
	{names: []string{"березняки", "березняках", "березняків"}, raion: RaionDniprovskyi, nameUA: "Березняки", lat: 50.4280, lon: 30.6010},
	{names: []string{"лівобережний масив", "лівобережний", "лівобережна", "лівобережної"}, raion: RaionDniprovskyi, nameUA: "Лівобережний масив", lat: 50.4520, lon: 30.5980},
	{names: []string{"райдужний", "райдужному", "райдужного"}, raion: RaionDniprovskyi, nameUA: "Райдужний масив", lat: 50.4890, lon: 30.5820},
	{names: []string{"гідропарк", "гідропарку"}, raion: RaionDniprovskyi, nameUA: "Гідропарк", lat: 50.4430, lon: 30.5770},
	{names: []string{"соцмісто", "соцмісті"}, raion: RaionDniprovskyi, nameUA: "Соцмісто", lat: 50.4490, lon: 30.6200},

	// Obolonskyi
	{names: []string{"оболонь", "оболоні", "оболонська набережна"}, raion: RaionObolonskyi, nameUA: "Оболонь", lat: 50.5050, lon: 30.5010},
	{names: []string{"мінський масив", "мінському масиві", "мінського масиву", "мінська"}, raion: RaionObolonskyi, nameUA: "Мінський масив", lat: 50.5180, lon: 30.4630},
	{names: []string{"пріорка", "пріорці"}, raion: RaionObolonskyi, nameUA: "Пріорка", lat: 50.4950, lon: 30.4580},
	{names: []string{"пуща-водиця", "пущі-водиці", "пуща водиця"}, raion: RaionObolonskyi, nameUA: "Пуща-Водиця", lat: 50.5400, lon: 30.3550},
	{names: []string{"почайна", "почайні", "почайну", "петрівка", "петрівці"}, raion: RaionObolonskyi, nameUA: "Почайна", lat: 50.4870, lon: 30.4980},

	// Podilskyi
	{names: []string{"поділ", "подолі", "подолу", "контрактова площа"}, raion: RaionPodilskyi, nameUA: "Поділ", lat: 50.4680, lon: 30.5170},
	{names: []string{"виноградар", "виноградарі", "виноградаря"}, raion: RaionPodilskyi, nameUA: "Виноградар", lat: 50.5080, lon: 30.4150},
	{names: []string{"вітряні гори", "вітряних горах"}, raion: RaionPodilskyi, nameUA: "Вітряні Гори", lat: 50.5000, lon: 30.4350},
	{names: []string{"куренівка", "куренівці", "куренівку"}, raion: RaionPodilskyi, nameUA: "Куренівка", lat: 50.4880, lon: 30.4700},
	{names: []string{"берковець", "берківці"}, raion: RaionPodilskyi, nameUA: "Берковець", lat: 50.4950, lon: 30.3580},
	{names: []string{"мостицький", "мостицькому"}, raion: RaionPodilskyi, nameUA: "Мостицький масив", lat: 50.4980, lon: 30.4370},

	// Pecherskyi
	{names: []string{"печерськ", "печерську", "печерська", "печерські пагорби"}, raion: RaionPecherskyi, nameUA: "Печерськ", lat: 50.4310, lon: 30.5450},
	{names: []string{"липки", "липках"}, raion: RaionPecherskyi, nameUA: "Липки", lat: 50.4430, lon: 30.5330},
	{names: []string{"клов", "кловський узвіз", "кловській"}, raion: RaionPecherskyi, nameUA: "Клов", lat: 50.4380, lon: 30.5350},
	{names: []string{"звіринець", "звіринці"}, raion: RaionPecherskyi, nameUA: "Звіринець", lat: 50.4190, lon: 30.5550},
	{names: []string{"лаврська", "печерська лавра", "лаври", "арсенальна"}, raion: RaionPecherskyi, nameUA: "Арсенальна / Лавра", lat: 50.4420, lon: 30.5520},
	{names: []string{"видубичі", "видубичах"}, raion: RaionPecherskyi, nameUA: "Видубичі", lat: 50.4030, lon: 30.5600},

	// Shevchenkivskyi
	{names: []string{"лук'янівка", "лукянівка", "лук'янівці", "лукянівці", "лук'янівку", "лукянівку", "лук'янівська"}, raion: RaionShevchenko, nameUA: "Лук'янівка", lat: 50.4620, lon: 30.4820},
	{names: []string{"сирець", "сирці", "сирця"}, raion: RaionShevchenko, nameUA: "Сирець", lat: 50.4740, lon: 30.4350},
	{names: []string{"шулявка", "шулявці", "шулявку"}, raion: RaionShevchenko, nameUA: "Шулявка", lat: 50.4540, lon: 30.4450},
	{names: []string{"нивки", "нивках", "неявки"}, raion: RaionShevchenko, nameUA: "Нивки", lat: 50.4610, lon: 30.4040},
	{names: []string{"хрещатик", "хрещатику", "майдан", "майдані", "центр києва", "в центрі", "центр"}, raion: RaionShevchenko, nameUA: "Центр", lat: 50.4470, lon: 30.5220},
	{names: []string{"кпі", "політех"}, raion: RaionShevchenko, nameUA: "КПІ", lat: 50.4500, lon: 30.4570},
	{names: []string{"татарка", "татарці"}, raion: RaionShevchenko, nameUA: "Татарка", lat: 50.4700, lon: 30.4900},

	// Solomyanskyi
	{names: []string{"солом'янка", "соломянка", "солом'янці", "соломянці", "солом'янку", "солома"}, raion: RaionSolomyanskyi, nameUA: "Солом'янка", lat: 50.4310, lon: 30.4700},
	{names: []string{"відрадний", "відрадному", "відрадного"}, raion: RaionSolomyanskyi, nameUA: "Відрадний", lat: 50.4350, lon: 30.4200},
	{names: []string{"чоколівка", "чоколівці", "севастопольська площа"}, raion: RaionSolomyanskyi, nameUA: "Чоколівка", lat: 50.4190, lon: 30.4570},
	{names: []string{"совки", "совках"}, raion: RaionSolomyanskyi, nameUA: "Совки", lat: 50.4050, lon: 30.4880},
	{names: []string{"караваєві дачі", "кардачі"}, raion: RaionSolomyanskyi, nameUA: "Караваєві Дачі", lat: 50.4370, lon: 30.4460},
	{names: []string{"батиєва гора", "батиєвій горі"}, raion: RaionSolomyanskyi, nameUA: "Батиєва Гора", lat: 50.4300, lon: 30.4920},

	// Sviatoshynskyi
	{names: []string{"борщагівка", "борщагівці", "борщагівку", "борщагівки", "південна борщагівка", "микільська борщагівка"}, raion: RaionSviatoshyn, nameUA: "Борщагівка", lat: 50.4150, lon: 30.3750},
	{names: []string{"петропавлівська борщагівка", "петропавлівську борщагівку"}, raion: RaionSviatoshyn, nameUA: "Петропавлівська Борщагівка", lat: 50.4350, lon: 30.3250},
	{names: []string{"софіївська борщагівка", "соф борщагівк"}, raion: RaionSviatoshyn, nameUA: "Софіївська Борщагівка", lat: 50.4050, lon: 30.3350},
	{names: []string{"святошин", "святошині", "святошино", "святошинська"}, raion: RaionSviatoshyn, nameUA: "Святошин", lat: 50.4570, lon: 30.3700},
	{names: []string{"академмістечко", "академмістечку"}, raion: RaionSviatoshyn, nameUA: "Академмістечко", lat: 50.4680, lon: 30.3550},
	{names: []string{"біличі", "біличах", "новобіличі"}, raion: RaionSviatoshyn, nameUA: "Біличі", lat: 50.4630, lon: 30.3400},

	// Holosiivskyi
	{names: []string{"голосієво", "голосіїв", "голосіївський парк"}, raion: RaionHolosiivskyi, nameUA: "Голосієво", lat: 50.3920, lon: 30.5050},
	{names: []string{"теремки", "теремках", "теремків", "теремки-1", "теремки-2", "теремки 1", "теремки 2"}, raion: RaionHolosiivskyi, nameUA: "Теремки", lat: 50.3660, lon: 30.4550},
	{names: []string{"корчувате", "корчуватому"}, raion: RaionHolosiivskyi, nameUA: "Корчувате", lat: 50.3670, lon: 30.5600},
	{names: []string{"деміївка", "деміївці", "деміївська", "автовокзал"}, raion: RaionHolosiivskyi, nameUA: "Деміївка", lat: 50.4050, lon: 30.5180},
	{names: []string{"пирогів", "пирогово", "пирогові"}, raion: RaionHolosiivskyi, nameUA: "Пирогів", lat: 50.3520, lon: 30.5100},
	{names: []string{"феофанія", "феофанії", "феофанію"}, raion: RaionHolosiivskyi, nameUA: "Феофанія", lat: 50.3420, lon: 30.4850},
	{names: []string{"вднг", "виставковий центр"}, raion: RaionHolosiivskyi, nameUA: "ВДНГ", lat: 50.3780, lon: 30.4780},
	{names: []string{"либідська", "либідської"}, raion: RaionHolosiivskyi, nameUA: "Либідська", lat: 50.4130, lon: 30.5240},

	// Bridges
	{names: []string{"південний міст", "південному мосту", "південного мосту"}, raion: RaionDarnytskyi, nameUA: "Південний міст", lat: 50.3980, lon: 30.5880},
	{names: []string{"північний міст", "північному мосту", "північного мосту", "московський міст"}, raion: RaionDesnyanskyi, nameUA: "Північний міст", lat: 50.4910, lon: 30.5380},
	{names: []string{"подільський міст", "подільському мосту", "подільсько-воскресенський"}, raion: RaionPodilskyi, nameUA: "Подільський міст", lat: 50.4720, lon: 30.5350},
	{names: []string{"міст патона", "мосту патона", "мостом патона"}, raion: RaionPecherskyi, nameUA: "Міст Патона", lat: 50.4280, lon: 30.5750},
	{names: []string{"міст метро", "мосту метро"}, raion: RaionDniprovskyi, nameUA: "Міст Метро", lat: 50.4430, lon: 30.5690},
	{names: []string{"дарницький міст", "дарницькому мосту"}, raion: RaionDarnytskyi, nameUA: "Дарницький міст", lat: 50.4160, lon: 30.5890},

	// Agglomeration / Near Kyiv Suburbs
	{names: []string{"бровари", "броварів", "броварах", "броварський"}, raion: RaionDesnyanskyi, nameUA: "Бровари", lat: 50.5110, lon: 30.7900},
	{names: []string{"бориспіль", "борисполя", "бориспільський"}, raion: RaionDarnytskyi, nameUA: "Бориспіль", lat: 50.3520, lon: 30.9550},
	{names: []string{"вишневе", "вишневого", "вишневому"}, raion: RaionSviatoshyn, nameUA: "Вишневе", lat: 50.3880, lon: 30.3580},
	{names: []string{"боярка", "боярки", "боярці", "боярку"}, raion: RaionSviatoshyn, nameUA: "Боярка", lat: 50.3270, lon: 30.2950},
	{names: []string{"васильків", "василькова", "василькові"}, raion: RaionHolosiivskyi, nameUA: "Васильків", lat: 50.1780, lon: 30.3150},
	{names: []string{"глеваха", "глевахи", "глеваху"}, raion: RaionHolosiivskyi, nameUA: "Глеваха", lat: 50.2650, lon: 30.3180},
	{names: []string{"калинівка", "калинівки", "калинівку"}, raion: RaionHolosiivskyi, nameUA: "Калинівка", lat: 50.2280, lon: 30.2350},
	{names: []string{"чабани", "чабанах"}, raion: RaionHolosiivskyi, nameUA: "Чабани", lat: 50.3420, lon: 30.4220},
	{names: []string{"гатне", "гатного"}, raion: RaionHolosiivskyi, nameUA: "Гатне", lat: 50.3600, lon: 30.4000},
	{names: []string{"хотів", "хотова"}, raion: RaionHolosiivskyi, nameUA: "Хотів", lat: 50.3350, lon: 30.4680},
	{names: []string{"ірпінь", "ірпеня", "ірпені"}, raion: RaionSviatoshyn, nameUA: "Ірпінь", lat: 50.5200, lon: 30.2450},
	{names: []string{"буча", "бучі"}, raion: RaionSviatoshyn, nameUA: "Буча", lat: 50.5480, lon: 30.2200},
	{names: []string{"гостомель", "гостомеля"}, raion: RaionSviatoshyn, nameUA: "Гостомель", lat: 50.5680, lon: 30.2650},
	{names: []string{"вишгород", "вишгорода", "вишгородський"}, raion: RaionObolonskyi, nameUA: "Вишгород", lat: 50.5830, lon: 30.4900},
	{names: []string{"козин", "козина"}, raion: RaionHolosiivskyi, nameUA: "Козин", lat: 50.2180, lon: 30.6700},
	{names: []string{"вишеньки", "вишеньок"}, raion: RaionDarnytskyi, nameUA: "Вишеньки", lat: 50.3000, lon: 30.6900},
	{names: []string{"гнідин", "гнідина"}, raion: RaionDarnytskyi, nameUA: "Гнідин", lat: 50.3320, lon: 30.7150},
	{names: []string{"велика димерка", "велику димерку"}, raion: RaionDesnyanskyi, nameUA: "Велика Димерка", lat: 50.5900, lon: 30.9100},
	{names: []string{"віта-поштова", "віта поштова", "ввта-поштова"}, raion: RaionHolosiivskyi, nameUA: "Віта-Поштова", lat: 50.3180, lon: 30.3850},
	{names: []string{"чубинське", "велика олександрівка"}, raion: RaionDarnytskyi, nameUA: "Чубинське", lat: 50.3800, lon: 30.8500},
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
	{RaionPecherskyi, []string{"печерськ"}},
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
