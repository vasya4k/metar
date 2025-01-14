package metar

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/vasya4k/metar/wind"
)

// Remark - Additional information not included in the main message
type Remark struct {
	WindOnRWY []WindOnRWY `json:"wind_on_rwy"`
	QBB       int         `json:"qbb"`      // Cloud base in meters
	МТOBSC    bool        `json:"мтobsc"`   // Mountains obscured
	MASTOBSC  bool        `json:"mastobsc"` // Mast obscured
	OBSTOBSC  bool        `json:"obstobsc"` // Obstacle obscured
	QFE       int         `json:"qfe"`      // Q-code Field Elevation (mmHg)
}

// WindOnRWY - surface wind observations on the runways
type WindOnRWY struct {
	Runway string `json:"runway"`
	wind.Wind
}

func parseRemarks(tokens []string) *Remark {
	if len(tokens) == 0 {
		return nil
	}
	RMK := &Remark{}
	for count := 0; count < len(tokens); {
		// Wind value on runway. Not documented, but used in URSS and UHMA
		regex := regexp.MustCompile(`^(R\d{2}[LCR]?)/((\d{3})?(VRB)?(\d{2})?(G\d\d)?(MPS|KT))`)
		matches := regex.FindStringSubmatch(tokens[count])
		if len(matches) != 0 && matches[0] != "" {
			wnd := &WindOnRWY{}
			wnd.Runway = matches[1][1:]
			input := matches[2]
			if count < len(tokens)-1 {
				input += tokens[count+1]
			}
			count += wnd.ParseWind(input)
			RMK.WindOnRWY = append(RMK.WindOnRWY, *wnd)
		}

		if count < len(tokens) && strings.HasPrefix(tokens[count], "QBB") {
			RMK.QBB, _ = strconv.Atoi(tokens[count][3:])
			count++
		}

		for count < len(tokens)-1 && tokens[count+1] == "OBSC" {
			switch tokens[count] {
			case "MT":
				RMK.МТOBSC = true
			case "MAST":
				RMK.MASTOBSC = true
			case "OBST":
				RMK.OBSTOBSC = true
			}
			count += 2
		}
		// may be QFE767/1022 (mmHg/hPa)
		if count < len(tokens) && strings.HasPrefix(tokens[count], "QFE") {
			RMK.QFE, _ = strconv.Atoi(tokens[count][3:6])
			count++
		}
		count++
	}
	return RMK
}
