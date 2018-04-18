package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/minya/erc/erclib"
	"github.com/minya/ercInfoBot/model"
	"github.com/minya/goutils/config"
	"github.com/minya/telegram"
)

var settings BotSettings
var storage model.FirebaseStorage

func handle(upd telegram.Update) interface{} {
	log.Printf("Update: %v\n", upd)
	userID := upd.Message.From.Id

	userInfo, userInfoErr := storage.GetUserInfo(userID)

	if nil != userInfoErr {
		log.Printf("Login not found for user %v. Creating stub.\n", userID)
		storage.SaveUser(userID, userInfo)
	} else {
		log.Printf("Login for user %v found: %v\n", userID, userInfo.Login)
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

	var userInfo model.UserInfo
	userInfo.Login = login
	userInfo.Password = password
	userInfo.Account = account

	storage.SaveUser(upd.Message.From.Id, userInfo)

	return telegram.ReplyMessage{
		ChatId: upd.Message.Chat.Id,
		Text: fmt.Sprintf("You have been registered. "+
			"Your balance is: %v", balanceInfo.AtTheEnd.Total),
		ReplyMarkup: replyButtons(),
	}
}

func get(upd telegram.Update, userInfo model.UserInfo) interface{} {
	log.Printf("USERINFO %v\n", userInfo)
	if userInfo.Login == "" {
		return telegram.ReplyMessage{
			ChatId: upd.Message.Chat.Id,
			Text:   "Please, register: /reg <login> <password> <account>",
		}
	}

	balanceInfo, _ := getBalanceInfo(userInfo.Login, userInfo.Password, userInfo.Account)
	return telegram.ReplyMessage{
		ChatId:      upd.Message.Chat.Id,
		Text:        formatBalance(balanceInfo),
		ReplyMarkup: replyButtons(),
	}
}

func receipt(upd telegram.Update, userInfo model.UserInfo) interface{} {
	receipt, _ := erclib.GetReceipt(userInfo.Login, userInfo.Password, userInfo.Account)
	return telegram.ReplyDocument{
		ChatId:  upd.Message.Chat.Id,
		Caption: "Квитанция",
		InputFile: telegram.InputFile{
			Content:  receipt,
			FileName: "receipt.pdf",
		},
		ReplyMarkup: replyButtons(),
	}
}

func setUpNotification(upd telegram.Update, userInfo model.UserInfo, turnOn bool) telegram.ReplyMessage {
	balanceInfo, err := getBalanceInfo(
		userInfo.Login, userInfo.Password, userInfo.Account)
	var lastSeenState string
	if err == nil {
		lastSeenState = fmt.Sprintf("%v", balanceInfo)
	}
	userID := upd.Message.From.Id
	user, err := storage.GetUserInfo(userID)
	if err != nil {
		return telegram.ReplyMessage{
			ChatId:      upd.Message.Chat.Id,
			Text:        "Ошибка",
			ReplyMarkup: replyButtons(),
		}
	}

	user.Subscription = model.SubscriptionInfo{
		ChatId:        upd.Message.Chat.Id,
		LastSeenState: lastSeenState,
	}

	storage.SaveUser(userID, user)

	return telegram.ReplyMessage{
		ChatId:      upd.Message.Chat.Id,
		Text:        "Вы подписаны на уведомления",
		ReplyMarkup: replyButtons(),
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
			"/receipt -- get receipt (pdf)\n" +
			"/get -- get balance\n" +
			"/notify $on|$off -- set up notifications"

	return telegram.ReplyMessage{
		ChatId: upd.Message.Chat.Id,
		Text:   helpMsg,
	}
}

func updateLoop(sleepDuration time.Duration) {
	for true {
		log.Printf("Update...\n")
		subsMap, err := storage.GetUsers()
		if err != nil {
			log.Printf("Error: %v\n", err)
		} else {
			for id, userInfo := range subsMap {
				log.Printf("[Update] Check user %v\n", id)
				if userInfo.Subscription.ChatId == 0 {
					log.Printf("[Update] User %v is not subscribed. Skip.\n", id)
					continue
				}
				balanceInfo, err := getBalanceInfo(
					userInfo.Login, userInfo.Password, userInfo.Account)
				if err != nil {
					log.Printf("[Update] Error: can't get balance for user %v\n", id)
					continue
				}
				newState := fmt.Sprintf("%v", balanceInfo)

				sub := &userInfo.Subscription
				if sub.LastSeenState == "" {
					sub.LastSeenState = newState
					storage.SaveUser(id, userInfo)
					log.Printf("[Update] Initial balance correction for user %v\n", id)
				} else if sub.LastSeenState != newState {
					log.Printf("[Update] Balance changed for user %v\n", id)
					sub.LastSeenState = newState
					storage.SaveUser(id, userInfo)

					messageText := "Баланс обновился:\n" + formatBalance(balanceInfo)
					msg := telegram.ReplyMessage{
						ChatId:      userInfo.Subscription.ChatId,
						Text:        messageText,
						ReplyMarkup: replyButtons(),
					}
					err = telegram.SendMessage(settings.ID, msg)
					if err != nil {
						fmt.Printf("%v\n", err)
					}
				} else {
					log.Printf("[Update] Balance hasn't been changed\n")
				}
			}
		}
		time.Sleep(sleepDuration)
	}
}

func main() {
	errCfg := config.UnmarshalJson(&settings, "~/.ercInfoBot/settings.json")
	if nil != errCfg {
		log.Printf("Unable to get config: %v\n", errCfg)
		return
	}

	duration, errParseDuration := time.ParseDuration(settings.UpdateCheckPeriod)
	if errParseDuration != nil {
		log.Fatalf("Unable to parse duration from '%v' \n", settings.UpdateCheckPeriod)
	}
	if !settings.areValid() {
		log.Fatalf("Incorrect settings: %v\n", settings)
	}
	fbSettings := settings.StorageSettings
	storage = model.NewFirebaseStorage(
		fbSettings.BaseURL,
		fbSettings.APIKey,
		fbSettings.Login,
		fbSettings.Password)

	go updateLoop(duration)
	listenErr := telegram.StartListen(settings.ID, 8080, handle)
	if nil != listenErr {
		log.Printf("Unable to start listen: %v\n", listenErr)
	}
}

func init() {
	var logPath string
	flag.StringVar(&logPath, "logpath", "ercInfoBot.log", "Path to write logs")
	flag.Parse()
	setUpLogger(logPath)
}

func setUpLogger(logPath string) {
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(logFile)
}

func replyButtons() telegram.ReplyKeyboardMarkup {
	return telegram.ReplyKeyboardMarkup{
		Keyboard: [][]telegram.KeyboardButton{
			{
				telegram.KeyboardButton{Text: "/receipt"},
				telegram.KeyboardButton{Text: "/get"},
			},
		},
		ResizeKeyboard: true,
	}
}

// BotSettings struct to represent stored settings
type BotSettings struct {
	ID                string           `json:"id"`
	UpdateCheckPeriod string           `json:"updateCheckPeriod"`
	StorageSettings   FirebaseSettings `json:"storageSettings"`
}

func (theSettings BotSettings) areValid() bool {
	fbSettings := &theSettings.StorageSettings
	return settings.ID != "" &&
		fbSettings.APIKey != "" &&
		fbSettings.BaseURL != "" &&
		fbSettings.Login != "" &&
		fbSettings.Password != "" &&
		theSettings.UpdateCheckPeriod != ""
}

// FirebaseSettings struct is to store/retrieve settings
type FirebaseSettings struct {
	BaseURL  string `json:"baseUrl"`
	APIKey   string `json:"apiKey"`
	Login    string `json:"login"`
	Password string `json:"password"`
}
