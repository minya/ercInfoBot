package main

import (
	"fmt"
	"github.com/minya/erc/erclib"
	"github.com/minya/goutils/config"
	"github.com/minya/telegram"
	"log"
	"time"
)

var settings BotSettings

const strNotifySleepDuration = "4h"

func handle(upd telegram.Update) interface{} {
	log.Printf("Update: %v\n", upd)
	userId := upd.Message.From.Id

	var userInfo UserInfo
	userInfoPath := fmt.Sprintf(".ercInfoBot/users/%v.json", userId)
	userInfoErr := config.UnmarshalJson(&userInfo, userInfoPath)

	if nil != userInfoErr {
		log.Printf("Login not found for user %v. Creating stub.\n", userId)
		config.MarshalJson(userInfo, userInfoPath)
	} else {
		log.Printf("Login for user %v found: %v\n", userId, userInfo.Login)
	}

	cmd, cmdParseErr := ParseCommand(upd.Message.Text)

	if cmdParseErr != nil {
		log.Printf("Error parse command: %v\n", cmdParseErr)
		return help(upd)
	}

	log.Printf("Process command: %v\n", cmd.Command)

	switch cmd.Command {
	case "/reg":
		return register(upd, cmd.Args[0], cmd.Args[1], cmd.Args[2])
	case "/notify":
		return setUpNotification(upd, userInfo, cmd.Args[0] == "on")
	case "/get":
		return get(upd, userInfo)
	case "/receipt":
		return receipt(upd, userInfo)
	default:
		log.Printf("Unknown command: %v\n", cmd.Command)
		return help(upd)
	}
}

func register(upd telegram.Update, login string, password string, account string) interface{} {
	balanceInfo, errBalanceInfo := getBalanceInfo(login, password, account)
	if errBalanceInfo != nil {
		return telegram.ReplyMessage{
			ChatId: upd.Message.Chat.Id,
			Text:   "Wrong login/password. Please, register: /reg <login> <password> <account>",
		}
	}

	var userInfo UserInfo
	userInfo.Login = login
	userInfo.Password = password
	userInfo.Account = account

	userInfoPath := fmt.Sprintf(".ercInfoBot/users/%v.json", upd.Message.From.Id)
	config.MarshalJson(userInfo, userInfoPath)

	return telegram.ReplyMessage{
		ChatId: upd.Message.Chat.Id,
		Text:   fmt.Sprintf("You have been registered. Ur balance is: %v", balanceInfo.AtTheEnd.Total),
		ReplyMarkup: telegram.InlineKeyboardMarkup{
			Keyboard: [][]telegram.KeyboardButton{
				{
					telegram.KeyboardButton{Text: "/get"},
				},
			},
		},
	}
}

func get(upd telegram.Update, userInfo UserInfo) interface{} {
	if userInfo.Login == "" {
		return telegram.ReplyMessage{
			ChatId: upd.Message.Chat.Id,
			Text:   "Please, register: /reg <login> <password> <account>",
		}
	}

	balanceInfo, _ := getBalanceInfo(userInfo.Login, userInfo.Password, userInfo.Account)
	return telegram.ReplyMessage{
		ChatId: upd.Message.Chat.Id,
		Text:   formatBalance(balanceInfo),
		ReplyMarkup: telegram.InlineKeyboardMarkup{
			Keyboard: [][]telegram.KeyboardButton{
				{
					telegram.KeyboardButton{Text: "/get"},
				},
			},
		},
	}
}

func receipt(upd telegram.Update, userInfo UserInfo) interface{} {
	receipt, _ := erclib.GetReceipt(userInfo.Login, userInfo.Password, userInfo.Account)
	return telegram.ReplyDocument{
		ChatId:  upd.Message.Chat.Id,
		Caption: "Квитанция",
		InputFile: telegram.InputFile{
			Content:  receipt,
			FileName: "receipt.pdf",
		},
	}
}

func setUpNotification(upd telegram.Update, userInfo UserInfo, turnOn bool) telegram.ReplyMessage {
	return telegram.ReplyMessage{
		ChatId: upd.Message.Chat.Id,
		Text:   "Not implemented yet",
	}
}

func getBalanceInfo(login string, password string, accNumber string) (erclib.BalanceInfo, error) {
	bal, err := erclib.GetBalanceInfo(login, password, accNumber, time.Now())
	if nil != err {
		return erclib.BalanceInfo{}, err
	}
	return bal, nil
}

func formatBalance(balance erclib.BalanceInfo) string {
	return fmt.Sprintf(
		"%v\nНачислено: %v\nПоступления: %v\nИтого: %v",
		balance.Month,
		formatBalanceRow(balance.Credit),
		formatBalanceRow(balance.Debit),
		formatBalanceRow(balance.AtTheEnd))
}

func formatBalanceRow(row erclib.Details) string {
	return fmt.Sprintf(
		"%v (УК: %v, Капремонт: %v)",
		row.Total, row.CompanyPart, row.RepairPart)
}

func help(upd telegram.Update) telegram.ReplyMessage {
	helpMsg :=
		"/reg $login $password $account -- register your account\n" +
			"/notify $on|$off -- set up notifications"

	return telegram.ReplyMessage{
		ChatId: upd.Message.Chat.Id,
		Text:   helpMsg,
	}
}

func updateLoop(sleepDuration time.Duration) {
	for true {
		log.Printf("Notify\n")

		time.Sleep(sleepDuration)
	}
}

func main() {
	errCfg := config.UnmarshalJson(&settings, ".ercInfoBot/settings.json")
	if nil != errCfg {
		log.Printf("Unable to get config: %v\n", errCfg)
		return
	}
	duration, errParseDuration := time.ParseDuration(strNotifySleepDuration)
	if errParseDuration != nil {
		log.Fatalf("Unable to parse duration from '%v' \n", strNotifySleepDuration)
	}
	go updateLoop(duration)
	listenErr := telegram.StartListen(settings.Id, 8080, handle)
	if nil != listenErr {
		log.Printf("Unable to start listen: %v\n", listenErr)
	}

}

type UserInfo struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Account  string `json:"account"`
	Notify   bool   `json:"notify"`
}

type BotSettings struct {
	Id string
}
