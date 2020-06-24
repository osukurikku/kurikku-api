package krv1pr

import (
	"strconv"

	"zxq.co/ripple/rippleapi/common"
)

type StreamerObject struct {
	StreamID     int    `json:"stream_id" db:"id"`
	UserID       int    `json:"user_id" db:"user_id"`
	StreamerName string `json:"streamer_name" db:"streamer"`
	Title        string `json:"title" db:"name"`
	ViewerCount  int    `json:"viewer_count" db:"viewer_count"`
}

type StreamerResponse struct {
	common.ResponseBase
	Streamers []StreamerObject `json:"data"`
}

func AllStreamersGet(md common.MethodData) common.CodeMessager {
	var r StreamerResponse
	limitQuery := ""
	lim := md.Query("limit")
	if lim != "" {
		_, err := strconv.Atoi(lim)
		if err == nil {
			limitQuery = "LIMIT " + lim
		}
	}

	streamers := []StreamerObject{}
	md.DB.Select(&streamers, "select * from twitch_streams order by viewer_count desc "+limitQuery)

	r.Code = 200
	r.Streamers = streamers
	return r
}
