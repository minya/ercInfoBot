package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/minya/erc/erclib"
	"github.com/minya/ercInfoBot/model"
	"github.com/minya/telegram"
)

type ercclient interface {
	GetAccounts() ([]erclib.Account, error)
	GetBalanceInfo(account string, t time.Time) (erclib.BalanceInfo, error)
	GetReceipt(accNumber string) ([]byte, error)
}

type handler struct {
	storage        model.UserStorage
	buildERCClient func(string, string) ercclient
}

func createHandler(storage model.UserStorage, buildERCClient func(string, string) ercclient) handler {
	return handler{storage: storage, buildERCClient: buildERCClient}
}

//handle every incoming update
func (h *handler) handle(upd telegram.Update) interface{} {
	log.Printf("Update: %v\n", upd)
	userID := upd.CallbackQuery.From.Id
	if userID == 0 {
		userID = upd.Message.From.Id
	}

	userInfo, userInfoErr := h.storage.GetUserInfo(userID)

	if nil != userInfoErr {
		log.Printf("Login not found for user %v. Creating stub.\n", userID)
		h.storage.SaveUser(userID, userInfo)
	} else {
		log.Printf("Login for user %v found: %v\n", userID, userInfo.Login)
	}

	cmdText := upd.CallbackQuery.Data
	if cmdText == "" {
		log.Printf("Parse cmd from Message\n")
		cmdText = upd.Message.Text
	}
	cmd, cmdParseErr := ParseCommand(cmdText)
	if cmdParseErr != nil {
		log.Printf("Error parse command: %v\n", cmdParseErr)
		return help(upd)
	}

	log.Printf("Process command: %v\n", cmd)

	if cmd.Command == "/reg" {
		return h.register(upd, cmd.Args[0], cmd.Args[1])
	}

	if cmd.Command == "/help" {
		return help(upd)
	}

	log.Printf("USERINFO %v\n", userInfo)
	if userInfo.Login == "" {
		return replyWithMessage(
			upd, "Подключите личный кабинет: /reg <login> <password>")
	}

	var accountNum string
	ercClient := h.buildERCClient(userInfo.Login, userInfo.Password)
	accounts, _ := ercClient.GetAccounts()
	if len(cmd.Args) == 0 || cmd.Args[0] == "" {
		log.Printf("No account in query")
		if len(accounts) > 1 {
			return replyChooseAccount(upd.Message.Chat.Id, cmd.Command, accounts)
		}
		accountNum = accounts[0].Number
	} else {
		accountNum = cmd.Args[0]
	}

	log.Printf("Account number is %v\n", accountNum)

	account, errNoAccount := findAccount(accounts, accountNum)
	if errNoAccount != nil {
		return replyWithMessage(
			upd,
			fmt.Sprintf("Лицевой счет %v не найден среди подключенных в личном кабинете", accountNum))
	}

	switch cmd.Command {
	case "/notify":
		return h.setUpNotification(upd, ercClient, account)
	case "/get":
		return get(upd, ercClient, account)
	case "/receipt":
		return receipt(upd, ercClient, account)
	default:
		log.Printf("Unknown command: %v\n", cmd.Command)
		return help(upd)
	}
}

func replyChooseAccount(chatID int, sourceCmd string, accounts []erclib.Account) telegram.ReplyMessage {
	return telegram.ReplyMessage{
		ChatId:      chatID,
		Text:        fmt.Sprintf("По какому лицевому счету вы хотите %v?", makeOpName(sourceCmd)),
		ReplyMarkup: chooseAccountButtons(sourceCmd, accounts),
	}
}

func makeOpName(cmd string) string {
	switch cmd {
	case "/get":
		return "получить баланс"
	case "/receipt":
		return "получить квитанцию"
	case "/notify":
		return "настроить уведомления"
	}
	return "произвести операцию"
}

func chooseAccountButtons(sourceCmd string, accounts []erclib.Account) telegram.InlineKeyboardMarkup {
	keyboard := [][]telegram.InlineKeyboardButton{}
	for _, account := range accounts {
		keyboard = append(keyboard, []telegram.InlineKeyboardButton{
			telegram.InlineKeyboardButton{
				Text:         account.Address,
				CallbackData: fmt.Sprintf("%v %v", sourceCmd, account.Number),
			},
		})
	}

	return telegram.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}

func (h *handler) register(upd telegram.Update, login string, password string) interface{} {
	ercClient := erclib.NewErcClientWithCredentials(login, password)
	accounts, errAccounts := ercClient.GetAccounts()
	if errAccounts != nil {
		return telegram.ReplyMessage{
			ChatId: upd.Message.Chat.Id,
			Text:   "Wrong login/password. Please, register: /reg <login> <password>",
		}
	}

	var userInfo model.UserInfo
	userInfo.Login = login
	userInfo.Password = password

	saveErr := h.storage.SaveUser(upd.Message.From.Id, userInfo)

	if saveErr != nil {
		log.Printf(fmt.Sprintf("Error while saving user: %v\n", saveErr))
		return telegram.ReplyMessage{
			ChatId: upd.Message.Chat.Id,
			Text:   "Error while registering user",
		}
	}

	return telegram.ReplyMessage{
		ChatId: upd.Message.Chat.Id,
		Text: fmt.Sprintf("You have been registered. "+
			"Your accounts are : %v", listAccounts(&accounts)),
		ReplyMarkup: replyButtons(),
	}
}

