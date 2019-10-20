package krapi

import (
	"zxq.co/ripple/rippleapi/common"
)

type beatmap struct {
	BeatmapID          int                  `json:"beatmap_id"`
	BeatmapsetID       int                  `json:"beatmapset_id"`
	BeatmapMD5         string               `json:"beatmap_md5"`
	SongName           string               `json:"song_name"`
	AR                 float32              `json:"ar"`
	OD                 float32              `json:"od"`
	Difficulty         float64              `json:"difficulty"`
	Diff2              difficulty           `json:"difficulty2"` // fuck nyo
	MaxCombo           int                  `json:"max_combo"`
	HitLength          int                  `json:"hit_length"`
	Ranked             int                  `json:"ranked"`
	RankedStatusFrozen int                  `json:"ranked_status_frozen"`
	LatestUpdate       common.UnixTimestamp `json:"latest_update"`
	PlayCount          int                  `json:"playcount"`
	BPM                int                  `json:"bpm"`
}

type difficulty struct {
	STD   float64 `json:"std"`
	Taiko float64 `json:"taiko"`
	CTB   float64 `json:"ctb"`
	Mania float64 `json:"mania"`
}

type MostPlayedItem struct {
	BeatmapsetID	   int			`json:"beatmapset_id"`
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
	beatmaps.beatmapset_id,
	beatmaps.beatmap_id, beatmaps.song_name, beatmaps.ar, beatmaps.od, beatmaps.difficulty_std, 
	beatmaps.difficulty_taiko, beatmaps.difficulty_ctb, beatmaps.difficulty_mania, beatmaps.bpm, 
	scores.play_mode, COUNT(scores.beatmap_md5) AS cbm5 FROM scores 
RIGHT JOIN beatmaps
ON beatmaps.beatmap_md5 = scores.beatmap_md5
WHERE scores.play_mode = 0
GROUP BY scores.beatmap_md5
ORDER BY cbm5 DESC
LIMIT 5;
`

func Beatmaps5GET(md common.MethodData) common.CodeMessager {
	var resp beatmapsResponse
	resp.Code = 200

	rows, err := md.DB.Query(baseBeatmapSelect1)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kotorikku instance admin and tell them to fix the API.")
	}
	for rows.Next() {
		var b MostPlayedItem
		err := rows.Scan(
			&b.BeatmapsetID, &b.BeatmapID,
			&b.SongName, &b.AR, &b.OD, &b.Diff2.STD, &b.Diff2.Taiko,
			&b.Diff2.CTB, &b.Diff2.Mania, &b.BPM,
			&b.PlayMode, &b.Count,
		)
		if err != nil {
			md.Err(err)
			continue
		}
		resp.MostPlayed = append(resp.MostPlayed, b)
	}

	return resp
}
