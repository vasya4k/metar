package metar

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/vasya4k/metar/clouds"
	"github.com/vasya4k/metar/phenomena"
)

func TestParseTrendData(t *testing.T) {

	Convey("Trends parsing tests", t, func() {
		var input []string
		var expected *Trend

		Convey(`test incorrect time, horizontal visibility and phenomena`, func() {
			input = []string{"BECMG", "2526/2526", "0500", "FG"}

			expected = &Trend{Type: BECMG,
				Visibility: getVis([]string{"0500"}),
				Phenomena:  []phenomena.Phenomenon{{Vicinity: false, Abbreviation: "FG", Intensity: ""}},
				ParsingErrors: []error{
					&time.ParseError{Layout: "2006010215", Value: "2023052526", LayoutElem: "15", ValueElem: "", Message: ": hour out of range"},
					&time.ParseError{Layout: "2006010215", Value: "2023052526", LayoutElem: "15", ValueElem: "", Message: ": hour out of range"},
				},
			}
			So(parseTrendData(input), ShouldResemble, expected)
		})

		Convey(`test misspelled time and cloud layer`, func() {
			input = []string{"BECMG", "AT220O", "TL23O0", "BKN015//"}
			expected = &Trend{Type: BECMG,
				Clouds: []clouds.Cloud{getCloud("BKN015//")},
				ParsingErrors: []error{
					&time.ParseError{Layout: "200601021504", Value: "20230531220O", LayoutElem: "04", ValueElem: "0O", Message: ""},
					&time.ParseError{Layout: "200601021504", Value: "2023053123O0", LayoutElem: "04", ValueElem: "O0", Message: ""},
				},
			}
			So(parseTrendData(input), ShouldResemble, expected)
		})

		Convey(`test for FM time and CAVOK condition`, func() {
			input = []string{"FM241200", "18003MPS", "CAVOK"}
			expected = &Trend{Type: FM,
				FM:    time.Date(curYear, curMonth, 24, 12, 0, 0, 0, time.UTC),
				Wind:  getWind("18003MPS"),
				CAVOK: true,

				ParsingErrors: nil,
			}
			So(parseTrendData(input), ShouldResemble, expected)
		})

		Convey(`test for vertical visibility and phenomena`, func() {
			input = []string{"TEMPO", "2509/2515", "0500", "FG", "VV003"}
			expected = &Trend{Type: TEMPO,
				FM:                 time.Date(curYear, curMonth, 25, 9, 0, 0, 0, time.UTC),
				TL:                 time.Date(curYear, curMonth, 25, 15, 0, 0, 0, time.UTC),
				Visibility:         getVis([]string{"0500"}),
				Phenomena:          []phenomena.Phenomenon{{Vicinity: false, Abbreviation: "FG", Intensity: ""}},
				VerticalVisibility: 300,
				ParsingErrors:      nil,
			}
			So(parseTrendData(input), ShouldResemble, expected)
		})

		Convey(`test for phenomena and multiple cloud layers`, func() {
			input = []string{"TEMPO", "2506/2512", "3100", "-SHRA", "BR", "BKN005", "OVC020CB"}
			expected = &Trend{Type: TEMPO,
				FM:            time.Date(curYear, curMonth, 25, 6, 0, 0, 0, time.UTC),
				TL:            time.Date(curYear, curMonth, 25, 12, 0, 0, 0, time.UTC),
				Visibility:    getVis([]string{"3100"}),
				Phenomena:     []phenomena.Phenomenon{{Vicinity: false, Abbreviation: "SHRA", Intensity: "-"}, {Vicinity: false, Abbreviation: "BR", Intensity: ""}},
				Clouds:        []clouds.Cloud{getCloud("BKN005"), getCloud("OVC020CB")},
				ParsingErrors: nil,
			}
			So(parseTrendData(input), ShouldResemble, expected)
		})

		Convey(`test for wind and expected time of changes`, func() {
			input = []string{"TEMPO", "FM1230", "TL1330", "18005MPS", "CAVOK"}
			expected = &Trend{Type: TEMPO,
				Wind:          getWind("18005MPS"),
				CAVOK:         true,
				FM:            time.Date(curYear, curMonth, curDay, 12, 30, 0, 0, time.UTC),
				TL:            time.Date(curYear, curMonth, curDay, 13, 30, 0, 0, time.UTC),
				ParsingErrors: nil,
			}
			So(parseTrendData(input), ShouldResemble, expected)
		})

		Convey(`test incorrect FM time and 24 hours at TL-time `, func() {
			input = []string{"BECMG", "FM22o0", "TL2400", "BKN015//"}
			expected = &Trend{Type: BECMG,
				TL:     time.Date(curYear, curMonth, curDay+1, 00, 0, 0, 0, time.UTC),
				Clouds: []clouds.Cloud{getCloud("BKN015//")},
				ParsingErrors: []error{
					&time.ParseError{Layout: "200601021504", Value: "2023053122o0", LayoutElem: "04", ValueElem: "o0", Message: ""},
				},
			}
			So(parseTrendData(input), ShouldResemble, expected)
		})

		Convey(`test AT-time`, func() {
			input = []string{"TEMPO", "AT1230", "18005MPS", "BKN003"}
			expected = &Trend{Type: TEMPO,
				Wind:          getWind("18005MPS"),
				AT:            time.Date(curYear, curMonth, curDay, 12, 30, 0, 0, time.UTC),
				Clouds:        []clouds.Cloud{getCloud("BKN003")},
				ParsingErrors: nil,
			}
			So(parseTrendData(input), ShouldResemble, expected)
		})

	})

}
