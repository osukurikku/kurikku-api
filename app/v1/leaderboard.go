package v1

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"

	redis "gopkg.in/redis.v5"

	"math"

	"zxq.co/ripple/ocl"
	"zxq.co/ripple/rippleapi/common"
)

type leaderboardUser struct {
	userData
	ChosenMode    modeData `json:"chosen_mode"`
	PlayStyle     int      `json:"play_style"`
	FavouriteMode int      `json:"favourite_mode"`
}

type leaderboardResponse struct {
	common.ResponseBase
	Users   []leaderboardUser `json:"users"`
	MaxPage int               `json:"max_page"`
}

const lbUserQuery = `
SELECT
	users.id, users.username, users.register_datetime, users.privileges, users.latest_activity,

	users_stats.username_aka, users_stats.country,
	users_stats.play_style, users_stats.favourite_mode,

	users_stats.ranked_score_%[1]s, users_stats.total_score_%[1]s, users_stats.playcount_%[1]s,
	users_stats.replays_watched_%[1]s, users_stats.total_hits_%[1]s,
	users_stats.avg_accuracy_%[1]s, users_stats.pp_%[1]s
FROM users
INNER JOIN users_stats ON users_stats.id = users.id
WHERE users.id IN (?)
`

const slbUserQuery = `
SELECT
	users.id, users.username, users_stats.country,

	users_stats.ranked_score_std, users_stats.total_score_std, users_stats.playcount_std,
	users_stats.avg_accuracy_std, users_stats.pp_std, 

	users_stats.skill_stamina, users_stats.skill_tenacity, users_stats.skill_agility, 
	users_stats.skill_precision, users_stats.skill_memory, users_stats.skill_accuracy, users_stats.skill_reaction
FROM users
INNER JOIN users_stats ON users_stats.id = users.id
WHERE users.id IN (?)
`

// LeaderboardGET gets the leaderboard.
func LeaderboardGET(md common.MethodData) common.CodeMessager {
	m := getMode(md.Query("mode"))

	// md.Query.Country
	p := common.Int(md.Query("p")) - 1
	if p < 0 {
		p = 0
	}
	l := common.InString(1, md.Query("l"), 500, 50)

	key := "ripple:leaderboard:" + m
	if md.Query("country") != "" {
		key += ":" + md.Query("country")
	}

	results, err := md.R.ZRevRange(key, int64(p*l), int64(p*l+l-1)).Result()
	if err != nil {
		md.Err(err)
		return Err500
	}

	var resp leaderboardResponse
	resp.Code = 200

	var (
		maxCount int64
		maxPage  int
	)
	maxCount, _ = md.R.ZCount(key, "-inf", "+inf").Result()
	if maxCount <= 0 {
		maxPage = 1
	}
	if math.Mod(float64(maxCount)/float64(l), 1) == 0 {
		maxPage = int(int(maxCount) / l)
	} else {
		maxPage = int(int(maxCount)/l) + 1
	}

	resp.MaxPage = maxPage

	if len(results) == 0 {
		return resp
	}

	query := fmt.Sprintf(lbUserQuery+` ORDER BY users_stats.pp_%[1]s DESC, users_stats.ranked_score_%[1]s DESC`, m)
	query, params, _ := sqlx.In(query, results)
	rows, err := md.DB.Query(query, params...)
	if err != nil {
		md.Err(err)
		return Err500
	}
	for rows.Next() {
		var u leaderboardUser
		err := rows.Scan(
			&u.ID, &u.Username, &u.RegisteredOn, &u.Privileges, &u.LatestActivity,

			&u.UsernameAKA, &u.Country, &u.PlayStyle, &u.FavouriteMode,

			&u.ChosenMode.RankedScore, &u.ChosenMode.TotalScore, &u.ChosenMode.PlayCount,
			&u.ChosenMode.ReplaysWatched, &u.ChosenMode.TotalHits,
			&u.ChosenMode.Accuracy, &u.ChosenMode.PP,
		)
		if err != nil {
			md.Err(err)
			continue
		}
		u.ChosenMode.Level = ocl.GetLevelPrecise(int64(u.ChosenMode.TotalScore))
		if i := leaderboardPosition(md.R, m, u.ID); i != nil {
			u.ChosenMode.GlobalLeaderboardRank = i
		}
		if i := countryPosition(md.R, m, u.ID, u.Country); i != nil {
			u.ChosenMode.CountryLeaderboardRank = i
		}
		resp.Users = append(resp.Users, u)
	}
	return resp
}

