package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/minya/erc/erclib"
	"github.com/minya/telegram"
	"github.com/minya/telegramInfoBot/core"
	"github.com/minya/telegramInfoBot/model"
)

// Process every update
func Process(upd telegram.Update, h *core.Handler) interface{} {
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
		return register(upd, cmd.Args[0], cmd.Args[1], h)
	}

	if cmd.Command == "/help" {
		return help(upd)
	}

	userID := core.GetUserID(&upd)
	userInfo, userInfoErr := h.Storage.GetUserInfo(userID)
	if userInfoErr != nil {
		return replyWithMessage(&upd, "Cant get user info", nil)
	}
	log.Printf("USERINFO %v\n", userInfo)
	if userInfo.Login == "" {
		return replyWithMessage(&upd, "Подключите личный кабинет: /reg <login> <password>", nil)
	}

	var accountNum string
	ercClient := erclib.NewErcClientWithCredentials(userInfo.Login, userInfo.Password)
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
			&upd, fmt.Sprintf("Лицевой счет %v не найден среди подключенных в личном кабинете", accountNum), nil)
	}

	switch cmd.Command {
	case "/notify":
		return setUpNotification(&upd, &ercClient, &account, h)
	case "/get":
		return get(&upd, &ercClient, &account)
	case "/receipt":
		return receipt(&upd, &ercClient, &account)
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

func register(upd telegram.Update, login string, password string, h *core.Handler) interface{} {
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

	saveErr := h.Storage.SaveUser(upd.Message.From.Id, &userInfo)

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

func get(upd *telegram.Update, ercClient *erclib.ErcClient, account *erclib.Account) interface{} {
	balanceInfo, _ := ercClient.GetBalanceInfo(account.Number, time.Now())
	return telegram.ReplyMessage{
		ChatId:      core.GetReplyToChatID(upd),
		Text:        formatBalance(account, &balanceInfo),
		ReplyMarkup: replyButtons(),
	}
}

func receipt(upd *telegram.Update, ercClient *erclib.ErcClient, account *erclib.Account) interface{} {
	receipt, err := ercClient.GetReceipt(account.Number)
	if err != nil {
		log.Printf("%v\n", err)
		return replyWithMessage(upd, "Не удалось загрузить квитанцию", nil)
	}

	return telegram.ReplyDocument{
		ChatId:  core.GetReplyToChatID(upd),
		Caption: fmt.Sprintf("Квитанция (%v)", account.Address),
		InputFile: telegram.InputFile{
			Content:  receipt,
			FileName: fmt.Sprintf("%v.pdf", account.Number),
		},
		ReplyMarkup: replyButtons(),
	}
}

func setUpNotification(
	upd *telegram.Update,
	ercClient *erclib.ErcClient,
	account *erclib.Account,
	h *core.Handler) telegram.ReplyMessage {

	balanceInfo, err := ercClient.GetBalanceInfo(account.Number, time.Now())
	var lastSeenState string
	if err == nil {
		lastSeenState = fmt.Sprintf("%v", balanceInfo)
	}
	userID := core.GetUserID(upd)
	user, err := h.Storage.GetUserInfo(userID)
	chatID := core.GetReplyToChatID(upd)
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

	h.Storage.SaveUser(userID, &user)

	return telegram.ReplyMessage{
		ChatId: core.GetReplyToChatID(upd),
		Text: fmt.Sprintf(
			"Вы подписаны на уведомления по лицевому счету %v (%v)",
			account.Number,
			account.Address),
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

func formatBalance(account *erclib.Account, balance *erclib.BalanceInfo) string {
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

func ProcessNotification(
	id int,
	userInfo *model.UserInfo,
	accountNum string,
	sub *model.SubscriptionInfo,
	api *telegram.Api,
	userStorage model.UserStorage) error {
	ercClient := erclib.NewErcClientWithCredentials(userInfo.Login, userInfo.Password)
	accounts, err := ercClient.GetAccounts()
	if err != nil {
		log.Printf("WARN  No accounts")
		return err
	}
	for accountNum, sub := range userInfo.Subscriptions {
		account, err := findAccount(accounts, accountNum)
		if err != nil {
			log.Printf("WARN  No account %v among accounts", accountNum)
			return err
		}
		msg, err := compareAndNotify(api, id, account, &sub, userInfo, &ercClient, userStorage)
		err = api.SendMessage(*msg)
		if err != nil {
			fmt.Printf("%v\n", err)
			return err
		}
	}
	return nil
}

func compareAndNotify(
	api *telegram.Api,
	userID int,
	account erclib.Account,
	sub *model.SubscriptionInfo,
	userInfo *model.UserInfo,
	ercClient *erclib.ErcClient,
	userStorage model.UserStorage) (*telegram.ReplyMessage, error) {

	if sub.ChatID == 0 {
		log.Printf("[Update] User %v is not subscribed. Skip.\n", userID)
		return nil, nil
	}
	balanceInfo, err := ercClient.GetBalanceInfo(account.Number, time.Now())
	if err != nil {
		log.Printf("[Update] Error: can't get balance for user %v\n", userID)
		return nil, err
	}
	newState := fmt.Sprintf("%v", balanceInfo)

	if sub.LastSeenState == "" {
		sub.LastSeenState = newState
		userInfo.Subscriptions[account.Number] = *sub
		userStorage.SaveUser(userID, userInfo)
		log.Printf("[Update] Initial balance correction for user %v\n", userID)
	} else if sub.LastSeenState != newState {
		log.Printf("[Update] Balance changed for user %v\n", userID)
		sub.LastSeenState = newState
		userInfo.Subscriptions[account.Number] = *sub
		userStorage.SaveUser(userID, userInfo)

		messageText := "Баланс обновился:\n" + formatBalance(&account, &balanceInfo)
		msg := telegram.ReplyMessage{
			ChatId:      sub.ChatID,
			Text:        messageText,
			ReplyMarkup: replyButtons(),
		}
		return &msg, nil
	}

	log.Printf("[Update] Balance hasn't been changed\n")
	return nil, nil
}

func chooseAccountButtons(sourceCmd string, accounts []erclib.Account) telegram.InlineKeyboardMarkup {
	keyboard := [][]telegram.InlineKeyboardButton{}
	for _, account := range accounts {
		keyboard = append(keyboard, []telegram.InlineKeyboardButton{
			{
				Text:         account.Address,
				CallbackData: fmt.Sprintf("%v %v", sourceCmd, account.Number),
			},
		})
	}

	return telegram.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}

func replyWithMessage(upd *telegram.Update, message string, replyButtons *telegram.ReplyKeyboardMarkup) telegram.ReplyMessage {
	return telegram.ReplyMessage{
		ChatId:      core.GetReplyToChatID(upd),
		Text:        message,
		ReplyMarkup: replyButtons,
	}
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
