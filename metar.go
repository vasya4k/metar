// Package metar provides METAR (METeorological Aerodrome Report) message decoding
package metar

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vasya4k/metar/clouds"
	cnv "github.com/vasya4k/metar/conversion"
	ph "github.com/vasya4k/metar/phenomena"
	rwy "github.com/vasya4k/metar/runways"
	vis "github.com/vasya4k/metar/visibility"
	"github.com/vasya4k/metar/wind"
)

// CurYearStr - year of message. By default read all messages in the current date. Can be redefined if necessary
var CurYearStr string

// CurMonthStr - month of message
var CurMonthStr string

// CurDayStr - day of message
var CurDayStr string

func init() {
	now := time.Now()
	CurYearStr = strconv.Itoa(now.Year())
	CurMonthStr = fmt.Sprintf("%02d", now.Month())
	CurDayStr = fmt.Sprintf("%02d", now.Day())
}

// MetarMessage - Meteorological report presented as a data structure
type MetarMessage struct {
	RawData                      string                 `json:"raw_data"`  // The raw METAR
	COR                          bool                   `json:"corrected"` // Correction to observation
	Station                      string                 `json:"station"`   // 4-letter ICAO station identifier
	DateTime                     time.Time              `json:"issued"`    // Time (in ISO8601 date/time format) this METAR was observed
	Auto                         bool                   `json:"auto"`      // METAR from automatic observing systems with no human intervention
	NIL                          bool                   `json:"nil"`       // event of missing METAR
	wind.Wind                                           //	Surface wind
	CAVOK                        bool                   `json:"cavok"` // Ceiling And Visibility OK, indicating no cloud below 5,000 ft (1,500 m) or the highest minimum sector altitude and no cumulonimbus or towering cumulus at any level, a visibility of 10 km (6 mi) or more and no significant weather change.
	vis.Visibility                                      // Horizontal visibility
	RWYvisibility                []rwy.VisualRange      `json:"visual_range"` // Runway visual range
	ph.Phenomena                 `json:"phenomena"`     // Present Weather
	PhenomenaNotDefined          bool                   `json:"phenomena_not_defined"`           // Not detected by the automatic station - “//”
	VerticalVisibility           int                    `json:"vertical_visibility"`             // Vertical visibility (ft)
	VerticalVisibilityNotDefined bool                   `json:"vertical_visibility_not_defined"` // “///”
	clouds.Clouds                `json:"clouds"`        // Cloud amount and height
	Temperature                  int                    `json:"temperature"`
	Dewpoint                     int                    `json:"dew_point"` // Dew point in degrees Celsius
	QNHhPa                       int                    `json:"qnh_hpa"`   // Altimeter setting.  Atmospheric pressure adjusted to mean sea level
	RecentPhenomena              ph.Phenomena           `json:"recent_phenomena"`
	RWYState                     []rwy.State            `json:"runway_state"`
	WindShear                    []rwy.RunwayDesignator `json:"wind_shear"`
	TREND                        []Trend                `json:"trend"`
	NOSIG                        bool                   `json:"no_sig"`             //OR NO SIGnificant changes coming within the next two hours
	Remarks                      *Remark                `json:"remarks"`            //OR NO SIGnificant changes coming within the next two hours
	NotDecodedTokens             []string               `json:"not_decoded_tokens"` // An array of tokens that couldn't be decoded
}

// RAW - returns the original message text
func (m *MetarMessage) RAW() string { return m.RawData }

func (m *MetarMessage) appendTrend(input []string) {
	if trend := parseTrendData(input); trend != nil {
		m.TREND = append(m.TREND, *trend)
	}
}

