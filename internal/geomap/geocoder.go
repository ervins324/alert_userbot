package geomap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"alert-userbot/internal/geoparse"
)

// ErrLocationNotFound is returned when a requested location cannot be resolved.
var ErrLocationNotFound = errors.New("location not found in Kyiv")

// Coordinates represents geographic latitude and longitude.
type Coordinates = Coord

// StaticKyivLocations provides a fallback / fast-path dictionary for critical
// Kyiv landmarks, districts, neighborhoods, transport hubs, and facilities.
// When a query matches this map, no external network request is made.
var StaticKyivLocations = map[string]Coordinates{
	// Districts (Administrative Raions)
	"оболонь":              {Lat: 50.5050, Lon: 30.5010},
	"оболонський":          {Lat: 50.5100, Lon: 30.4850},
	"оболонський район":    {Lat: 50.5100, Lon: 30.4850},
	"позняки":              {Lat: 50.3980, Lon: 30.6340},
	"осокорки":             {Lat: 50.3920, Lon: 30.6180},
	"троєщина":             {Lat: 50.5180, Lon: 30.6020},
	"деснянський":          {Lat: 50.5150, Lon: 30.6150},
	"деснянський район":    {Lat: 50.5150, Lon: 30.6150},
	"дарниця":              {Lat: 50.4400, Lon: 30.6200},
	"дарницький":           {Lat: 50.4050, Lon: 30.6600},
	"дарницький район":     {Lat: 50.4050, Lon: 30.6600},
	"дніпровський":         {Lat: 50.4500, Lon: 30.6000},
	"дніпровський район":   {Lat: 50.4500, Lon: 30.6000},
	"печерськ":             {Lat: 50.4310, Lon: 30.5450},
	"печерський":           {Lat: 50.4300, Lon: 30.5450},
	"печерський район":     {Lat: 50.4300, Lon: 30.5450},
	"поділ":                {Lat: 50.4680, Lon: 30.5170},
	"подільський":          {Lat: 50.4850, Lon: 30.4350},
	"подільський район":    {Lat: 50.4850, Lon: 30.4350},
	"святошин":             {Lat: 50.4570, Lon: 30.3700},
	"святошино":            {Lat: 50.4570, Lon: 30.3700},
	"святошинський":        {Lat: 50.4550, Lon: 30.3600},
	"святошинський район":  {Lat: 50.4550, Lon: 30.3600},
	"солом'янка":           {Lat: 50.4310, Lon: 30.4700},
	"соломянка":            {Lat: 50.4310, Lon: 30.4700},
	"солом'янський":        {Lat: 50.4250, Lon: 30.4450},
	"соломянський":         {Lat: 50.4250, Lon: 30.4450},
	"солом'янський район":  {Lat: 50.4250, Lon: 30.4450},
	"шевченківський":       {Lat: 50.4600, Lon: 30.4700},
	"шевченківський район": {Lat: 50.4600, Lon: 30.4700},
	"голосіїв":             {Lat: 50.3920, Lon: 30.5050},
	"голосієво":            {Lat: 50.3920, Lon: 30.5050},
	"голосіївський":        {Lat: 50.3750, Lon: 30.5050},
	"голосіївський район":  {Lat: 50.3750, Lon: 30.5050},

	// Critical Landmarks explicitly mentioned in prompt
	"лук'янівка": {Lat: 50.4620, Lon: 30.4820},
	"лукянівка":  {Lat: 50.4620, Lon: 30.4820},
	"видубичі":   {Lat: 50.4030, Lon: 30.5600},

	// Neighborhoods & Microdistricts
	"харківський масив":   {Lat: 50.4070, Lon: 30.6650},
	"лісовий масив":       {Lat: 50.4780, Lon: 30.6350},
	"лісовий":             {Lat: 50.4780, Lon: 30.6350},
	"воскресенка":         {Lat: 50.4850, Lon: 30.5900},
	"русанівка":           {Lat: 50.4380, Lon: 30.5970},
	"березняки":           {Lat: 50.4280, Lon: 30.6010},
	"лівобережний":        {Lat: 50.4520, Lon: 30.5980},
	"лівобережна":         {Lat: 50.4520, Lon: 30.5980},
	"райдужний":           {Lat: 50.4890, Lon: 30.5820},
	"гідропарк":           {Lat: 50.4430, Lon: 30.5770},
	"соцмісто":            {Lat: 50.4490, Lon: 30.6200},
	"дврз":                {Lat: 50.4480, Lon: 30.6860},
	"рембаза":             {Lat: 50.4250, Lon: 30.6900},
	"бортничі":            {Lat: 50.3750, Lon: 30.7000},
	"червоний хутір":      {Lat: 50.4080, Lon: 30.6920},
	"мінський масив":      {Lat: 50.5180, Lon: 30.4630},
	"пріорка":             {Lat: 50.4950, Lon: 30.4580},
	"куренівка":           {Lat: 50.4880, Lon: 30.4700},
	"виноградар":          {Lat: 50.5080, Lon: 30.4150},
	"вітряні гори":        {Lat: 50.5000, Lon: 30.4350},
	"мостицький":          {Lat: 50.4980, Lon: 30.4370},
	"берковець":           {Lat: 50.4950, Lon: 30.3580},
	"пуща-водиця":         {Lat: 50.5400, Lon: 30.3550},
	"пуща водиця":         {Lat: 50.5400, Lon: 30.3550},
	"почайна":             {Lat: 50.4870, Lon: 30.4980},
	"петрівка":            {Lat: 50.4870, Lon: 30.4980},
	"липки":               {Lat: 50.4430, Lon: 30.5330},
	"клов":                {Lat: 50.4380, Lon: 30.5350},
	"звіринець":           {Lat: 50.4190, Lon: 30.5550},
	"арсенальна":          {Lat: 50.4420, Lon: 30.5520},
	"сирець":              {Lat: 50.4740, Lon: 30.4350},
	"шулявка":             {Lat: 50.4540, Lon: 30.4450},
	"нивки":               {Lat: 50.4610, Lon: 30.4040},
	"хрещатик":            {Lat: 50.4470, Lon: 30.5220},
	"майдан":              {Lat: 50.4501, Lon: 30.5234},
	"майдан незалежності": {Lat: 50.4501, Lon: 30.5234},
	"центр":               {Lat: 50.4470, Lon: 30.5220},
	"кпі":                 {Lat: 50.4500, Lon: 30.4570},
	"політех":             {Lat: 50.4500, Lon: 30.4570},
	"татарка":             {Lat: 50.4700, Lon: 30.4900},
	"відрадний":           {Lat: 50.4350, Lon: 30.4200},
	"чоколівка":           {Lat: 50.4190, Lon: 30.4570},
	"совки":               {Lat: 50.4050, Lon: 30.4880},
	"караваєві дачі":      {Lat: 50.4370, Lon: 30.4460},
	"кардачі":             {Lat: 50.4370, Lon: 30.4460},
	"батиєва гора":        {Lat: 50.4300, Lon: 30.4920},
	"борщагівка":          {Lat: 50.4150, Lon: 30.3750},
	"академмістечко":      {Lat: 50.4680, Lon: 30.3550},
	"біличі":              {Lat: 50.4630, Lon: 30.3400},
	"новобіличі":          {Lat: 50.4630, Lon: 30.3400},
	"теремки":             {Lat: 50.3660, Lon: 30.4550},
	"корчувате":           {Lat: 50.3670, Lon: 30.5600},
	"деміївка":            {Lat: 50.4050, Lon: 30.5180},
	"пирогів":             {Lat: 50.3520, Lon: 30.5100},
	"пирогово":            {Lat: 50.3520, Lon: 30.5100},
	"феофанія":            {Lat: 50.3420, Lon: 30.4850},
	"вднг":                {Lat: 50.3780, Lon: 30.4780},
	"либідська":           {Lat: 50.4130, Lon: 30.5240},

	// Energy & Transport Facilities
	"тец-5":              {Lat: 50.3950, Lon: 30.5650},
	"тец 5":              {Lat: 50.3950, Lon: 30.5650},
	"тец5":               {Lat: 50.3950, Lon: 30.5650},
	"тец-6":              {Lat: 50.5310, Lon: 30.6540},
	"тец 6":              {Lat: 50.5310, Lon: 30.6540},
	"тец6":               {Lat: 50.5310, Lon: 30.6540},
	"вокзал":             {Lat: 50.4400, Lon: 30.4890},
	"центральний вокзал": {Lat: 50.4400, Lon: 30.4890},
	"залізничний вокзал": {Lat: 50.4400, Lon: 30.4890},
	"жуляни":             {Lat: 50.4020, Lon: 30.4490},
	"аеропорт київ":      {Lat: 50.4020, Lon: 30.4490},
	"аеропорт жуляни":    {Lat: 50.4020, Lon: 30.4490},
	"бориспіль":          {Lat: 50.3520, Lon: 30.9550},
	"бровари":            {Lat: 50.5110, Lon: 30.7900},
	"ірпінь":             {Lat: 50.5200, Lon: 30.2450},
	"буча":               {Lat: 50.5480, Lon: 30.2200},
	"гостомель":          {Lat: 50.5680, Lon: 30.2650},
	"вишгород":           {Lat: 50.5830, Lon: 30.4900},

	// Bridges
	"південний міст":   {Lat: 50.3980, Lon: 30.5880},
	"північний міст":   {Lat: 50.4910, Lon: 30.5380},
	"московський міст": {Lat: 50.4910, Lon: 30.5380},
	"подільський міст": {Lat: 50.4720, Lon: 30.5350},
	"міст патона":      {Lat: 50.4280, Lon: 30.5750},
	"міст метро":       {Lat: 50.4430, Lon: 30.5690},
	"дарницький міст":  {Lat: 50.4160, Lon: 30.5890},
}

