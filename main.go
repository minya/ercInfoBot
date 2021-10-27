package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/minya/erc/erclib"
	"github.com/minya/ercInfoBot/model"
	"github.com/minya/goutils/config"
	"github.com/minya/telegram"
)

func main() {
	settings, storage, updateCheckPeriod := initialize()
	ntf := notifier{botToken: settings.ID, storage: storage, sleepDuration: updateCheckPeriod}
	botApi := telegram.NewApi(settings.ID)
	ntf.Start(&botApi)
	var makeERCClient = func(l string, p string) ercclient {
		return erclib.NewErcClientWithCredentials(l, p)
	}
	h := createHandler(storage, makeERCClient)
	listenErr := telegram.StartListen(settings.ID, 8080, h.handle)
	if nil != listenErr {
		log.Printf("Unable to start listen: %v\n", listenErr)
	}
}

func initialize() (BotSettings, model.FirebaseStorage, time.Duration) {
	var settings BotSettings
	var storage model.FirebaseStorage
	var updateCheckPeriod time.Duration
	var logPath string
	flag.StringVar(&logPath, "logpath", "ercInfoBot.log", "Path to write logs")
	var configPath string
	flag.StringVar(&configPath, "cfg", "~/.ercInfoBot/settings.json", "Path to write logs")
	flag.Parse()
	setUpLogger(logPath)

	errCfg := config.UnmarshalJson(&settings, configPath)

	if nil != errCfg {
		panic("Unable to get config")
	}
	log.Printf("Config read: %v\n", settings)

	var errParseDuration error
	updateCheckPeriod, errParseDuration = time.ParseDuration(settings.UpdateCheckPeriod)
	if errParseDuration != nil {
		log.Fatalf("Unable to parse duration from '%v' \n", settings.UpdateCheckPeriod)
	}
	if !settings.areValid() {
		log.Fatalf("Incorrect settings: %v\n", settings)
		panic("Incorrect settings")
	}

	fbSettings := settings.StorageSettings
	storage = model.NewFirebaseStorage(
		fbSettings.BaseURL,
		fbSettings.APIKey,
		fbSettings.Login,
		fbSettings.Password)
	return settings, storage, updateCheckPeriod
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
	return theSettings.ID != "" &&
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
