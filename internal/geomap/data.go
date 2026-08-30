package geomap

import "alert-userbot/internal/geoparse"

// Coord represents a (latitude, longitude) coordinate.
type Coord struct {
	Lat float64
	Lon float64
}

// Kyiv bounding box for projection (covers Kyiv city and immediate agglomeration)
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

// DniproRiverPolygon holds the coordinates along the Dnipro river flowing through Kyiv.
var DniproRiverPolygon = []Coord{
	// North entering Kyiv (Vyshhorod / Kyiv Reservoir dam)
	{Lat: 50.590, Lon: 30.505},
	{Lat: 50.565, Lon: 30.525},
	{Lat: 50.540, Lon: 30.535},
	// Obolon / Desna confluence
	{Lat: 50.518, Lon: 30.540},
	{Lat: 50.498, Lon: 30.535},
	// Pivnichnyi Bridge
	{Lat: 50.491, Lon: 30.538},
	// Trukhaniv Island / Podil
	{Lat: 50.470, Lon: 30.530},
	{Lat: 50.458, Lon: 30.533},
	// Poshtova ploshcha / Podilskyi bridge
	{Lat: 50.455, Lon: 30.530},
	// Metro bridge / Hydropark
	{Lat: 50.443, Lon: 30.569},
	// Paton bridge
	{Lat: 50.428, Lon: 30.575},
	// Darnytskyi bridge
	{Lat: 50.416, Lon: 30.589},
	// Pivdennyi bridge
	{Lat: 50.398, Lon: 30.588},
	// Zhukiv island / Kozyn direction (South)
	{Lat: 50.360, Lon: 30.580},
	{Lat: 50.320, Lon: 30.575},
	{Lat: 50.280, Lon: 30.570},
	{Lat: 50.250, Lon: 30.570},
}

