package krv1pr

import (
	"crypto/md5"
	"encoding/hex"
	"math/rand"
	"strconv"
	"zxq.co/ripple/rippleapi/common"
)

var (
	PAYMENT_PRIVATEKEY = "TGOyJlHviIyhnOXz43i"
	PAYMENT_SHOP_ID = "4339"
)

type SignatureBaseResult struct {
	common.ResponseBase
	PayID string `json:"payid"`
	Sign string `json:"signature"`
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func GenerateSignature(md common.MethodData) common.CodeMessager {
	amount := md.Query("amount") // Сумма к оплате
	_, err := strconv.Atoi(amount)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	pay_id := strconv.Itoa(rand.Intn(99999999 - 1000) + 1000) // Номер счета
	currency := "RUB" // Валюта платежа

	sign := GetMD5Hash(currency+":"+amount+":"+PAYMENT_PRIVATEKEY+":"+PAYMENT_SHOP_ID+":"+pay_id)

	signResult := SignatureBaseResult{}
	signResult.Code = 200
	signResult.Sign = sign
	signResult.PayID = pay_id
	return signResult
}

func CheckPayment(md common.MethodData) common.CodeMessager {
	amount := md.Query("amount")
	pay_id := md.Query("pay_id")
	user_id := md.Query("field1")

	if len(amount) < 1 || len(pay_id) < 1 || len(user_id) < 1 {
		return common.SimpleResponse(500, "Some is strange wwww~~~")
	}

	signToCheck := GetMD5Hash(PAYMENT_SHOP_ID+":"+amount+":"+pay_id+":"+PAYMENT_PRIVATEKEY)

	if signToCheck != md.Query("sign") {
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	amountFloat, err := strconv.ParseFloat(amount, 2)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}
	amountInt := int(amountFloat)


	_ = md.DB.QueryRow("UPDATE users SET balance = balance+"+ strconv.Itoa(amountInt) + " WHERE id = " + user_id)
	return common.SimpleResponse(200, "OK")
}