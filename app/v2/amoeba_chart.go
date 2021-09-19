package v2

import (
	"strings"

	"time"

	"gopkg.in/thehowl/go-osuapi.v1"
	"zxq.co/ripple/rippleapi/common"
	"zxq.co/x/getrank"
)

type amoebaChartItem struct {
	BeatmapId string `json:"_bid"`
	Grade     string `json:"_grade"`
	When      string `json:"_when"`
	Name      string `json:"name"`
	X         int64  `json:"x"`
	Y         int64  `json:"y"`
}

type amoebaChartResponse struct {
	common.ResponseBase
	Data []amoebaChartItem `json:"data"`
}

// AmoebaScoresChart
func AmoebaScoresChart(md common.MethodData) common.CodeMessager {
	iduser := md.Query("userid")
	//userid=1000&mode=0
	mode := md.Query("mode")
	response := amoebaChartResponse{}

	rows, err := md.DB.Query(`
SELECT
	scores.time, scores.pp, beatmaps.song_name, beatmaps.beatmap_id,

	scores.play_mode, scores.mods, scores.accuracy, scores.300_count, scores.100_count, scores.50_count, scores.misses_count

FROM scores
RIGHT JOIN beatmaps ON beatmaps.beatmap_md5 = scores.beatmap_md5
WHERE scores.userid = ? AND scores.completed = 3 AND scores.play_mode = ?
ORDER BY scores.pp DESC
LIMIT 100`, iduser, mode)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}
	for rows.Next() {
		var (
			chartItem amoebaChartItem

			playMode  int
			mods      int
			accuracy  float64
			count300  int
			count100  int
			count50   int
			countMiss int
		)

		err := rows.Scan(&chartItem.X, &chartItem.Y, &chartItem.Name, &chartItem.BeatmapId, &playMode, &mods, &accuracy, &count300, &count100, &count50, &countMiss)
		if err != nil {
			md.Err(err)
			continue
		}

		chartItem.Grade = strings.ToUpper(getrank.GetRank(
			osuapi.Mode(playMode),
			osuapi.Mods(mods),
			accuracy,
			count300,
			count100,
			count50,
			countMiss,
		))
		chartItem.When = time.Unix(chartItem.X, 0).Format(time.RFC3339)
		chartItem.X = chartItem.X * 1000

		response.Data = append(response.Data, chartItem)
	}

	response.Code = 200
	return response
}
