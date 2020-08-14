package v2

import (
	"strings"

	"zxq.co/ripple/rippleapi/common"
)

type nickAvailableResponse struct {
	common.ResponseBase
	Available bool
}

// GetAvailableUsername .
func GetAvailableUsername(md common.MethodData) common.CodeMessager {
	nick := md.Query("username")
	response := nickAvailableResponse{}

	safe := strings.ReplaceAll(strings.ToLower(nick), " ", "_")
	availableDB := ""
	md.DB.Get(&availableDB, "SELECT username_safe FROM users WHERE LOWER(username_safe) = LOWER(?)", safe)
	response.Available = bool(len(availableDB) == 0)
	response.Code = 200
	return response
}
