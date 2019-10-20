package krapi

import (
	"zxq.co/ripple/rippleapi/common"
	"fmt"
)

const baseBeatmapSelect2 = `
SELECT 
	beatmaps.beatmapset_id,
	beatmaps.beatmap_id, beatmaps.song_name, beatmaps.ar, beatmaps.od, beatmaps.difficulty_std, 
	beatmaps.difficulty_taiko, beatmaps.difficulty_ctb, beatmaps.difficulty_mania, beatmaps.bpm, 
	scores.play_mode, COUNT(scores.beatmap_md5) AS cbm5 FROM scores 
RIGHT JOIN beatmaps
ON beatmaps.beatmap_md5 = scores.beatmap_md5
WHERE scores.userid = %s AND scores.play_mode = %s
GROUP BY scores.beatmap_md5
ORDER BY cbm5 DESC
LIMIT 5 OFFSET %s;
`

func UsersMostPlayedBM(md common.MethodData) common.CodeMessager {
	var resp beatmapsResponse
	resp.Code = 200

	mode := md.Query("mode")
	uid := md.Query("uid")
	offset := md.Query("offset")

	rows, err := md.DB.Query(fmt.Sprintf(baseBeatmapSelect2, uid, mode, offset))
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
