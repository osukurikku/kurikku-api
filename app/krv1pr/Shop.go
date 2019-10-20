package krv1pr

import (
	"strconv"
	"strings"
	"time"
	"zxq.co/ripple/rippleapi/common")

type Item struct {
	ID int
	Name string
	Description string
	Cost int
	Image string
	CanBuy bool
	Condition string
}

type User struct {
	UserID int
	Username string
	Privileges int
	DonorExpire int
	Balance int
}

type ShopItemsResponse struct {
	common.ResponseBase
	Balance int `json:"balance"`
	Items []Item `json:"items"`
}

func GetShopItems(md common.MethodData) common.CodeMessager {
	var arrayItems []Item
	var user User
	err := md.DB.QueryRow(`SELECT users.id, users.username, users.privileges, users.donor_expire, users.balance FROM users WHERE users.id = `+strconv.Itoa(md.ID())).Scan(&user.UserID, &user.Username, &user.Privileges, &user.DonorExpire, &user.Balance)
	rows, err := md.DB.Query(`SELECT * FROM shop_items`)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}
	for rows.Next() {
		var item Item
		err := rows.Scan(
			&item.ID, &item.Name, &item.Description, &item.Cost, &item.Image, &item.Condition,
		)
		if strings.HasPrefix(item.Condition, "unban") {
			// Check can user buy or not
			hasBan := common.UserPrivileges(user.Privileges)&common.UserPrivilegeNormal > 0
			if hasBan {
				item.CanBuy = false;
			} else {
				item.CanBuy = true;
			}
		} else {
			item.CanBuy = true
		}
		item.Condition = ""
		if err != nil {
			md.Err(err)
			return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
		}
		arrayItems = append(arrayItems, item)
	}

	resultResponse := ShopItemsResponse{}
	resultResponse.Code = 200
	resultResponse.Items = arrayItems
	resultResponse.Balance = user.Balance
	return resultResponse
}


func BuyShopItem(md common.MethodData) common.CodeMessager {
	itemID := md.Query("itemID")
	var item Item
	if len(itemID) < 1 {
		return common.SimpleResponse(500, "Please enter itemID")
	}
	err := md.DB.QueryRow(`SELECT * FROM shop_items WHERE id = `+itemID).Scan(&item.ID, &item.Name, &item.Description, &item.Cost, &item.Image, &item.Condition,)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	if len(item.Condition) < 1 {
		return common.SimpleResponse(400, "Item not found")
	}

	var user User
	//user == md.ID()
	err = md.DB.QueryRow(`SELECT users.id, users.username, users.privileges, users.donor_expire, users.balance FROM users WHERE users.id = `+strconv.Itoa(md.ID())).Scan(&user.UserID, &user.Username, &user.Privileges, &user.DonorExpire, &user.Balance)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	if user.Balance < item.Cost {
		return common.SimpleResponse(400, "Balance is enough")
	}

	resolvedCondition := strings.Split(item.Condition,";")
	if len(resolvedCondition) < 1 {
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kurikku instance admin and tell them to fix the API.")
	}

	//SIMPLE-CONDITION
	//donate;months
	//donate;1;

	//unban;
	//unban;

	switch resolvedCondition[0] {
	case "donate":
		// PART RIPPLE PHP CODE REWRITTEN IN GO
		// IDK HOW THIS WORK ;D THAT WHY I USE THAT
		intDonorMonths, _ := strconv.Atoi(resolvedCondition[1])
		hasDonor := common.UserPrivileges(user.Privileges)&common.UserPrivilegeDonor > 0
		start := 0
		if !hasDonor {
			start = int(time.Now().Unix())
		} else {
			start = user.DonorExpire;
			if start < int(time.Now().Unix()) {
				start = int(time.Now().Unix())
			}
		}

		unixExpire := start+((30*86400)*intDonorMonths);
		_ = md.DB.QueryRow("UPDATE users SET privileges = privileges | 4, donor_expire = "+strconv.Itoa(unixExpire)+" WHERE id = "+strconv.Itoa(user.UserID))

		var idBadge int
		_ = md.DB.QueryRow("SELECT id FROM badges WHERE name = 'Donator' OR name = 'Donor' LIMIT 1").Scan(idBadge);

		var idRecord int
		_ = md.DB.QueryRow("SELECT id FROM user_badges WHERE user = "+strconv.Itoa(user.UserID)+" AND badge = "+strconv.Itoa(idBadge)+" LIMIT 1").Scan(&idRecord);
		if (idRecord < 1) {
			_  = md.DB.QueryRow("INSERT INTO user_badges(user, badge) VALUES ("+strconv.Itoa(user.UserID)+", "+strconv.Itoa(idBadge)+");");
		}
		break;
	case "unban":
		isBan := common.UserPrivileges(user.Privileges)&common.UserPrivilegePublic > 0
		if isBan {
			return common.SimpleResponse(500, "Bro, you are not banned")
		}
		newPrivileges := user.Privileges | 2;
		newPrivileges |= 1

		_ = md.DB.QueryRow("UPDATE users SET privileges = "+strconv.Itoa(newPrivileges)+", ban_datetime = 0 WHERE id = "+strconv.Itoa(user.UserID)+" LIMIT 1")

		break;
	}

	_ = md.DB.QueryRow("UPDATE users SET balance = balance-"+strconv.Itoa(item.Cost)+" WHERE id = "+strconv.Itoa(user.UserID))
	return common.SimpleResponse(200, "OK")
}