// KyivRaionBoundaries holds boundary polygons for all 10 Kyiv districts.
var KyivRaionBoundaries = map[geoparse.RaionID]RaionPolygon{
	geoparse.RaionObolonskyi: {
		ID:     geoparse.RaionObolonskyi,
		NameUA: "Оболонський",
		Center: Coord{Lat: 50.510, Lon: 30.485},
		Boundary: []Coord{
			{Lat: 50.580, Lon: 30.340}, // Pushcha-Vodytsia NW
			{Lat: 50.585, Lon: 30.510}, // North dam / Dnipro
			{Lat: 50.520, Lon: 30.535}, // Dnipro along Obolon
			{Lat: 50.485, Lon: 30.530}, // Pivnichnyi bridge
			{Lat: 50.480, Lon: 30.470}, // Kurenivka boundary
			{Lat: 50.495, Lon: 30.420}, // Vynohradar border
			{Lat: 50.530, Lon: 30.340}, // West border
			{Lat: 50.580, Lon: 30.340},
		},
	},

	geoparse.RaionDesnyanskyi: {
		ID:     geoparse.RaionDesnyanskyi,
		NameUA: "Деснянський",
		Center: Coord{Lat: 50.515, Lon: 30.615},
		Boundary: []Coord{
			{Lat: 50.585, Lon: 30.510}, // North border
			{Lat: 50.570, Lon: 30.640}, // Troieshchyna North-East (TEC-6)
			{Lat: 50.530, Lon: 30.700}, // Bykivnya East
			{Lat: 50.470, Lon: 30.710}, // Brovary highway
			{Lat: 50.465, Lon: 30.630}, // Lisovyi / Brovarskyi Ave
			{Lat: 50.475, Lon: 30.580}, // Raiduzhnyi border
			{Lat: 50.491, Lon: 30.538}, // Pivnichnyi Bridge
			{Lat: 50.540, Lon: 30.535}, // Dnipro north
			{Lat: 50.585, Lon: 30.510},
		},
	},

	geoparse.RaionPodilskyi: {
		ID:     geoparse.RaionPodilskyi,
		NameUA: "Подільський",
		Center: Coord{Lat: 50.485, Lon: 30.435},
		Boundary: []Coord{
			{Lat: 50.495, Lon: 30.420}, // Vynohradar (Retroville)
			{Lat: 50.480, Lon: 30.470}, // Kurenivka
			{Lat: 50.485, Lon: 30.530}, // Havanskyi / Dnipro
			{Lat: 50.455, Lon: 30.530}, // Poshtova Square
			{Lat: 50.460, Lon: 30.480}, // Tatarka border
			{Lat: 50.475, Lon: 30.410}, // Syrets border
			{Lat: 50.510, Lon: 30.370}, // Berkovets
			{Lat: 50.530, Lon: 30.340}, // Pushcha border
			{Lat: 50.495, Lon: 30.420},
		},
	},

	geoparse.RaionShevchenko: {
		ID:     geoparse.RaionShevchenko,
		NameUA: "Шевченківський",
		Center: Coord{Lat: 50.460, Lon: 30.470},
		Boundary: []Coord{
			{Lat: 50.475, Lon: 30.410}, // Syrets
			{Lat: 50.460, Lon: 30.480}, // Tatarka / Podil border
			{Lat: 50.455, Lon: 30.530}, // Khreshchatyk / Maidan
			{Lat: 50.440, Lon: 30.520}, // Bessarabska / Shevchenko Blvd
			{Lat: 50.438, Lon: 30.475}, // Vokzalna / Zhylianska
			{Lat: 50.450, Lon: 30.430}, // Shuliavka / Peremohy Ave
			{Lat: 50.465, Lon: 30.395}, // Nyvky border
			{Lat: 50.475, Lon: 30.410},
		},
	},

	geoparse.RaionSviatoshyn: {
		ID:     geoparse.RaionSviatoshyn,
		NameUA: "Святошинський",
		Center: Coord{Lat: 50.455, Lon: 30.360},
		Boundary: []Coord{
			{Lat: 50.510, Lon: 30.370}, // Lavina / Berkovets
			{Lat: 50.475, Lon: 30.410}, // Nyvky
			{Lat: 50.450, Lon: 30.430}, // Shuliavka border
			{Lat: 50.430, Lon: 30.390}, // Borshchahivka
			{Lat: 50.395, Lon: 30.370}, // South Borshchahivka
			{Lat: 50.420, Lon: 30.250}, // West border (Kyiv Ring)
			{Lat: 50.470, Lon: 30.260}, // Bilychi / Irpin border
			{Lat: 50.510, Lon: 30.370},
		},
	},

	geoparse.RaionSolomyanskyi: {
		ID:     geoparse.RaionSolomyanskyi,
		NameUA: "Солом'янський",
		Center: Coord{Lat: 50.425, Lon: 30.445},
		Boundary: []Coord{
			{Lat: 50.450, Lon: 30.430}, // Shuliavka
			{Lat: 50.438, Lon: 30.475}, // Central Station / Zhylianska
			{Lat: 50.430, Lon: 30.500}, // Batyieva Gora
			{Lat: 50.410, Lon: 30.490}, // Protasiv Yar
			{Lat: 50.375, Lon: 30.450}, // Zhuliany airport / Ring Road
			{Lat: 50.395, Lon: 30.370}, // Borshchahivka border
			{Lat: 50.430, Lon: 30.390}, // Vidradnyi
			{Lat: 50.450, Lon: 30.430},
		},
	},

	geoparse.RaionPecherskyi: {
		ID:     geoparse.RaionPecherskyi,
		NameUA: "Печерський",
		Center: Coord{Lat: 50.430, Lon: 30.545},
		Boundary: []Coord{
			{Lat: 50.455, Lon: 30.530}, // Maidan / Poshtova
			{Lat: 50.443, Lon: 30.569}, // Metro bridge / Dnipro
			{Lat: 50.428, Lon: 30.575}, // Paton bridge / Dnipro
			{Lat: 50.405, Lon: 30.565}, // Vydubychi / TEC-5
			{Lat: 50.408, Lon: 30.525}, // Lybidska square
			{Lat: 50.430, Lon: 30.500}, // Protasiv / Klov
			{Lat: 50.440, Lon: 30.520}, // Bessarabska square
			{Lat: 50.455, Lon: 30.530},
		},
	},

	geoparse.RaionHolosiivskyi: {
		ID:     geoparse.RaionHolosiivskyi,
		NameUA: "Голосіївський",
		Center: Coord{Lat: 50.375, Lon: 30.505},
		Boundary: []Coord{
			{Lat: 50.408, Lon: 30.525}, // Lybidska
			{Lat: 50.405, Lon: 30.565}, // Vydubychi / Dnipro
			{Lat: 50.360, Lon: 30.580}, // Korchuvate / Dnipro
			{Lat: 50.290, Lon: 30.570}, // South city border / Kozyn
			{Lat: 50.320, Lon: 30.500}, // Pyrohiv / Feofaniya
			{Lat: 50.350, Lon: 30.430}, // Teremky / Respublika / Odeska ploshcha
			{Lat: 50.375, Lon: 30.450}, // Zhuliany border / Sovky
			{Lat: 50.410, Lon: 30.490}, // Demiivka
			{Lat: 50.408, Lon: 30.525},
		},
	},

	geoparse.RaionDniprovskyi: {
		ID:     geoparse.RaionDniprovskyi,
		NameUA: "Дніпровський",
		Center: Coord{Lat: 50.450, Lon: 30.600},
		Boundary: []Coord{
			{Lat: 50.491, Lon: 30.538}, // Pivnichnyi Bridge
			{Lat: 50.475, Lon: 30.580}, // Raiduzhnyi / Rusanivski Sady
			{Lat: 50.465, Lon: 30.630}, // Lisovyi / Brovarskyi Ave (Prospekt)
			{Lat: 50.445, Lon: 30.670}, // DVRZ border
			{Lat: 50.430, Lon: 30.630}, // Darnytska Sq / Leningradska
			{Lat: 50.420, Lon: 30.590}, // Berezniaky / Paton
			{Lat: 50.443, Lon: 30.569}, // Metro bridge / Rusanivka
			{Lat: 50.470, Lon: 30.530}, // Trukhaniv / Dnipro
			{Lat: 50.491, Lon: 30.538},
		},
	},

	geoparse.RaionDarnytskyi: {
		ID:     geoparse.RaionDarnytskyi,
		NameUA: "Дарницький",
		Center: Coord{Lat: 50.405, Lon: 30.660},
		Boundary: []Coord{
			{Lat: 50.430, Lon: 30.630}, // Darnytska Square
			{Lat: 50.445, Lon: 30.670}, // DVRZ / Rembaza
			{Lat: 50.440, Lon: 30.730}, // East border / Lis
			{Lat: 50.360, Lon: 30.760}, // Bortnychi SE
			{Lat: 50.340, Lon: 30.660}, // Osokorky Dachas / Nyzhni Sady
			{Lat: 50.398, Lon: 30.588}, // Pivdennyi bridge / River Mall
			{Lat: 50.416, Lon: 30.589}, // Darnytskyi bridge
			{Lat: 50.420, Lon: 30.590}, // Berezniaky border
			{Lat: 50.430, Lon: 30.630},
		},
	},
}