type skillsData struct {
	Stamina   int `json:"stamina"`
	Tenacity  int `json:"tenacity"`
	Agility   int `json:"agility"`
	Precision int `json:"precision"`
	Memory    int `json:"memory"`
	Accuracy  int `json:"accuracy"`
	Reaction  int `json:"reaction"`
}

type skillLeaderboardUser struct {
	ID          int        `json:"id"`
	Username    string     `json:"username"`
	Country     string     `json:"country"`
	GlobalRank  int        `json:"global_rank"`
	RankedScore int        `json:"ranked_score"`
	TotalScore  int        `json:"total_score"`
	PlayCount   int        `json:"playcount"`
	Accuracy    float64    `json:"accuracy"`
	PP          int        `json:"pp"`
	Skills      skillsData `json:"skills"`
}

type skillLeaderboardResponse struct {
	common.ResponseBase
	Users   []skillLeaderboardUser `json:"users"`
	MaxPage int                    `json:"max_page"`
}

// SkillsLeaderboardGET gets the skills leaderboard.
func SkillsLeaderboardGET(md common.MethodData) common.CodeMessager {
	// md.Query.Country
	p := common.Int(md.Query("p")) - 1
	if p < 0 {
		p = 0
	}
	l := common.InString(1, md.Query("l"), 500, 50)

	key := "ripple:leaderboard:std"
	results, err := md.R.ZRevRange(key, 0, -1).Result()
	if err != nil {
		md.Err(err)
		return Err500
	}

	var resp skillLeaderboardResponse
	resp.Code = 200

	var (
		maxCount int64
		maxPage  int
	)
	maxCount, _ = md.R.ZCount(key, "-inf", "+inf").Result()
	if maxCount <= 0 {
		maxPage = 1
	}
	if math.Mod(float64(maxCount)/float64(l), 1) == 0 {
		maxPage = int(int(maxCount) / l)
	} else {
		maxPage = int(int(maxCount)/l) + 1
	}

	resp.MaxPage = maxPage

	if len(results) == 0 {
		return resp
	}

	order := "pp_std"
	if md.Query("by") != "" {
		switch md.Query("by") {
		case "stamina":
			order = "skill_stamina"
			break
		case "tenacity":
			order = "skill_tenacity"
			break
		case "agility":
			order = "skill_agility"
			break
		case "precision":
			order = "skill_precision"
			break
		case "memory":
			order = "skill_memory"
			break
		case "accuracy":
			order = "skill_accuracy"
			break
		case "reaction":
			order = "skill_reaction"
			break
		default:
			break
		}
	}

	typeSort := "DESC"
	if md.Query("order") != "" {
		switch md.Query("order") {
		case "asc":
			typeSort = "ASC"
			break
		case "desc":
			typeSort = "DESC"
			break
		default:
			break
		}
	}

	query := fmt.Sprintf(slbUserQuery+` ORDER BY %s %s LIMIT %d OFFSET %d`, order, typeSort, l, p*l)
	query, params, _ := sqlx.In(query, results)
	rows, err := md.DB.Query(query, params...)
	if err != nil {
		md.Err(err)
		return Err500
	}

	for rows.Next() {
		var u skillLeaderboardUser
		err := rows.Scan(
			&u.ID, &u.Username, &u.Country,

			&u.RankedScore, &u.TotalScore, &u.PlayCount,
			&u.Accuracy, &u.PP,

			&u.Skills.Stamina, &u.Skills.Tenacity, &u.Skills.Agility,
			&u.Skills.Precision, &u.Skills.Memory, &u.Skills.Accuracy, &u.Skills.Reaction,
		)
		if err != nil {
			md.Err(err)
			continue
		}
		if i := leaderboardPosition(md.R, "std", u.ID); i != nil {
			u.GlobalRank = *i
		}
		resp.Users = append(resp.Users, u)
	}
	return resp
}

func leaderboardPosition(r *redis.Client, mode string, user int) *int {
	return _position(r, "ripple:leaderboard:"+mode, user)
}

func countryPosition(r *redis.Client, mode string, user int, country string) *int {
	return _position(r, "ripple:leaderboard:"+mode+":"+strings.ToLower(country), user)
}

func _position(r *redis.Client, key string, user int) *int {
	res := r.ZRevRank(key, strconv.Itoa(user))
	if res.Err() == redis.Nil {
		return nil
	}
	x := int(res.Val()) + 1
	return &x
}
