package krv1pr

import (
	"strings"
	"zxq.co/ripple/rippleapi/common"
)

type BackgroundsResponse struct {
	common.ResponseBase
	Backgrounds	[]string	`json:"backgrounds"`
}

func GetBGs(md common.MethodData) common.CodeMessager {
	response := BackgroundsResponse{}
	
	var rawString string
	err := md.DB.QueryRow(`SELECT custom_bgs FROM user_kotrik_settings WHERE uid = ?`, md.ID()).Scan(&rawString)
	if err != nil {
		md.DB.Exec(`INSERT INTO user_kotrik_settings (uid) VALUES (?)`, md.ID())
		
		err = md.DB.QueryRow(`SELECT custom_bgs FROM user_kotrik_settings WHERE uid = ?`, md.ID()).Scan(&rawString)		
		if err != nil {
			md.Err(err)
			return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
		}
	}
	splitted := strings.Split(rawString, "|")
	if splitted[0] != "" {
		response.Backgrounds = splitted
	} else {
		response.Backgrounds = []string{}
	}

	response.Code = 200
	return response
}

func UpdateBGs(md common.MethodData) common.CodeMessager {
	//BackgroundsResponse
	allowedTypes := []string{".png", ".jpeg", ".jpg", ".bmp"}
	response := BackgroundsResponse{}
	var bgsInput struct {
		Backgrounds []string `json:"bgs"`
	}
	md.Unmarshal(&bgsInput)
	if bgsInput.Backgrounds == nil {
		return common.SimpleResponse(500, "Missed field bgs")
	}
	unpackedBackgrounds := bgsInput.Backgrounds

	var user User
	//user == md.ID()
	err := md.DB.QueryRow(`SELECT users.id, users.username, users.privileges, users.donor_expire FROM users WHERE users.id = ?`, md.ID()).Scan(&user.UserID, &user.Username, &user.Privileges, &user.DonorExpire)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	var rawString string
	err = md.DB.QueryRow(`SELECT custom_bgs FROM user_kotrik_settings WHERE uid = ?`, md.ID()).Scan(&rawString)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}
	splitted := strings.Split(rawString, "|")

	// donater - 8, standard - 4
	maxRecords := 4
	isDonor := common.UserPrivileges(user.Privileges)&common.UserPrivilegeDonor > 0
	if isDonor || len(splitted) >= 8 {
		maxRecords = 8
	}
	if len(unpackedBackgrounds) > maxRecords {
		unpackedBackgrounds = unpackedBackgrounds[:maxRecords]
	}

	for _, s := range unpackedBackgrounds {
		isAllowed := false
		for _, s2 := range allowedTypes {
			if strings.HasSuffix(s, s2) {
				isAllowed = true
			}
		}
		if !isAllowed {
			return common.SimpleResponse(500, "One of backgrounds is unsupported!")
		}
	}

	unsplitted := strings.Join(unpackedBackgrounds, "|")
	md.DB.Exec(`UPDATE user_kotrik_settings SET custom_bgs = ? WHERE uid = ?`, unsplitted, md.ID())

	response.Backgrounds = unpackedBackgrounds
	response.Code = 200
	return response
}

