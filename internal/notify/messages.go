package notify

import "fmt"

type texts struct {
	lowStatTitle  string
	lowStat       map[string]string
	friendTitle   string
	friendBody    string
	tradeTitle    string
	tradeStatuses map[string]string
}

var catalog = map[string]texts{
	"uk": {
		lowStatTitle: "%s потребує уваги",
		lowStat: map[string]string{
			"satiety":     "%s зголоднів — час нагодувати!",
			"mood":        "%s сумує — пограйтеся з ним.",
			"energy":      "%s дуже втомився — вкладіть його спати.",
			"cleanliness": "%s забруднився — час купатися.",
		},
		friendTitle: "Новий запит у друзі",
		friendBody:  "%s хоче додати вас у друзі.",
		tradeTitle:  "Обмін предметами",
		tradeStatuses: map[string]string{
			"pending":   "%s пропонує вам обмін.",
			"completed": "Обмін з %s завершено.",
			"declined":  "%s відхилив(-ла) ваш обмін.",
			"cancelled": "%s скасував(-ла) обмін.",
			"failed":    "Обмін з %s не вдався.",
			"expired":   "Пропозиція обміну з %s завершилася.",
		},
	},
	"en": {
		lowStatTitle: "%s needs you",
		lowStat: map[string]string{
			"satiety":     "%s is hungry — time to feed!",
			"mood":        "%s is sad — come and play.",
			"energy":      "%s is exhausted — put them to bed.",
			"cleanliness": "%s got dirty — bath time.",
		},
		friendTitle: "New friend request",
		friendBody:  "%s wants to be your friend.",
		tradeTitle:  "Item trade",
		tradeStatuses: map[string]string{
			"pending":   "%s offers you a trade.",
			"completed": "Trade with %s is complete.",
			"declined":  "%s declined your trade.",
			"cancelled": "%s cancelled the trade.",
			"failed":    "Trade with %s failed.",
			"expired":   "Trade offer with %s has expired.",
		},
	},
}

func textsFor(lang string) texts {
	if t, ok := catalog[lang]; ok {
		return t
	}
	return catalog["uk"]
}

func LowStat(userID, lang, petName, stat string) Notification {
	t := textsFor(lang)
	body, ok := t.lowStat[stat]
	if !ok {
		body = t.lowStat["satiety"]
	}
	return Notification{
		UserID: userID,
		Kind:   KindLowStat,
		Title:  fmt.Sprintf(t.lowStatTitle, petName),
		Body:   fmt.Sprintf(body, petName),
		URL:    "/",
		Tag:    "low-stat",
	}
}

func FriendRequest(userID, lang, fromNickname string) Notification {
	t := textsFor(lang)
	return Notification{
		UserID: userID,
		Kind:   KindFriendRequest,
		Title:  t.friendTitle,
		Body:   fmt.Sprintf(t.friendBody, fromNickname),
		URL:    "/?tab=friends",
		Tag:    "friend-request",
	}
}

func TradeChanged(userID, lang, otherNickname, status, offerID string) Notification {
	t := textsFor(lang)
	body, ok := t.tradeStatuses[status]
	if !ok {
		body = t.tradeStatuses["pending"]
	}
	return Notification{
		UserID: userID,
		Kind:   KindTrade,
		Title:  t.tradeTitle,
		Body:   fmt.Sprintf(body, otherNickname),
		URL:    "/?tab=trade",
		Tag:    "trade-" + offerID,
	}
}
