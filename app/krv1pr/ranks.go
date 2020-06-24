package krv1pr

import (
	"strings"

	"gopkg.in/thehowl/go-osuapi.v1"
	"zxq.co/ripple/rippleapi/common"
	"zxq.co/x/getrank"
)

type Score struct {
	ID         int                  `json:"id"`
	BeatmapMD5 string               `json:"beatmap_md5"`
	Score      int64                `json:"score"`
	MaxCombo   int                  `json:"max_combo"`
	FullCombo  bool                 `json:"full_combo"`
	Mods       int                  `json:"mods"`
	Count300   int                  `json:"count_300"`
	Count100   int                  `json:"count_100"`
	Count50    int                  `json:"count_50"`
	CountGeki  int                  `json:"count_geki"`
	CountKatu  int                  `json:"count_katu"`
	CountMiss  int                  `json:"count_miss"`
	Time       common.UnixTimestamp `json:"time"`
	PlayMode   int                  `json:"play_mode"`
	Accuracy   float64              `json:"accuracy"`
	PP         float32              `json:"pp"`
	Rank       string               `json:"rank"`
	Completed  int                  `json:"completed"`
}

type UserRanksResponse struct {
	common.ResponseBase
	A    int `json:"a"`
	S    int `json:"s"`
	SH   int `json:"sh"`
	SS   int `json:"ss"`
	SSHD int `json:"sshd"`
}

func RanksGET(md common.MethodData) common.CodeMessager {
	iduser := md.Query("userid")
	//userid=1000&mode=0
	mode := md.Query("mode")

	var r UserRanksResponse

	rows, err := md.DB.Query(`
SELECT
	scores.id, scores.beatmap_md5, scores.score,
	scores.max_combo, scores.full_combo, scores.mods,
	scores.300_count, scores.100_count, scores.50_count,
	scores.gekis_count, scores.katus_count, scores.misses_count,
	scores.time, scores.play_mode, scores.accuracy, scores.pp,
	scores.completed
FROM scores
WHERE scores.userid = ? AND scores.completed = 3 AND scores.play_mode = ?`, iduser, mode)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}
	for rows.Next() {
		var (
			s Score
		)
		err := rows.Scan(
			&s.ID, &s.BeatmapMD5, &s.Score,
			&s.MaxCombo, &s.FullCombo, &s.Mods,
			&s.Count300, &s.Count100, &s.Count50,
			&s.CountGeki, &s.CountKatu, &s.CountMiss,
			&s.Time, &s.PlayMode, &s.Accuracy, &s.PP,
			&s.Completed,
		)
		if err != nil {
			md.Err(err)
			continue
		}
		s.Rank = strings.ToUpper(getrank.GetRank(
			osuapi.Mode(s.PlayMode),
			osuapi.Mods(s.Mods),
			s.Accuracy,
			s.Count300,
			s.Count100,
			s.Count50,
			s.CountMiss,
		))
		switch s.Rank {
		case "SSH":
			r.SSHD += 1
			break
		case "SSHD":
			r.SSHD += 1
			break
		case "SS":
			r.SS += 1
			break
		case "SH":
			r.SH += 1
			break
		case "S":
			r.S += 1
			break
		case "A":
			r.A += 1
			break
		}
	}
	r.Code = 200
	return r
}
