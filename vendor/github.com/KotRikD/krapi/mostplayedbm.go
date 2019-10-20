package krapi

import (
	"zxq.co/ripple/rippleapi/common"
)

type difficulty struct {
	STD   float64 `json:"std"`
	Taiko float64 `json:"taiko"`
	CTB   float64 `json:"ctb"`
	Mania float64 `json:"mania"`
}

type MostPlayedItem struct {
	BeatmapID          int                  `json:"beatmap_id"`
	SongName           string               `json:"song_name"`
	AR                 float32              `json:"ar"`
	OD                 float32              `json:"od"`
	Diff2              difficulty           `json:"difficulty2"` // fuck nyo
	BPM                int                  `json:"bpm"`
	PlayMode		   int					`json:"play_mode"`
	Count			   int					`json:"count"`
}

type beatmapsResponse struct {
	common.ResponseBase
	MostPlayed []MostPlayedItem `json:"beatmaps"`
}

const baseBeatmapSelect1 = `
SELECT 
	beatmaps.beatmap_id, beatmaps.song_name, beatmaps.ar, beatmaps.od, beatmaps.difficulty_std, 
	beatmaps.difficulty_taiko, beatmaps.difficulty_ctb, beatmaps.difficulty_mania, beatmaps.bpm, 
	scores.play_mode, COUNT(scores.beatmap_md5) AS cbm5 FROM scores 
RIGHT JOIN beatmaps
ON beatmaps.beatmap_md5 = scores.beatmap_md5
GROUP BY scores.beatmap_md5
ORDER BY cbm5 DESC
LIMIT 5;
`

func Beatmaps5GET(md common.MethodData) common.CodeMessager {
	var resp beatmapsResponse
	resp.Code = 200

	mode := md.Query("mode")
	uid := md.Query("uid")

	rows, err := md.DB.Query(baseBeatmapSelect1)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kotorikku instance admin and tell them to fix the API.")
	}
	for rows.Next() {
		var b MostPlayedItem
		err := rows.Scan(
			&b.BeatmapID,
			&b.SongName, &b.AR, &b.OD, &b.Diff2.STD, &b.Diff2.Taiko,
			&b.Diff2.CTB, &b.Diff2.Mania, &b.BPM,
			&b.PlayMode, &b.Count,
		)
		if err != nil {
			md.Err(err)
			continue
		}
		resp.Beatmaps = append(resp.MostPlayed, b)
	}

	return resp
}