type geocodeCacheItem struct {
	coords      Coordinates
	displayName string
	found       bool
}

// Geocoder handles location resolution using static dictionary, sync.Map in-memory cache,
// and OpenStreetMap Nominatim API with rate limiting (<= 1 req/sec).
type Geocoder struct {
	client    *http.Client
	cache     sync.Map // normalized query string -> *geocodeCacheItem
	rateMu    sync.Mutex
	lastReq   time.Time
	minReqGap time.Duration
	baseURL   string
	userAgent string
	logger    *slog.Logger
}

// NewGeocoder creates a new Geocoder instance.
func NewGeocoder(client *http.Client, logger *slog.Logger) *Geocoder {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Geocoder{
		client:    client,
		minReqGap: 1050 * time.Millisecond, // strictly complies with Nominatim 1 req/sec policy
		baseURL:   "https://nominatim.openstreetmap.org/search",
		userAgent: "KyivAirAlertMonitor/1.0 (https://github.com/alert-userbot)",
		logger:    logger,
	}
}

// LookupStatic checks the static fallback dictionary for a given place name.
func LookupStatic(query string) (Coordinates, string, bool) {
	norm := normalizeQuery(query)
	if coords, ok := StaticKyivLocations[norm]; ok {
		return coords, titleCase(norm), true
	}

	// Try removing common Ukrainian prepositions & prefixes
	prefixes := []string{"на ", "в ", "у ", "біля ", "р-н ", "район ", "масив ", "вул. ", "вулиця ", "пр-т ", "проспект "}
	trimmed := norm
	for _, p := range prefixes {
		if strings.HasPrefix(trimmed, p) {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, p))
			if coords, ok := StaticKyivLocations[trimmed]; ok {
				return coords, titleCase(trimmed), true
			}
		}
	}

	// Check against geoparse database for inflections & known Kyiv landmarks
	if loc := geoparse.ExtractLocation(norm); loc != nil {
		if len(loc.Points) > 0 {
			pt := loc.Points[0]
			return Coordinates{Lat: pt.Lat, Lon: pt.Lon}, pt.NameUA, true
		}
		if len(loc.MatchedRaions) > 0 {
			rID := loc.MatchedRaions[0]
			if rInfo, exists := geoparse.AllRaions[rID]; exists {
				return Coordinates{Lat: rInfo.CenterLat, Lon: rInfo.CenterLon}, rInfo.NameUA, true
			}
		}
	}

	return Coordinates{}, "", false
}

