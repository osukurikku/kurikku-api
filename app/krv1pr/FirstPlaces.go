package krv1pr

import (
	"strings"
	"fmt"
	"strconv"
	"zxq.co/ripple/rippleapi/common"
	"gopkg.in/thehowl/go-osuapi.v1"
	"zxq.co/x/getrank"
)

func whereClauseUser(md common.MethodData, tableName string) (*common.CodeMessager, string, interface{}) {
	switch {
	case md.Query("id") == "self":
		return nil, tableName + ".userid = ?", md.ID()
	case md.Query("id") != "":
		id, err := strconv.Atoi(md.Query("id"))
		if err != nil {
			a := common.SimpleResponse(400, "please pass a valid user ID")
			return &a, "", nil
		}
		return nil, tableName + ".userid = ?", id
	}
	a := common.SimpleResponse(400, "you need to pass either querystring parameters name or id")
	return &a, "", nil
}

func genModeClause(md common.MethodData) string {
	var modeClause string
	if md.Query("mode") != "" {
		m, err := strconv.Atoi(md.Query("mode"))
		if err == nil && m >= 0 && m <= 3 {
			modeClause = fmt.Sprintf("AND scores.play_mode = '%d'", m)
		}
	}
	return modeClause
}

func getMode(m string) string {
	switch m {
	case "1":
		return "taiko"
	case "2":
		return "ctb"
	case "3":
		return "mania"
	default:
		return "std"
	}
}

type difficulty struct {
	STD   float64 `json:"std"`
	Taiko float64 `json:"taiko"`
	CTB   float64 `json:"ctb"`
	Mania float64 `json:"mania"`
}

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
}

type userScore struct {
	Score
	Beatmap beatmap `json:"beatmap"`
}

type userScoresResponse struct {
	common.ResponseBase
	Scores []userScore `json:"scores"`
}

func scoresPuts(md common.MethodData, whereClause string, params ...interface{}) common.CodeMessager {
	rows, err := md.DB.Query(whereClause, params...)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}
	var scores []userScore
	for rows.Next() {
		var (
			us userScore
			b  beatmap
		)
		err = rows.Scan(
			&us.ID, &us.BeatmapMD5, &us.Score.Score,
			&us.MaxCombo, &us.FullCombo, &us.Mods,
			&us.Count300, &us.Count100, &us.Count50,
			&us.CountGeki, &us.CountKatu, &us.CountMiss,
			&us.Time, &us.PlayMode, &us.Accuracy, &us.PP,
			&us.Completed,

			&b.BeatmapID, &b.BeatmapsetID, &b.BeatmapMD5,
			&b.SongName, &b.AR, &b.OD, &b.Diff2.STD,
			&b.Diff2.Taiko, &b.Diff2.CTB, &b.Diff2.Mania,
			&b.MaxCombo, &b.HitLength, &b.Ranked,
			&b.RankedStatusFrozen, &b.LatestUpdate,
		)
		if err != nil {
			md.Err(err)
			return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
		}
		b.Difficulty = b.Diff2.STD
		us.Beatmap = b
		us.Rank = strings.ToUpper(getrank.GetRank(
			osuapi.Mode(us.PlayMode),
			osuapi.Mods(us.Mods),
			us.Accuracy,
			us.Count300,
			us.Count100,
			us.Count50,
			us.CountMiss,
		))
		scores = append(scores, us)
	}
	r := userScoresResponse{}
	r.Code = 200
	r.Scores = scores
	return r
}

func FirstScoresBestGET(md common.MethodData) common.CodeMessager {
	cm, wc, param := whereClauseUser(md, "fpscores")
	if cm != nil {
		return *cm
	}
	mc := genModeClause(md)
	return scoresPuts(md, fmt.Sprintf(`SELECT
        fpscores.id, fpscores.beatmap_md5, fpscores.score,
        fpscores.max_combo, fpscores.full_combo, fpscores.mods,
        fpscores.300_count, fpscores.100_count, fpscores.50_count,
        fpscores.gekis_count, fpscores.katus_count, fpscores.misses_count,
        fpscores.time, fpscores.play_mode, fpscores.accuracy, fpscores.pp,
        fpscores.completed,

        beatmaps.beatmap_id, beatmaps.beatmapset_id, beatmaps.beatmap_md5,
        beatmaps.song_name, beatmaps.ar, beatmaps.od, beatmaps.difficulty_std,
        beatmaps.difficulty_taiko, beatmaps.difficulty_ctb, beatmaps.difficulty_mania,
        beatmaps.max_combo, beatmaps.hit_length, beatmaps.ranked,
        beatmaps.ranked_status_freezed, beatmaps.latest_update
FROM ( SELECT * FROM (SELECT * FROM scores WHERE scores.completed = '3' %s GROUP BY scores.beatmap_md5, scores.score ORDER BY scores.score DESC ) AS fps1
GROUP BY fps1.beatmap_md5
ORDER BY fps1.pp DESC) AS fpscores
INNER JOIN beatmaps ON beatmaps.beatmap_md5 = fpscores.beatmap_md5
INNER JOIN users ON users.id = fpscores.userid
WHERE %s AND %s
%s
` ,mc, md.User.OnlyUserPublic(true), wc, common.Paginate(md.Query("p"), md.Query("l"), 100), ), param)
}