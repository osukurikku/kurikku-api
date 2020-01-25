package v2

import (
	"zxq.co/ripple/rippleapi/common"
)

type leaderboardResponse struct {
	common.ResponseBase
	Country11  []string `json:"country11"`
	Country500 []string `json:"country500"`
}

// LeaderboardGET gets the leaderboard.
func GetLeaderBoardCountries(md common.MethodData) common.CodeMessager {
	response := leaderboardResponse{}

	country11 := md.R.ZRevRange("hanayo:country_list", 0, 10).Val()
	country500 := md.R.ZRevRange("hanayo:country_list", 0, 499).Val()

	response.Country11 = country11
	response.Country500 = country500
	response.Code = 200
	return response
}