// NewMETAR - creates a new METAR based on the original message
func NewMETAR(inputtext string) (*MetarMessage, error) {

	m := &MetarMessage{
		RawData: inputtext,
	}
	headerRx := myRegexp{regexp.MustCompile(`^(?P<type>(METAR|SPECI)\s)?(?P<cor>COR\s)?(?P<station>\w{4})\s(?P<time>\d{6}Z)(?P<auto>\sAUTO)?(?P<nil>\sNIL)?`)}
	headermap := headerRx.FindStringSubmatchMap(m.RawData)
	m.Station = headermap["station"]
	m.DateTime, _ = time.Parse("200601021504Z", CurYearStr+CurMonthStr+headermap["time"])
	m.COR = headermap["cor"] != ""
	m.Auto = headermap["auto"] != ""
	m.NIL = headermap["nil"] != ""
	if m.Station == "" && m.DateTime.IsZero() {
		return m, fmt.Errorf("not valid message in input")
	}
	if m.NIL {
		return m, nil
	}
	tokens := strings.Split(m.RawData, " ")
	count := 0
	totalcount := len(tokens)
	// skip station info, date/time, etc.
	for _, value := range headermap {
		if value != "" {
			count++
		}
	}

	var trends [][]string
	var remarks []string
	// split the array of tokens to parts: main section, remarks and trends
	// First, let's remove the RMK group, as it can contain TEMPO (RMK WHT TEMPO GRN)
	for i := totalcount - 1; i > count; i-- {
		if tokens[i] == "RMK" {
			remarks = append(remarks, tokens[i:totalcount]...)
			totalcount = i
		}
	}
	for i := totalcount - 1; i > count; i-- {
		if tokens[i] == TEMPO || tokens[i] == BECMG {
			//for correct order of following on reverse parsing append []trends to current trend
			trends = append([][]string{tokens[i:totalcount]}, trends[0:]...)
			totalcount = i
		}
	}
	// trends
	for _, trendstr := range trends {
		m.appendTrend(trendstr)
	}
	// remarks
	m.Remarks = parseRemarks(remarks)
	// main section
	m.decodeMetar(tokens[count:totalcount])
	return m, nil
}

type myRegexp struct {
	*regexp.Regexp
}

func (r *myRegexp) FindStringSubmatchMap(s string) map[string]string {
	captures := make(map[string]string)
	match := r.FindStringSubmatch(s)
	if match == nil {
		return captures
	}
	for i, name := range r.SubexpNames() {
		// Ignore the whole regexp match and unnamed groups
		if i == 0 || name == "" {
			continue
		}
		captures[name] = match[i]
	}
	return captures
}

func (m *MetarMessage) decodeMetar(tokens []string) {

	if tokens[len(tokens)-1] == "NOSIG" {
		m.NOSIG = true
		tokens = tokens[:len(tokens)-1]
	}
	totalcount := len(tokens)
	for count := 0; count < totalcount; {
		// Surface wind
		count += m.ParseWind(strings.Join(tokens[count:], " "))
		if tokens[count] == "CAVOK" {
			m.CAVOK = true
			count++
		} else {
			count = setMetarWeatherCondition(m, count, tokens)
		} //end !CAVOK
		// Temperature and dew point
		if m.setTemperature(tokens[count]) {
			count++
		}
		// Altimeter setting
		if m.setAltimetr(tokens[count]) {
			count++
		}
		//	All the following elements are optional
		// Recent weather
		for count < totalcount && m.RecentPhenomena.AppendRecentPhenomena(tokens[count]) {
			count++
		}
		// Wind shear
		if ok, tokensused := m.appendWindShears(tokens, count); ok {
			count += tokensused
		}
		// TODO Sea surface condition
		// W19/S4  W15/Н7  W15/Н17 W15/Н175

		// State of the runway(s)
		for count < totalcount && m.appendRunwayState(tokens[count]) {
			count++
		}
		// The token is not recognized or is located in the wrong position
		if count < totalcount {
			m.NotDecodedTokens = append(m.NotDecodedTokens, tokens[count])
			count++
		}
	} // End main section
}

func setMetarWeatherCondition(m *MetarMessage, count int, tokens []string) int {
	// Horizontal visibility
	if tokensused := m.ParseVisibility(tokens[count:]); tokensused > 0 {
		count += tokensused
	}
	// Runway visual range
	for count < len(tokens) && m.appendRunwayVisualRange(tokens[count]) {
		count++
	}
	// Present Weather
	if count < len(tokens) && tokens[count] == "//" {
		m.PhenomenaNotDefined = true
		count++
	}
	for count < len(tokens) && m.AppendPhenomena(tokens[count]) {
		count++
	}
	// Vertical visibility
	if count < len(tokens) && m.setVerticalVisibility(tokens[count]) {
		count++
	}
	// Cloudiness description
	for count < len(tokens) && m.AppendCloud(tokens[count]) {
		count++
	}
	return count
}

