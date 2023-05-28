package wind

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/vasya4k/metar/conversion"
)

func TestParseWind(t *testing.T) {

	type testpair struct {
		input    string
		expected *Wind
		tokens   int
	}

	var windtests = []testpair{
		{"31005MPS", &Wind{310, 5, 0, false, 0, 0, false, false, false, MPS}, 1},
		{"31010KPH", &Wind{310, 10, 0, false, 0, 0, false, false, false, KPH}, 1},
		{"31010G15KMH", &Wind{310, 10, 15, false, 0, 0, false, false, false, KMH}, 1},
		{"VRB15MPS", &Wind{0, 15, 0, true, 0, 0, false, false, false, MPS}, 1},
		{"00000MPS", &Wind{0, 0, 0, false, 0, 0, false, false, false, MPS}, 1},
		{"240P49MPS", &Wind{240, 49, 0, false, 0, 0, true, false, false, MPS}, 1},
		{"04008G20MPS", &Wind{40, 8, 20, false, 0, 0, false, false, false, MPS}, 1},
		{"22003G08MPS 280V350", &Wind{220, 3, 8, false, 280, 350, false, false, false, MPS}, 2},
		{"14010KT", &Wind{140, 10, 0, false, 0, 0, false, false, false, KT}, 1},
		{"14010G15KT", &Wind{140, 10, 15, false, 0, 0, false, false, false, KT}, 1},
		{"/////KT", &Wind{0, 0, 0, false, 0, 0, false, true, true, KT}, 1},
		{"BKN020", &Wind{0, 0, 0, false, 0, 0, false, false, false, ""}, 0},
	}

	Convey("Wind parsing tests", t, func() {
		Convey("wind must parsed correctly", func() {
			for _, pair := range windtests {
				wnd := &Wind{}
				tokensused := wnd.ParseWind(pair.input)
				So(wnd, ShouldResemble, pair.expected)
				So(tokensused, ShouldEqual, pair.tokens)
			}
		})

		Convey("speed and gusts speed in meters per second must calculated correctly", func() {
			for _, pair := range windtests {
				wnd := &Wind{}
				wnd.ParseWind(pair.input)
				switch wnd.unit {
				case KPH, KMH:
					So(wnd.SpeedMps(), ShouldAlmostEqual, KphToMps(wnd.Speed), .5)
					So(wnd.GustsSpeedMps(), ShouldAlmostEqual, KphToMps(wnd.GustsSpeed), .5)
				case KT:
					So(wnd.SpeedMps(), ShouldAlmostEqual, KtsToMps(float64(wnd.Speed)), .5)
					So(wnd.GustsSpeedMps(), ShouldAlmostEqual, KtsToMps(float64(wnd.GustsSpeed)), .5)
				case MPS:
					So(wnd.SpeedMps(), ShouldEqual, wnd.Speed)
					So(wnd.GustsSpeedMps(), ShouldEqual, wnd.GustsSpeed)
				}
			}
		})

		Convey("speed and gusts speed in knots must calculated correctly", func() {
			for _, pair := range windtests {
				wnd := &Wind{}
				wnd.ParseWind(pair.input)
				switch wnd.unit {
				case MPS:
					So(wnd.SpeedKt(), ShouldAlmostEqual, MpsToKts(float64(wnd.Speed)), .5)
					So(wnd.GustsSpeedKt(), ShouldAlmostEqual, MpsToKts(float64(wnd.GustsSpeed)), .5)
				case KPH, KMH:
					So(wnd.SpeedKt(), ShouldAlmostEqual, KphToKts(wnd.Speed), .5)
					So(wnd.GustsSpeedKt(), ShouldAlmostEqual, KphToKts(wnd.GustsSpeed), .5)
				case KT:
					So(wnd.SpeedKt(), ShouldEqual, wnd.Speed)
					So(wnd.GustsSpeedKt(), ShouldEqual, wnd.GustsSpeed)
				}

			}
		})

	})
}
