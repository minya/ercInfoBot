package main

import (
	"fmt"
	"github.com/minya/erc/erclib"
	"github.com/minya/goutils/config"
	"github.com/minya/telegram"
	"log"
	"regexp"
	"time"
)

var settings BotSettings

func handle(upd telegram.Update) telegram.ReplyMessage {
	log.Printf("Update: %v\n", upd)
	userName := upd.Message.Chat.Username

	var userInfo UserInfo
	userInfoPath := fmt.Sprintf(".ercInfoBot/users/%v.json", userName)
	userInfoErr := config.UnmarshalJson(&userInfo, userInfoPath)

	if nil != userInfoErr {
		log.Printf("Login not found for user %v. Creating stub.\n", userName)
		config.MarshalJson(userInfo, userInfoPath)
	} else {
		log.Printf("Login for user %v found: %v\n", userName, userInfo.Login)
	}

	cmd, cmdParseErr := parseCommand(upd)

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
	default:
		log.Printf("Unknown command: %v\n", cmd.Command)
		return help(upd)
	}
}

func register(upd telegram.Update, login string, password string, account string) telegram.ReplyMessage {
	balanceInfo, errBalanceInfo := getBalanceInfo(login, password, account)
	if errBalanceInfo != nil {
		return telegram.ReplyMessage{
			ChatId: upd.Message.Chat.Id,
			Text:   "Wrong login/password. Please, register: /reg <login> <password>",
		}
	}

	var userInfo UserInfo
	userInfo.Login = login
	userInfo.Password = password
	userInfo.Account = account

	userInfoPath := fmt.Sprintf(".ercInfoBot/users/%v.json", upd.Message.Chat.Username)
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

func get(upd telegram.Update, userInfo UserInfo) telegram.ReplyMessage {
	if userInfo.Login == "" {
		return telegram.ReplyMessage{
			ChatId: upd.Message.Chat.Id,
			Text:   "Please, register: /reg <login> <password>",
		}
	}

	balanceInfo, _ := getBalanceInfo(userInfo.Login, userInfo.Password, userInfo.Account)
	return telegram.ReplyMessage{
		ChatId: upd.Message.Chat.Id,
		Text:   fmt.Sprintf("%v", balanceInfo.AtTheEnd.Total),
		ReplyMarkup: telegram.InlineKeyboardMarkup{
			Keyboard: [][]telegram.KeyboardButton{
				{
					telegram.KeyboardButton{Text: "/get"},
				},
			},
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

func help(upd telegram.Update) telegram.ReplyMessage {
	helpMsg :=
		"/reg $login $password $account -- register your account\n" +
			"/notify $on|$off -- set up notifications"

	return telegram.ReplyMessage{
		ChatId: upd.Message.Chat.Id,
		Text:   helpMsg,
	}
}

func parseCommand(upd telegram.Update) (Command, error) {
	reCommand, _ := regexp.Compile("((/reg) (\\w+) (\\w+) (\\w+))|((/help))|((/get))|(/notify (on|off))")
	match := reCommand.FindAllStringSubmatch(upd.Message.Text, -1)
	log.Printf("TEXT: %v\n", upd.Message.Text)
	log.Printf("MATCH: %v\n", match)
	cmd := Command{}
	if len(match) == 0 {
		return cmd, fmt.Errorf("Unknown command: %v\n", upd.Message.Text)
	}

	cmd.Command = match[0][0]
	switch cmd.Command {
	case "/reg":
		cmd.Args = make([]string, 3, 3)
		cmd.Args[0] = match[0][2]
		cmd.Args[1] = match[0][3]
		cmd.Args[2] = match[0][4]
	case "/notify":
		cmd.Args = make([]string, 1, 1)
		cmd.Args[0] = match[0][2]
	case "/help":
		cmd.Args = make([]string, 0, 0)
	case "/get":
		cmd.Args = make([]string, 0, 0)
	default:
		return cmd, fmt.Errorf("Unknown command: %v", cmd.Command)
	}

	return cmd, nil
}

func main() {
	errCfg := config.UnmarshalJson(&settings, ".ercInfoBot/settings.json")
	if nil != errCfg {
		log.Printf("Unable to get config: %v\n", errCfg)
		return
	}
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

type Command struct {
	Command string
	Args    []string
}