// Checks whether the string is a temperature and dew point values and writes this values
func (m *MetarMessage) setTemperature(input string) bool {
	regex := regexp.MustCompile(`^(M)?(\d{2})/(M)?(\d{2})$`)
	matches := regex.FindStringSubmatch(input)
	if len(matches) != 0 {
		m.Temperature, _ = strconv.Atoi(matches[2])
		m.Dewpoint, _ = strconv.Atoi(matches[4])
		if matches[1] == "M" {
			m.Temperature = -m.Temperature
		}
		if matches[3] == "M" {
			m.Dewpoint = -m.Dewpoint
		}
		return true
	}
	return false
}

func (m *MetarMessage) setAltimetr(input string) bool {
	regex := regexp.MustCompile(`([Q|A])(\d{4})`)
	matches := regex.FindStringSubmatch(input)
	if len(matches) != 0 {
		if matches[1] == "A" {
			inHg, _ := strconv.ParseFloat(matches[2][:2]+"."+matches[2][2:4], 64)
			m.QNHhPa = int(cnv.InHgTohPa(inHg))
		} else {
			m.QNHhPa, _ = strconv.Atoi(matches[2])
		}
		return true
	}
	return false
}

func (m *MetarMessage) appendRunwayVisualRange(input string) bool {
	if RWYvis, ok := rwy.ParseVisualRange(input); ok {
		m.RWYvisibility = append(m.RWYvisibility, RWYvis)
		return true
	}
	return false
}

func (m *MetarMessage) setVerticalVisibility(input string) bool {
	if vv, nd, ok := parseVerticalVisibility(input); ok {
		m.VerticalVisibility = vv
		m.VerticalVisibilityNotDefined = nd
		return true
	}
	return false
}

func (m *MetarMessage) appendRunwayState(input string) bool {

	if input == "R/SNOCLO" {
		rwc := new(rwy.State)
		rwc.SNOCLO = true
		m.RWYState = append(m.RWYState, *rwc)
		return true
	}
	if rwc, ok := rwy.ParseState(input); ok {
		m.RWYState = append(m.RWYState, rwc)
		return true
	}
	return false
}

func parseVerticalVisibility(input string) (vv int, nd bool, ok bool) {

	regex := regexp.MustCompile(`VV(\d{3}|///)`)
	matches := regex.FindStringSubmatch(input)
	if len(matches) != 0 && matches[0] != "" {
		ok = true
		if matches[1] != "///" {
			vv, _ = strconv.Atoi(matches[1])
			vv *= 100
		} else {
			nd = true
		}
	}
	return
}

func (m *MetarMessage) appendWindShears(tokens []string, count int) (ok bool, tokensused int) {

	regex := regexp.MustCompile(`^WS\s((R\d{2}[LCR]?)|(ALL\sRWY))`)
	matches := regex.FindStringSubmatch(strings.Join(tokens[count:], " "))
	for ; len(matches) > 0; matches = regex.FindStringSubmatch(strings.Join(tokens[count:], " ")) {
		ok = true
		if matches[3] != "" { // WS ALL RWY
			rd := new(rwy.RunwayDesignator)
			rd.AllRunways = true
			m.WindShear = append(m.WindShear, *rd)
			tokensused += 3
			count += 3
		}
		if matches[2] != "" { // WS R03
			m.WindShear = append(m.WindShear, rwy.NewRD(matches[1]))
			tokensused += 2
			count += 2
		}
	}
	return
}

type weatherCondition interface {
	ParseVisibility([]string) int
	AppendPhenomena(string) bool
	setVerticalVisibility(string) bool
	AppendCloud(string) bool
}

// decoder for not CAVOK conditions in TAF messages and trends. Returns new current position in []string
func decodeWeatherCondition(t weatherCondition, count int, tokens []string) int {
	// Horizontal visibility.
	if count < len(tokens) {
		count += t.ParseVisibility(tokens[count:])
	}
	// Weather or NSW - no significant weather
	for count < len(tokens) && t.AppendPhenomena(tokens[count]) {
		count++
	}
	// Vertical visibility
	if count < len(tokens) && t.setVerticalVisibility(tokens[count]) {
		count++
	}
	// Clouds.
	for count < len(tokens) && t.AppendCloud(tokens[count]) {
		count++
	}
	return count
}
