package main

import (
	"fmt"
	"log"
	"time"

	"github.com/minya/erc/erclib"
	"github.com/minya/ercInfoBot/model"
	"github.com/minya/telegram"
)

type notifier struct {
	sleepDuration time.Duration
	storage       model.UserStorage
	botToken      string
}

func (n notifier) Start() {
	go n.updateLoop()
}

func (n notifier) updateLoop() {
	for true {
		log.Printf("Update...\n")
		subsMap, err := n.storage.GetUsers()
		if err != nil {
			log.Printf("Error: %v\n", err)
		} else {
			for id, userInfo := range subsMap {
				log.Printf("[Update] Check user %v\n", id)
				for accountNum, sub := range userInfo.Subscriptions {
					ercClient := erclib.NewErcClientWithCredentials(userInfo.Login, userInfo.Password)
					accounts, err := ercClient.GetAccounts()
					if err != nil {
						log.Printf("WARN  No accounts")
						continue
					}
					account, err := findAccount(accounts, accountNum)
					if err != nil {
						log.Printf("WARN  No account %v among accounts", accountNum)
						continue
					}
					n.compareAndNotify(id, account, sub, userInfo, ercClient)
				}
			}
		}
		time.Sleep(n.sleepDuration)
	}
}

func (n notifier) compareAndNotify(
	userID int, account erclib.Account, sub model.SubscriptionInfo, userInfo model.UserInfo, ercClient erclib.ErcClient) {

	if sub.ChatID == 0 {
		log.Printf("[Update] User %v is not subscribed. Skip.\n", userID)
		return
	}
	balanceInfo, err := ercClient.GetBalanceInfo(account.Number, time.Now())
	if err != nil {
		log.Printf("[Update] Error: can't get balance for user %v\n", userID)
		return
	}
	newState := fmt.Sprintf("%v", balanceInfo)

	if sub.LastSeenState == "" {
		sub.LastSeenState = newState
		userInfo.Subscriptions[account.Number] = sub
		n.storage.SaveUser(userID, userInfo)
		log.Printf("[Update] Initial balance correction for user %v\n", userID)
	} else if sub.LastSeenState != newState {
		log.Printf("[Update] Balance changed for user %v\n", userID)
		sub.LastSeenState = newState
		userInfo.Subscriptions[account.Number] = sub
		n.storage.SaveUser(userID, userInfo)

		messageText := "Баланс обновился:\n" + formatBalance(account, balanceInfo)
		msg := telegram.ReplyMessage{
			ChatId:      sub.ChatID,
			Text:        messageText,
			ReplyMarkup: replyButtons(),
		}
		err = telegram.SendMessage(n.botToken, msg)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	} else {
		log.Printf("[Update] Balance hasn't been changed\n")
	}
}
