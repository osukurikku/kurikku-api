package krapi

//

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/thehowl/go-osuapi.v1"
	"zxq.co/ripple/rippleapi/common"
	"zxq.co/x/getrank"
)

type Score struct {
	ID        int     `json:"id"`
	Score     int64   `json:"score"`
	Mods      int     `json:"mods"`
	Count300  int     `json:"count_300"`
	Count100  int     `json:"count_100"`
	Count50   int     `json:"count_50"`
	CountMiss int     `json:"count_miss"`
	PlayMode  int     `json:"play_mode"`
	Accuracy  float64 `json:"accuracy"`
	Rank      string  `json:"rank"`
}

type LogSimple struct {
	SongName  string               `json:"song_name"`
	LogBody   string               `json:"body"`
	Time      common.UnixTimestamp `json:"time"`
	ScoreID   int                  `json:"scoreid"`
	BeatmapID int                  `json:"beatmap_id"`
	Rank      string               `json:"rank"`
}

type DataGraph struct {
	Day   string `json:"day"`
	Value int    `json:"value"`
}

type PPGraph struct {
	MinLimit int         `json:"minLimit"`
	MaxLimit int         `json:"maxLimit"`
	Data     []DataGraph `json:"data"`
}

type Massive struct {
	common.ResponseBase
	Log    []LogSimple `json:"logs"`
	Graphs PPGraph     `json:"ppGraph"`
}

func ToNearThousand(value int) int {
	return (value + 500) / 1000 * 1000
}

func reverse(graphsRaw []DataGraph) []DataGraph {
	graphs := graphsRaw
	for i, j := 0, len(graphs)-1; i < j; i, j = i+1, j-1 {
		graphs[i], graphs[j] = graphs[j], graphs[i]
	}

	return graphs
}

//-1500
//+3000

func LogsGET(md common.MethodData) common.CodeMessager {
	id := md.Query("userid")
	mode := md.Query("mode")
	//Getting Logs
	results, err := md.DB.Query(`SELECT 
beatmaps.song_name, 
users_logs.log, users_logs.time, users_logs.scoreid, 
beatmaps.beatmap_id,
scores.play_mode, scores.mods, scores.accuracy, scores.300_count, scores.100_count, scores.50_count, scores.misses_count
FROM users_logs 
LEFT JOIN beatmaps ON (beatmaps.beatmap_md5 = users_logs.beatmap_md5)
INNER JOIN scores ON scores.id = users_logs.scoreid
WHERE user = ? 
AND users_logs.game_mode = ? 
AND users_logs.time > ?
ORDER BY users_logs.time  
DESC LIMIT 5
`, id, mode, int(time.Now().Unix())-2592000)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	var response Massive
	var logs []LogSimple

	defer results.Close()
	for results.Next() {
		var ls LogSimple
		var s Score
		results.Scan(
			&ls.SongName, &ls.LogBody, &ls.Time, &ls.ScoreID, &ls.BeatmapID,
			&s.PlayMode, &s.Mods, &s.Accuracy, &s.Count300, &s.Count100, &s.Count50, &s.CountMiss,
		)

		ls.Rank = strings.ToUpper(getrank.GetRank(
			osuapi.Mode(s.PlayMode),
			osuapi.Mods(s.Mods),
			s.Accuracy,
			s.Count300,
			s.Count100,
			s.Count50,
			s.CountMiss,
		))

		logs = append(logs, ls)
	}
	if err := results.Err(); err != nil {
		md.Err(err)
	}

	//Getting Graphs
	resultsG, err2 := md.DB.Query(fmt.Sprintf(`SELECT day, pp FROM user_ticks_graph 
WHERE user_id = %s AND type = 'pp_graph' AND mode = %s ORDER BY day DESC LIMIT 30`, id, mode))
	if err2 != nil {
		md.Err(err2)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	var ppgraph PPGraph
	minPPValue := 0
	maxPPValue := 0

	for resultsG.Next() {
		var dataInfo DataGraph

		resultsG.Scan(
			&dataInfo.Day, &dataInfo.Value,
		)

		if ToNearThousand(dataInfo.Value) > ToNearThousand(maxPPValue) {
			maxPPValue = ToNearThousand(dataInfo.Value)
		}

		if minPPValue == 0 {
			minPPValue = dataInfo.Value
		} else if dataInfo.Value < minPPValue {
			minPPValue = dataInfo.Value
		}

		ppgraph.Data = append(ppgraph.Data, dataInfo)
		ppgraph.Data = reverse(ppgraph.Data)
	}

	response.Log = logs
	response.Graphs = ppgraph
	response.Graphs.MaxLimit = maxPPValue + 500
	response.Graphs.MinLimit = ToNearThousand(minPPValue - 500)
	if response.Graphs.MinLimit <= 0 {
		response.Graphs.MinLimit = 1
	}
	if response.Graphs.MaxLimit <= 0 {
		response.Graphs.MaxLimit = 1
	}
	response.Code = 200
	return response
}