func get(upd telegram.Update, ercClient ercclient, account erclib.Account) interface{} {
	balanceInfo, _ := ercClient.GetBalanceInfo(account.Number, time.Now())
	return telegram.ReplyMessage{
		ChatId:      getReplyToChatID(upd),
		Text:        formatBalance(account, balanceInfo),
		ReplyMarkup: replyButtons(),
	}
}

func receipt(upd telegram.Update, ercClient ercclient, account erclib.Account) interface{} {
	receipt, err := ercClient.GetReceipt(account.Number)
	if err != nil {
		log.Printf("%v\n", err)
		return replyWithMessage(upd, "Не удалось загрузить квитанцию")
	}

	return telegram.ReplyDocument{
		ChatId:  getReplyToChatID(upd),
		Caption: fmt.Sprintf("Квитанция (%v)", account.Address),
		InputFile: telegram.InputFile{
			Content:  receipt,
			FileName: fmt.Sprintf("%v.pdf", account.Number),
		},
		ReplyMarkup: replyButtons(),
	}
}

func (h *handler) setUpNotification(
	upd telegram.Update,
	ercClient ercclient,
	account erclib.Account) telegram.ReplyMessage {

	balanceInfo, err := ercClient.GetBalanceInfo(account.Number, time.Now())
	var lastSeenState string
	if err == nil {
		lastSeenState = fmt.Sprintf("%v", balanceInfo)
	}
	userID := getUserID(upd)
	user, err := h.storage.GetUserInfo(userID)
	chatID := getReplyToChatID(upd)
	if err != nil {
		return telegram.ReplyMessage{
			ChatId:      chatID,
			Text:        "Ошибка",
			ReplyMarkup: replyButtons(),
		}
	}

	if user.Subscriptions == nil {
		user.Subscriptions = make(map[string]model.SubscriptionInfo)
	}
	user.Subscriptions[account.Number] = model.SubscriptionInfo{
		ChatID:        chatID,
		LastSeenState: lastSeenState,
	}

	h.storage.SaveUser(userID, user)

	return telegram.ReplyMessage{
		ChatId: getReplyToChatID(upd),
		Text: fmt.Sprintf(
			"Вы подписаны на уведомления по лицевому счету %v (%v)",
			account.Number,
			account.Address),
		ReplyMarkup: replyButtons(),
	}
}

func getReplyToChatID(upd telegram.Update) int {
	chatToReply := upd.Message.Chat.Id
	if chatToReply == 0 {
		chatToReply = upd.CallbackQuery.Message.Chat.Id
	}
	return chatToReply
}

func getUserID(upd telegram.Update) int {
	if upd.CallbackQuery.From.Id != 0 {
		return upd.CallbackQuery.From.Id
	}
	return upd.Message.From.Id
}

func formatBalance(account erclib.Account, balance erclib.BalanceInfo) string {
	return fmt.Sprintf(
		"%v:\n%v\n%v",
		account.Address,
		balance.Month,
		formatRequisites(balance.Rows))
}

func formatRequisites(rows []erclib.BalanceRow) string {
	var sb strings.Builder
	for _, row := range rows {
		sb.WriteString(fmt.Sprintf("%v: %v\n", row.Requisite, row.Amount))
	}
	return sb.String()
}

func listAccounts(accounts *[]erclib.Account) string {
	var builder strings.Builder
	first := true
	for _, account := range *accounts {
		if !first {
			builder.WriteString(", ")
		}
		first = false
		builder.WriteString(fmt.Sprintf("%v (%v)", account.Number, account.Address))
	}
	return builder.String()
}

func help(upd telegram.Update) telegram.ReplyMessage {
	helpMsg :=
		"/reg <login> <password> – Подключить личный кабинет\n" +
			"/receipt – Скачать квитанцию в pdf\n" +
			"/get – получить информацию о задолженности\n" +
			"/notify – подключить уведомления о задолженности"

	return telegram.ReplyMessage{
		ChatId: upd.Message.Chat.Id,
		Text:   helpMsg,
	}
}

func replyWithMessage(upd telegram.Update, message string) telegram.ReplyMessage {
	return telegram.ReplyMessage{
		ChatId:      getReplyToChatID(upd),
		Text:        message,
		ReplyMarkup: replyButtons(),
	}
}

func findAccount(accounts []erclib.Account, num string) (erclib.Account, error) {
	for _, acc := range accounts {
		if acc.Number == num {
			return acc, nil
		}
	}
	return erclib.Account{}, fmt.Errorf("No account with number %v", num)
}
