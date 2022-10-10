package krv1pr

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strconv"

	"zxq.co/ripple/rippleapi/common"
)

const (
	// keys are revoked, so don't try to search them from history :)
	PAYMENT_PRIVATEKEY            = ""
	PAYMENT_PRIVATEKEY_ADDITIONAL = ""
	PAYMENT_SHOP_ID               = ""
)

var (
	AVAILABLE_CURRENCIES = []string{"USD", "RUB", "EUR", "UAH"}
)

type SignatureBaseResult struct {
	common.ResponseBase
	PayID int    `json:"payid"`
	Sign  string `json:"signature"`
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func GenerateSignature(md common.MethodData) common.CodeMessager {
	if md.User.ID == 0 {
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	amount := md.Query("amount") // Сумма к оплате
	currency := md.Query("currency")
	if !stringInSlice(currency, AVAILABLE_CURRENCIES) {
		return common.SimpleResponse(500, "We're accepting only USD/RUB/EUR/UAH")
	}

	intForId, err := strconv.Atoi(md.Query("user_id"))
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	floatAmount, err := strconv.ParseFloat(amount, 2)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	lastPayID := 1
	err = md.DB.QueryRow(`select id from users_payment order by id desc limit 1`).Scan(&lastPayID)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	sign := getMD5Hash(fmt.Sprintf("%s:%.2f:%s:%d", PAYMENT_SHOP_ID, floatAmount, PAYMENT_PRIVATEKEY, lastPayID+1))
	md.DB.Exec(
		"insert into users_payment values (NULL, ?, ?, ?, 0)",
		intForId, floatAmount, currency,
	)

	signResult := SignatureBaseResult{}
	signResult.Code = 200
	signResult.Sign = sign
	signResult.PayID = lastPayID + 1
	return signResult
}

func CheckPayment(md common.MethodData) common.CodeMessager {
	arguments := md.Ctx.PostArgs()
	amount := string(arguments.Peek("credited"))
	pay_id := string(arguments.Peek("merchant_id"))
	user_id := string(arguments.Peek("custom_field[user_id]"))

	floatAmount, err := strconv.ParseFloat(amount, 2)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	intID, err := strconv.Atoi(pay_id)
	userIntID, err2 := strconv.Atoi(user_id)
	if err != nil || err2 != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	if len(amount) < 1 || len(pay_id) < 1 || len(user_id) < 1 {
		return common.SimpleResponse(500, "Some is strange wwww~~~")
	}

	signToCheck := getMD5Hash(fmt.Sprintf("%s:%s:%s:%d", PAYMENT_SHOP_ID, string(arguments.Peek("amount")), PAYMENT_PRIVATEKEY_ADDITIONAL, intID))
	if signToCheck != string(arguments.Peek("sign_2")) {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	md.DB.Exec(
		`update users set balance = balance + ? where id = ?`,
		int(floatAmount), userIntID,
	)
	md.DB.Exec(`update users_payment set completed = 1 where id = ?`,
		intID,
	)
	return common.SimpleResponse(200, "Good")
}
