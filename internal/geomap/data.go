package geomap

import "alert-userbot/internal/geoparse"

// Coord represents a (latitude, longitude) coordinate.
type Coord struct {
	Lat float64
	Lon float64
}

// Kyiv bounding box (covers Kyiv city and immediate agglomeration)
const (
	MinLat = 50.250
	MaxLat = 50.600
	MinLon = 30.200
	MaxLon = 30.850
)

// RaionPolygon holds the boundary coordinates and display metadata for a Kyiv district.
type RaionPolygon struct {
	ID       geoparse.RaionID
	NameUA   string
	Center   Coord
	Boundary []Coord
}

// KyivRaionBoundaries holds boundary polygons for all 10 Kyiv administrative raions derived from OpenStreetMap.
var KyivRaionBoundaries = map[geoparse.RaionID]RaionPolygon{
	geoparse.RaionObolonskyi: {
		ID:     geoparse.RaionObolonskyi,
		NameUA: "Оболонський",
		Center: Coord{Lat: 50.510, Lon: 30.485},
		Boundary: []Coord{
			{Lat: 50.540, Lon: 30.340}, // Pushcha-Vodytsia North-West
			{Lat: 50.560, Lon: 30.355},
			{Lat: 50.585, Lon: 30.490}, // Kyiv border north of Minskyi / Vyshhorod road
			{Lat: 50.588, Lon: 30.515}, // Dnipro entry / Vyshhorod dam border
			{Lat: 50.545, Lon: 30.535}, // Dnipro along Obolon
			{Lat: 50.510, Lon: 30.530}, // Obolonska Naberezhna
			{Lat: 50.490, Lon: 30.530}, // Pivnichnyi bridge / Rybalskyi border
			{Lat: 50.485, Lon: 30.490}, // Pochayna / Bandery Ave
			{Lat: 50.480, Lon: 30.450}, // Kurenivka / Syretska St
			{Lat: 50.495, Lon: 30.420}, // Vynohradar / Pravdy Ave border
			{Lat: 50.510, Lon: 30.360}, // Berkovets border
			{Lat: 50.540, Lon: 30.340},
		},
	},

	geoparse.RaionDesnyanskyi: {
		ID:     geoparse.RaionDesnyanskyi,
		NameUA: "Деснянський",
		Center: Coord{Lat: 50.515, Lon: 30.615},
		Boundary: []Coord{
			{Lat: 50.588, Lon: 30.515}, // North-West border along Dnipro
			{Lat: 50.575, Lon: 30.600}, // Troieshchyna North border (Pohreby border)
			{Lat: 50.565, Lon: 30.660}, // TEC-6 East / Knyazhychi road
			{Lat: 50.530, Lon: 30.700}, // Bykivnya North-East
			{Lat: 50.470, Lon: 30.715}, // Brovary highway / city border
			{Lat: 50.458, Lon: 30.640}, // Brovarskyi Ave / Lisova metro border
			{Lat: 50.465, Lon: 30.600}, // Chernihivska / Bratyslavska St border
			{Lat: 50.480, Lon: 30.580}, // Raiduzhnyi / Shukhevycha Ave border
			{Lat: 50.490, Lon: 30.538}, // Pivnichnyi Bridge
			{Lat: 50.545, Lon: 30.535}, // Dnipro along Muromets
			{Lat: 50.588, Lon: 30.515},
		},
	},

	geoparse.RaionDniprovskyi: {
		ID:     geoparse.RaionDniprovskyi,
		NameUA: "Дніпровський",
		Center: Coord{Lat: 50.450, Lon: 30.600},
		Boundary: []Coord{
			{Lat: 50.490, Lon: 30.538}, // Pivnichnyi Bridge
			{Lat: 50.480, Lon: 30.580}, // Shukhevycha / Raiduzhnyi border
			{Lat: 50.465, Lon: 30.600}, // Bratyslavska St / Lisovyi border
			{Lat: 50.458, Lon: 30.640}, // Brovarskyi Ave / Chernihivska border
			{Lat: 50.445, Lon: 30.665}, // DVRZ / Alma-Atynska border
			{Lat: 50.435, Lon: 30.630}, // Darnytska Square / Sobornosti Ave
			{Lat: 50.420, Lon: 30.600}, // Berezniaky / Prydniprovska border
			{Lat: 50.428, Lon: 30.575}, // Paton bridge / Dnipro
			{Lat: 50.443, Lon: 30.569}, // Metro bridge / Hydropark
			{Lat: 50.470, Lon: 30.530}, // Trukhaniv Island / Dnipro
			{Lat: 50.490, Lon: 30.538},
		},
	},

	geoparse.RaionDarnytskyi: {
		ID:     geoparse.RaionDarnytskyi,
		NameUA: "Дарницький",
		Center: Coord{Lat: 50.405, Lon: 30.660},
		Boundary: []Coord{
			{Lat: 50.435, Lon: 30.630}, // Darnytska Square
			{Lat: 50.445, Lon: 30.665}, // DVRZ / Railway border
			{Lat: 50.440, Lon: 30.735}, // Rembaza / East forest border
			{Lat: 50.365, Lon: 30.760}, // Bortnychi South-East / City border
			{Lat: 50.335, Lon: 30.660}, // Nyzhni Sady / Osokorky Dachas border
			{Lat: 50.380, Lon: 30.600}, // Dnipro along Osokorky / River Mall
			{Lat: 50.398, Lon: 30.588}, // Pivdennyi bridge
			{Lat: 50.416, Lon: 30.589}, // Darnytskyi bridge
			{Lat: 50.420, Lon: 30.600}, // Berezniaky / Sobornosti border
			{Lat: 50.435, Lon: 30.630},
		},
	},

	geoparse.RaionPecherskyi: {
		ID:     geoparse.RaionPecherskyi,
		NameUA: "Печерський",
		Center: Coord{Lat: 50.430, Lon: 30.545},
		Boundary: []Coord{
			{Lat: 50.455, Lon: 30.530}, // Khreshchatyk / European Square
			{Lat: 50.443, Lon: 30.569}, // Dnipro along Parkova road / Metro bridge
			{Lat: 50.428, Lon: 30.575}, // Paton bridge / Naberezhne highway
			{Lat: 50.405, Lon: 30.565}, // Vydubychi / TEC-5 / Pivdennyi bridge
			{Lat: 50.408, Lon: 30.525}, // Lybidska square / Druzhby Narodiv border
			{Lat: 50.425, Lon: 30.515}, // Velyka Vasylkivska St
			{Lat: 50.440, Lon: 30.520}, // Bessarabska square
			{Lat: 50.455, Lon: 30.530},
		},
	},

	geoparse.RaionPodilskyi: {
		ID:     geoparse.RaionPodilskyi,
		NameUA: "Подільський",
		Center: Coord{Lat: 50.485, Lon: 30.435},
		Boundary: []Coord{
			{Lat: 50.495, Lon: 30.420}, // Vynohradar / Retroville
			{Lat: 50.480, Lon: 30.450}, // Kurenivka / Priorka
			{Lat: 50.485, Lon: 30.490}, // Pochayna border
			{Lat: 50.485, Lon: 30.530}, // Havanskyi bridge / Rybalskyi peninsula
			{Lat: 50.455, Lon: 30.530}, // Poshtova Square
			{Lat: 50.460, Lon: 30.485}, // Tatarka border / Nyzhniy Val
			{Lat: 50.470, Lon: 30.420}, // Syrets border / Kotelnikova
			{Lat: 50.495, Lon: 30.370}, // Berkovets / Lavina border
			{Lat: 50.520, Lon: 30.350}, // Pushcha-Vodytsia south border
			{Lat: 50.495, Lon: 30.420},
		},
	},

	geoparse.RaionShevchenko: {
		ID:     geoparse.RaionShevchenko,
		NameUA: "Шевченківський",
		Center: Coord{Lat: 50.460, Lon: 30.470},
		Boundary: []Coord{
			{Lat: 50.470, Lon: 30.420}, // Syrets / Dorohozhytska
			{Lat: 50.460, Lon: 30.485}, // Tatarka / Lukianivka border
			{Lat: 50.455, Lon: 30.530}, // Maidan / Khreshchatyk
			{Lat: 50.440, Lon: 30.520}, // Bessarabska / Shevchenka Blvd
			{Lat: 50.438, Lon: 30.480}, // Vokzalna / Zhylianska border
			{Lat: 50.450, Lon: 30.440}, // Shuliavka / Beresteiska (Peremohy Ave)
			{Lat: 50.462, Lon: 30.400}, // Nyvky / Shcherbakivskoho St
			{Lat: 50.470, Lon: 30.420},
		},
	},

	geoparse.RaionSolomyanskyi: {
		ID:     geoparse.RaionSolomyanskyi,
		NameUA: "Солом'янський",
		Center: Coord{Lat: 50.425, Lon: 30.445},
		Boundary: []Coord{
			{Lat: 50.450, Lon: 30.440}, // Shuliavka (south of Beresteiskyi Ave)
			{Lat: 50.438, Lon: 30.480}, // Central Railway Station
			{Lat: 50.430, Lon: 30.500}, // Batyieva Hora / Protasiv Yar
			{Lat: 50.408, Lon: 30.500}, // Chokolivka / Krasnozoryana (Lobanovskoho)
			{Lat: 50.375, Lon: 30.450}, // Zhulyany Airport / Ring Road
			{Lat: 50.395, Lon: 30.380}, // Borshchahivka border / Vidradnyi
			{Lat: 50.430, Lon: 30.400}, // Vidradnyi / Vaclava Havela border
			{Lat: 50.450, Lon: 30.440},
		},
	},

	geoparse.RaionSviatoshyn: {
		ID:     geoparse.RaionSviatoshyn,
		NameUA: "Святошинський",
		Center: Coord{Lat: 50.455, Lon: 30.360},
		Boundary: []Coord{
			{Lat: 50.495, Lon: 30.370}, // Lavina / Berkovets / Kotsiubynske border
			{Lat: 50.462, Lon: 30.400}, // Nyvky (west of Tupolieva)
			{Lat: 50.450, Lon: 30.400}, // Sviatoshyn / Beresteiska
			{Lat: 50.430, Lon: 30.390}, // Mykilska Borshchahivka
			{Lat: 50.395, Lon: 30.370}, // South Borshchahivka / Ring Road
			{Lat: 50.420, Lon: 30.270}, // Kyiv Ring / Petropavlivska border
			{Lat: 50.465, Lon: 30.280}, // Bilychi / Novobiliychi / Irpin border
			{Lat: 50.495, Lon: 30.370},
		},
	},

	geoparse.RaionHolosiivskyi: {
		ID:     geoparse.RaionHolosiivskyi,
		NameUA: "Голосіївський",
		Center: Coord{Lat: 50.375, Lon: 30.505},
		Boundary: []Coord{
			{Lat: 50.408, Lon: 30.525}, // Lybidska square / Ocean Plaza
			{Lat: 50.405, Lon: 30.565}, // Vydubychi / Promzona
			{Lat: 50.360, Lon: 30.580}, // Korchuvate / Zhukiv Island along Dnipro
			{Lat: 50.280, Lon: 30.570}, // Koncha-Zaspa / Kozyn city border
			{Lat: 50.320, Lon: 30.500}, // Pyrohiv / Feofaniya
			{Lat: 50.350, Lon: 30.430}, // Teremky / Respublika / Odeska Square
			{Lat: 50.375, Lon: 30.450}, // Zhulyany border / Sovky
			{Lat: 50.410, Lon: 30.490}, // Demiivka / Holosiivska Square
			{Lat: 50.408, Lon: 30.525},
		},
	},
}