func normalizeQuery(query string) string {
	q := strings.ToLower(strings.TrimSpace(query))
	q = strings.TrimPrefix(q, "/map")
	q = strings.TrimSpace(q)
	q = strings.Trim(q, "!?.,;:'\"`«»()[]{}*~")
	return strings.TrimSpace(q)
}

func prepareNominatimQuery(query string) string {
	norm := strings.TrimSpace(query)
	norm = strings.TrimPrefix(norm, "/map")
	norm = strings.TrimSpace(norm)
	lower := strings.ToLower(norm)

	hasKyiv := strings.Contains(lower, "київ") || strings.Contains(lower, "kyiv") || strings.Contains(lower, "kiev")
	hasUA := strings.Contains(lower, "україна") || strings.Contains(lower, "ukraine")

	if !hasKyiv && !hasUA {
		return norm + ", Київ, Україна"
	}
	if hasKyiv && !hasUA {
		return norm + ", Україна"
	}
	return norm
}

func titleCase(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return ""
	}
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func (g *Geocoder) waitRateLimit(ctx context.Context) error {
	g.rateMu.Lock()
	defer g.rateMu.Unlock()

	elapsed := time.Since(g.lastReq)
	if elapsed < g.minReqGap {
		wait := g.minReqGap - elapsed
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	g.lastReq = time.Now()
	return nil
}

// Geocode resolves a location query to geographic coordinates.
// It checks:
// 1. Static dictionary (fast-path fallback)
// 2. sync.Map in-memory cache
// 3. OpenStreetMap Nominatim API with ", Київ, Україна" context suffix and 1 req/sec rate limit.
func (g *Geocoder) Geocode(ctx context.Context, query string) (*Coordinates, string, error) {
	normKey := normalizeQuery(query)
	if normKey == "" {
		return nil, "", ErrLocationNotFound
	}

	// 1. Check static dictionary (0 network requests)
	if coords, name, ok := LookupStatic(normKey); ok {
		g.logger.Debug("resolved location via static Kyiv dictionary",
			slog.String("query", query),
			slog.Float64("lat", coords.Lat),
			slog.Float64("lon", coords.Lon))
		return &coords, name, nil
	}

	// 2. Check in-memory sync.Map cache
	if val, ok := g.cache.Load(normKey); ok {
		item := val.(*geocodeCacheItem)
		if !item.found {
			return nil, "", ErrLocationNotFound
		}
		g.logger.Debug("resolved location via in-memory sync.Map cache",
			slog.String("query", query),
			slog.Float64("lat", item.coords.Lat),
			slog.Float64("lon", item.coords.Lon))
		return &item.coords, item.displayName, nil
	}

	// 3. Prepare OSM Nominatim query with context suffix
	fullQuery := prepareNominatimQuery(query)

	// 4. Rate-limit wait
	if err := g.waitRateLimit(ctx); err != nil {
		return nil, "", err
	}

	// 5. Query Nominatim API
	u, err := url.Parse(g.baseURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid geocoder base url: %w", err)
	}

	params := url.Values{}
	params.Set("q", fullQuery)
	params.Set("format", "jsonv2")
	params.Set("limit", "1")
	params.Set("countrycodes", "ua")
	params.Set("viewbox", fmt.Sprintf("%.3f,%.3f,%.3f,%.3f", MinLon, MaxLat, MaxLon, MinLat))
	params.Set("bounded", "0")
	params.Set("accept-language", "uk,en")
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to build geocoder request: %w", err)
	}
	req.Header.Set("User-Agent", g.userAgent)
	req.Header.Set("Accept", "application/json")

	g.logger.Info("querying OpenStreetMap Nominatim API",
		slog.String("query", fullQuery),
		slog.String("url", u.String()))

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("nominatim http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		g.logger.Warn("nominatim returned non-200 status",
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(b)))
		return nil, "", fmt.Errorf("nominatim returned status %d", resp.StatusCode)
	}

	type nominatimPlace struct {
		PlaceID     int64  `json:"place_id"`
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		DisplayName string `json:"display_name"`
		Name        string `json:"name"`
	}

	var results []nominatimPlace
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&results); err != nil {
		return nil, "", fmt.Errorf("failed to decode nominatim response: %w", err)
	}

	if len(results) == 0 {
		g.cache.Store(normKey, &geocodeCacheItem{found: false})
		return nil, "", ErrLocationNotFound
	}

	lat, errLat := strconv.ParseFloat(results[0].Lat, 64)
	lon, errLon := strconv.ParseFloat(results[0].Lon, 64)
	if errLat != nil || errLon != nil {
		return nil, "", fmt.Errorf("invalid coordinates from nominatim: lat=%s lon=%s", results[0].Lat, results[0].Lon)
	}

	// Basic boundary check for Ukraine / Kyiv region
	if lat < 44.0 || lat > 53.0 || lon < 22.0 || lon > 41.0 {
		g.cache.Store(normKey, &geocodeCacheItem{found: false})
		return nil, "", ErrLocationNotFound
	}

	displayName := results[0].DisplayName
	if results[0].Name != "" {
		displayName = results[0].Name
	}

	coords := Coordinates{Lat: lat, Lon: lon}

	// Cache successful lookup in sync.Map
	g.cache.Store(normKey, &geocodeCacheItem{
		coords:      coords,
		displayName: displayName,
		found:       true,
	})

	g.logger.Info("successfully geocoded location via Nominatim",
		slog.String("query", query),
		slog.String("display_name", displayName),
		slog.Float64("lat", lat),
		slog.Float64("lon", lon))

	return &coords, displayName, nil
}

// Resolve attempts to resolve input text into a LocationResult:
// 1. If text matches known Kyiv places or raions via geoparse.ExtractLocation, it returns that.
// 2. Otherwise it geocodes using the fallback dictionary, sync.Map cache, and Nominatim API.
func (g *Geocoder) Resolve(ctx context.Context, text string) (*geoparse.LocationResult, error) {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return nil, ErrLocationNotFound
	}

	// 1. Check if geoparse.ExtractLocation detects multiple points or alerts
	loc := geoparse.ExtractLocation(clean)
	if loc != nil && (len(loc.Points) > 0 || len(loc.MatchedRaions) > 0) {
		return loc, nil
	}

	// 2. Geocode query
	coords, name, err := g.Geocode(ctx, clean)
	if err != nil {
		return nil, err
	}

	title := clean
	title = strings.TrimPrefix(title, "/map")
	title = strings.TrimSpace(title)
	if title == "" {
		title = name
	}

	return &geoparse.LocationResult{
		Points: []geoparse.PointMatch{
			{
				NameUA: title,
				Lat:    coords.Lat,
				Lon:    coords.Lon,
			},
		},
		Description: title,
	}, nil
}
