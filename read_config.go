package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hjson/hjson-go"
	log "github.com/sirupsen/logrus"
)

// readConfig читает и валидирует конфиг, а также выставляет некоторые default-ы, если значений для параметров в конфиге
// нет.
func readConfig() error { //nolint:gocognit,gocyclo
	var (
		err            error
		executablePath string
		configLoaded   = false
	)

	executablePath, err = os.Executable()

	if err != nil {
		err := fmt.Errorf("unable to get current executable path: %w", err)

		return err
	}

	configJSONPath := fmt.Sprintf("%s/data/config.json", filepath.Dir(executablePath))

	locations := []string{
		"~/.bunyPresense-jabber-bot.json",
		"~/bunyPresense-jabber-bot.json",
		"/etc/bunyPresense-jabber-bot.json",
		configJSONPath,
	}

	for _, location := range locations {
		fileInfo, err := os.Stat(location)

		// Предполагаем, что файла либо нет, либо мы не можем его прочитать, второе надо бы логгировать, но пока забьём.
		if err != nil {
			continue
		}

		// Конфиг-файл длинноват для конфига, попробуем следующего кандидата.
		if fileInfo.Size() > 2777216 {
			log.Warnf("Config file %s is too long for config, skipping", location)

			continue
		}

		buf, err := os.ReadFile(location)

		// Не удалось прочитать, попробуем следующего кандидата.
		if err != nil {
			log.Warnf("Skip reading config file %s: %s", location, err)

			continue
		}

		// Исходя из документации, hjson какбы умеет парсить "кривой" json, но парсит его в map-ку.
		// Интереснее на выходе получить структурку: то есть мы вначале конфиг преобразуем в map-ку, затем эту map-ку
		// сериализуем в json, а потом json превращаем в структурку. Не очень эффективно, но он и нечасто требуется.
		var (
			sampleConfig myConfig
			tmp          map[string]interface{}
		)

		err = hjson.Unmarshal(buf, &tmp)

		// Не удалось распарсить - попробуем следующего кандидата.
		if err != nil {
			log.Warnf("Skip parsing config file %s: %s", location, err)

			continue
		}

		tmpJSON, err := json.Marshal(tmp)

		// Не удалось преобразовать map-ку в json
		if err != nil {
			log.Warnf("Skip parsing config file %s: %s", location, err)

			continue
		}

		if err := json.Unmarshal(tmpJSON, &sampleConfig); err != nil {
			log.Warnf("Skip parsing config file %s: %s", location, err)

			continue
		}

		// Значения для Jabber-клиента
		if sampleConfig.Jabber.Server == "" {
			log.Error("Jabber server is not defined in config, using localhost")
			sampleConfig.Jabber.Server = "localhost" //nolint:wsl
		}

		if sampleConfig.Jabber.Port == 0 {
			sampleConfig.Jabber.Port = 5222

			if sampleConfig.Jabber.Ssl {
				if !sampleConfig.Jabber.StartTLS {
					sampleConfig.Jabber.Port = 5223

					log.Info("Jabber port is not defined in config, using 5223")
				} else {
					log.Info("Jabber port is not defined in config, using 5222")
				}
			}
		}

		if !sampleConfig.Jabber.Ssl {
			sampleConfig.Jabber.StartTLS = false
		}

		if !sampleConfig.Jabber.Ssl || !sampleConfig.Jabber.StartTLS {
			sampleConfig.Jabber.SslVerify = false
		}

		// sampleConfig.Jabber.InsecureAllowUnencryptedAuth = false, если не задан

		if sampleConfig.Jabber.ConnectionTimeout == 0 {
			sampleConfig.Jabber.ConnectionTimeout = 10

			log.Info("Jabber server connection_timeout not defined in config, using 10 seconds")
		}

		if sampleConfig.Jabber.ReconnectDelay == 0 {
			sampleConfig.Jabber.ReconnectDelay = 3

			log.Info("Jabber server reconnect_delay not defined in config, using 3 seconds")
		}

		if sampleConfig.Jabber.ServerPingDelay == 0 {
			sampleConfig.Jabber.ServerPingDelay = 60

			log.Info("Jabber server_ping_delay not defined in config, using 60 seconds")
		}

		if sampleConfig.Jabber.MucPingDelay == 0 {
			sampleConfig.Jabber.MucPingDelay = 900

			log.Info("Jabber muc_ping_delay not defined in config, using 900 seconds")
		}

		if sampleConfig.Jabber.MucRejoinDelay == 0 {
			sampleConfig.Jabber.MucRejoinDelay = 3

			log.Info("Jabber muc_rejoin_delay not defined in config, using 3 seconds")
		}

		if sampleConfig.Jabber.PingSplayDelay == 0 {
			sampleConfig.Jabber.PingSplayDelay = 3

			log.Info("Jabber ping_splay_delay not defined in config, using 3 seconds")
		}

		if sampleConfig.Jabber.Nick == "" {
			return errors.New("jabber nick is not defined in config, quitting") //nolint:goerr113
		}

		if sampleConfig.Jabber.Resource == "" {
			sampleConfig.Jabber.Resource = "bunyPresense bot"

			log.Info("Jabber resource not defined in config, using bunyPresense bot")
		}

		if sampleConfig.Jabber.User == "" {
			sampleConfig.Jabber.User = fmt.Sprintf("%s@%s", sampleConfig.Jabber.Nick, sampleConfig.Jabber.Server)

			log.Infof("Jabber user not defined in config, guessing, it can be %s", sampleConfig.Jabber.User)
		}

		// Если sampleConfig.Jabber.Password не задан, то авторизации не будет

		// Если не задано ни одного мастера, то бот сам себе мастер
		if len(sampleConfig.Jabber.BotMasters) == 0 {
			sampleConfig.Jabber.BotMasters[0] = sampleConfig.Jabber.User
		}

		// Нам бот нужен в каких-то чат-румах, а не "просто так"
		if len(sampleConfig.Jabber.Channels) < 1 {
			return errors.New("no jabber channels/rooms defined in config, quitting") //nolint:goerr113
		}

		for n, channel := range sampleConfig.Jabber.Channels {
			if channel.Name == "" {
				return errors.New("no \"name\" entry in jabber channel config")
			}

			if channel.Nick == "" {
				sampleConfig.Jabber.Channels[n].Nick = sampleConfig.Jabber.Nick
			}

			// channel.Password может быть пустым, тогда пароля нет
			// channel.Bayes.Enabled будет false, если не проставлен

			if channel.Bayes.MinLength < 40 {
				channel.Bayes.MinLength = 40
			}

			if channel.Bayes.MinWords < 8 {
				channel.Bayes.MinWords = 8
			}

			switch channel.Bayes.DefaultAction {
			case "kick":
			case "ban":
			case "devoice":
			default:
				channel.Bayes.DefaultAction = "log"
			}

			// channel.AllCaps.Enabled будет false, если не указан
			if channel.AllCaps.MinLength < 10 {
				channel.AllCaps.MinLength = 10
			}

			switch channel.AllCaps.DefaultAction {
			case "kick":
			case "ban":
			case "devoice":
			default:
				channel.AllCaps.DefaultAction = "log"
			}
		}

		// Если список фраз с которыми стартует бот пустой, вносим в него 1 запись с пустой строкой
		if len(sampleConfig.Jabber.StartupStatus) == 0 {
			sampleConfig.Jabber.StartupStatus[0] = ""
		}

		// Если список статусов, с которыми работает бот пустой, вносим в него 1 запись с пустой строкой
		if len(sampleConfig.Jabber.RuntimeStatus.Text) == 0 {
			sampleConfig.Jabber.RuntimeStatus.Text[0] = ""
		}

		// Если sampleConfig.Jabber.RuntimeStatus.RotationTime не задан, то он равен 0
		// Если sampleConfig.Jabber.RuntimeStatus.RotationSplayTime не задан, то он равен 0

		// Если sampleConfig.Jabber.BanDelay не задан, то он равен 0
		// Если sampleConfig.Jabber.BanPhrasesEnable не задан, то он false

		// Если список фраз, с которыми банят пустой, то вносим в него одну позицию с пустой строкой
		if len(sampleConfig.Jabber.BanPhrases) == 0 {
			sampleConfig.Jabber.BanPhrases[0] = ""
		}

		if sampleConfig.CSign == "" {
			sampleConfig.CSign = "!"
		}

		if sampleConfig.Loglevel == "" {
			sampleConfig.Loglevel = "info"

			log.Info("loglevel not defined in config, using info")
		}

		// sampleConfig.Log = "" if not set

		config = sampleConfig
		configLoaded = true

		log.Infof("Using %s as config file", location) //nolint:wsl

		break
	}

	if !configLoaded {
		return errors.New("config was not loaded!") //nolint:goerr113
	}

	return err //nolint:wrapcheck
}
