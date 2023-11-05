package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/eleksir/go-xmpp"
	"github.com/hjson/hjson-go"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

// sigHandler Хэндлер сигналов закрывает все бд, все сетевые соединения и сваливает из приложения.
func sigHandler() {
	log.Debug("Installing signal handler")

	for s := range sigChan {
		switch s {
		case syscall.SIGINT:
			log.Infoln("Got SIGINT, quitting")
		case syscall.SIGTERM:
			log.Infoln("Got SIGTERM, quitting")
		case syscall.SIGQUIT:
			log.Infoln("Got SIGQUIT, quitting")

		// Заходим на новую итерацию, если у нас "неинтересный" сигнал.
		default:
			continue
		}

		// Чтобы не срать в логи ошибками, проставим shutdown state приложения в true.
		shutdown = true

		if isConnected && !shutdown {
			log.Debug("Try to set our presence to Unavailable and status to Offline")

			// Вот тут понадобится коллекция известных пользователей, чтобы им разослать presence, что бот свалил в offline
			// Пока за неимением лучшего сообщим об этом самим себе.
			for _, room := range roomsConnected {
				if _, err := talk.SendPresence(
					xmpp.Presence{ //nolint:exhaustruct
						To:     room,
						Status: "Offline",
						Type:   "unavailable",
					},
				); err != nil {
					log.Infof("Unable to send presence to jabber server: %s", err)
				}
			}

			// И закрываем соединение.
			log.Debugf("Closing connection to jabber server")

			if err := talk.Close(); err != nil {
				log.Infof("Unable to close connection to jabber server: %s", err)
			}
		}

		os.Exit(0)
	}
}

// readWhitelist читает и валидирует белые списки пользователей.
func readWhitelist() error {
	var (
		whitelistLoaded = false
		err             error
		executablePath  string
	)

	executablePath, err = os.Executable()

	if err != nil {
		err = fmt.Errorf("unable to get current executable path: %w", err)

		return err
	}

	whitelistJSONPath := fmt.Sprintf("%s/data/whitelist.json", filepath.Dir(executablePath))

	locations := []string{
		"~/.bunyPresense-jabber-bot-whitelist.json",
		"~/bunyPresense-jabber-bot-whitelist.json",
		"/etc/bunyPresense-jabber-bot-whitelist.json",
		whitelistJSONPath,
	}

	for _, location := range locations {
		fileInfo, err := os.Stat(location)

		// Предполагаем, что файла либо нет, либо мы не можем его прочитать, второе надо бы логгировать, но пока забьём
		if err != nil {
			continue
		}

		// Файл белого списка длинноват для белого списка, попробуем следующего кандидата
		if fileInfo.Size() > 2097152 {
			log.Warnf("Whitelist file %s is too long for whitelist, skipping", location)

			continue
		}

		buf, err := os.ReadFile(location)

		// Не удалось прочитать, попробуем следующего кандидата
		if err != nil {
			log.Warnf("Skip reading whitelist file %s: %s", location, err)

			continue
		}

		// Исходя из документации, hjson какбы умеет парсить "кривой" json, но парсит его в map-ку.
		// Интереснее на выходе получить структурку: то есть мы вначале конфиг преобразуем в map-ку, затем эту map-ку
		// сериализуем в json, а потом json превращаем в структурку. Не очень эффективно, но он и нечасто требуется.
		var (
			sampleWhitelist myWhiteList
			tmp             map[string]interface{}
		)

		err = hjson.Unmarshal(buf, &tmp)

		// Не удалось распарсить - попробуем следующего кандидата
		if err != nil {
			log.Warnf("Skip parsing whitelist file %s: %s", location, err)

			continue
		}

		tmpJSON, err := json.Marshal(tmp)

		// Не удалось преобразовать map-ку в json
		if err != nil {
			log.Warnf("Skip parsing whitelist file %s: %s", location, err)

			continue
		}

		if err := json.Unmarshal(tmpJSON, &sampleWhitelist); err != nil {
			log.Warnf("Skip parsing whitelist file %s: %s", location, err)

			continue
		}

		whiteList = sampleWhitelist
		whitelistLoaded = true

		log.Infof("Using %s as whiteList file", location)

		break
	}

	if !whitelistLoaded {
		return errors.New("whitelist was not loaded!") //nolint:goerr113
	}

	return err
}

// readBlacklist читает и валидирует чёрные списки пользователей.
func readBlacklist() error {
	var (
		blacklistLoaded = false
		err             error
		executablePath  string
	)

	executablePath, err = os.Executable()

	if err != nil {
		err = fmt.Errorf("unable to get current executable path: %w", err)

		return err
	}

	whitelistJSONPath := fmt.Sprintf("%s/data/blacklist.json", filepath.Dir(executablePath))

	locations := []string{
		"~/.bunyPresense-jabber-bot-blacklist.json",
		"~/bunyPresense-jabber-bot-blacklist.json",
		"/etc/bunyPresense-jabber-bot-blacklist.json",
		whitelistJSONPath,
	}

	for _, location := range locations {
		fileInfo, err := os.Stat(location)

		// Предполагаем, что файла либо нет, либо мы не можем его прочитать, второе надо бы логгировать, но пока забьём
		if err != nil {
			continue
		}

		// Файл чёрного списка длинноват для чёрного списка, попробуем следующего кандидата
		if fileInfo.Size() > 16777216 {
			log.Warnf("Blacklist file %s is too long for blacklist, skipping", location)

			continue
		}

		buf, err := os.ReadFile(location)

		// Не удалось прочитать, попробуем следующего кандидата
		if err != nil {
			log.Warnf("Skip reading blacklist file %s: %s", location, err)

			continue
		}

		// Исходя из документации, hjson какбы умеет парсить "кривой" json, но парсит его в map-ку.
		// Интереснее на выходе получить структурку: то есть мы вначале конфиг преобразуем в map-ку, затем эту map-ку
		// сериализуем в json, а потом json превращаем в структурку. Не очень эффективно, но он и нечасто требуется.
		var (
			sampleBlacklist myBlackList
			tmp             map[string]interface{}
		)

		err = hjson.Unmarshal(buf, &tmp)

		// Не удалось распарсить - попробуем следующего кандидата
		if err != nil {
			log.Warnf("Skip parsing blacklist file %s: %s", location, err)

			continue
		}

		tmpJSON, err := json.Marshal(tmp)

		// Не удалось преобразовать map-ку в json
		if err != nil {
			log.Warnf("Skip parsing blacklist file %s: %s", location, err)

			continue
		}

		if err := json.Unmarshal(tmpJSON, &sampleBlacklist); err != nil {
			log.Warnf("Skip parsing whitelist file %s: %s", location, err)

			continue
		}

		blackList = sampleBlacklist
		blacklistLoaded = true

		log.Infof("Using %s as blacklist file", location)

		break
	}

	if !blacklistLoaded {
		return errors.New("blacklist was not loaded!")
	}

	return err
}

// establishConnection устанавливает соединение с jabber-сервером.
func establishConnection() {
	var err error

	if connecting && !isConnected {
		return
	}

	// Проставляем глобальные переменные.
	connecting = true
	isConnected = false
	roomsConnected = make([]string, 0)

	log.Debugf("Establishing connection to %s", options.Host)
	talk, err = options.NewClient()

	if err != nil {
		gTomb.Kill(err)

		return
	}

	// По идее keepalive должен же проходить только, если мы уже на сервере, так?
	if _, err := talk.SendKeepAlive(); err != nil {
		log.Errorf("Try to send initial KeepAlive, got error: %s", err)

		gTomb.Kill(err)

		return
	}

	log.Info("Connected")

	// Джойнимся к чятикам, но делаем это в фоне, чтобы не блочиться на ошибках, например, если бота забанили
	for _, roomStruct := range config.Jabber.Channels {
		room := roomStruct.Name
		go joinMuc(room)
	}

	go RotateStatus("")

	lastActivity = time.Now().Unix()
	connecting = false
	isConnected = true

	log.Debugf("Sending disco#info to %s", config.Jabber.Server)

	_, err = talk.DiscoverInfo(talk.JID(), config.Jabber.Server)

	if err != nil {
		log.Infof("Unable to send disco#info to jabber server: %s", err)
		gTomb.Kill(err)

		return
	}
}

// joinMu джойнится к конференциям/каналам/комнатам в джаббере.
func joinMuc(room string) {
	log.Debugf("Sending disco#info from %s to %s", talk.JID(), room)

	if _, err := talk.DiscoverInfo(talk.JID(), room); err != nil {
		log.Infof("Unable to send disco#info to MUC %s: %s", room, err)

		gTomb.Kill(err)

		return
	}

	// Ждём, пока muc нам вернёт список фичей.
	for i := 0; i < (20 * int(config.Jabber.ConnectionTimeout)); i++ {
		var (
			myRoom    interface{}
			supported bool
			exist     bool
		)

		time.Sleep(50 * time.Millisecond)

		if myRoom, exist = mucCapsList.Get(room); !exist {
			// Пока не задискаверилась
			continue
		}

		if supported, exist = myRoom.(map[string]bool)["muc_unsecured"]; exist {
			if supported {
				break
			}

			log.Infof("Unable to join to password-protected room. Don't know how to enter passwords :)")

			return
		}
	}

	if _, err := talk.JoinMUCNoHistory(room, config.Jabber.Nick); err != nil {
		log.Errorf("Unable to join to MUC: %s", room)

		gTomb.Kill(err)

		return
	}

	log.Infof("Joining to MUC: %s", room)

	// Ждём, когда прилетит presence из комнаты, тогда мы точно знаем, что мы вошли.
	entered := false

	for i := 0; i < (20 * int(config.Jabber.ConnectionTimeout)); i++ {
		time.Sleep(50 * time.Millisecond)

		if slices.Contains(roomsConnected, room) {
			entered = true

			break
		}
	}

	if !entered {
		log.Errorf(
			"Unable to enter to MUC %s, join timeout after %d seconds (server does not return my presence for this room)",
			room,
			20*int(config.Jabber.ConnectionTimeout)+1,
		)

		return
	}

	// Вот теперь точно можно слать статус.
	log.Infof("Joined to MUC: %s", room)

	go RotateStatus(room)

	// Время проверить участников на предмет злобности
	namesInterface, present := roomPresences.Get(room)

	// Если room есть в списке presence-ов, то фигачим. room там должен быть, просто обязан.
	if present {
		for _, name := range interfaceToStringSlice(namesInterface) {
			var v xmpp.Presence

			_ = json.Unmarshal([]byte(name), &v)
			log.Infof("Fake presence forged for %s just for on-enter check", name)
			// Оно там внутри всё само обработает, если вдруг возникнет wire error, то зарекконетится.
			_ = bunyPresense(v)
		}
	}
}

// probeServerLiveness проверяет живость соединения с сервером. Для многих серверов обязательная штука, без которой
// они выкидывают клиента через некоторое время неактивности.
func probeServerLiveness() { //nolint:gocognit
	defer gTomb.Done()

	for {
		select {
		case <-gTomb.Dying():
			return

		default:
			for {
				if shutdown {
					return
				}

				sleepTime := time.Duration(config.Jabber.ServerPingDelay) * 1000 * time.Millisecond
				sleepTime += time.Duration(rand.Int63n(1000*config.Jabber.PingSplayDelay)) * time.Millisecond
				time.Sleep(sleepTime)

				if !isConnected {
					continue
				}

				// Пингуем, только если не было никакой активности в течение > config.Jabber.ServerPingDelay,
				// в худшем случе это будет ~ (config.Jabber.PingSplayDelay * 2) + config.Jabber.PingSplayDelay
				// if (time.Now().Unix() - lastServerActivity) < (config.Jabber.ServerPingDelay + config.Jabber.PingSplayDelay) {
				//	continue
				// }

				if serverCapsQueried { // Сервер ответил на disco#info
					var (
						value interface{}
						exist bool
					)

					value, exist = serverCapsList.Get("urn:xmpp:ping")

					switch {
					// Сервер анонсировал, что умеет в c2s пинги
					case exist && value.(bool):
						// Таймаут c2s пинга. Возьмём сумму задержки между пингами, добавим таймаут коннекта и добавим
						// максимальную корректировку разброса.
						txTimeout := config.Jabber.ServerPingDelay + config.Jabber.ConnectionTimeout
						txTimeout += config.Jabber.PingSplayDelay
						rxTimeout := txTimeout

						rxTimeAgo := time.Now().Unix() - serverPingTimestampRx

						if serverPingTimestampTx > 0 { // Первая пуля от нас ушла...
							switch {
							// Давненько мы не получали понгов от сервера, вероятно, соединение с сервером утеряно?
							case rxTimeAgo > (rxTimeout * 2):
								err := fmt.Errorf( //nolint:goerr113
									"stall connection detected. No c2s pong for %d seconds",
									rxTimeAgo,
								)

								gTomb.Kill(err)

								return

							// По-умолчанию, мы отправляем c2s пинг
							default:
								log.Debugf("Sending c2s ping from %s to %s", talk.JID(), config.Jabber.Server)

								if err := talk.PingC2S(talk.JID(), config.Jabber.Server); err != nil {
									gTomb.Kill(err)

									return
								}

								serverPingTimestampTx = time.Now().Unix()
							}
						} else { // Первая пуля пока не вылетела, отправляем
							log.Debugf("Sending first c2s ping from %s to %s", talk.JID(), config.Jabber.Server)

							if err := talk.PingC2S(talk.JID(), config.Jabber.Server); err != nil {
								gTomb.Kill(err)

								return
							}

							serverPingTimestampTx = time.Now().Unix()
						}

					// Сервер не анонсировал, что умеет в c2s пинги
					default:
						log.Debug("Sending keepalive whitespace ping")

						if _, err := talk.SendKeepAlive(); err != nil {
							gTomb.Kill(err)

							return
						}
					}
				} else { // Сервер не ответил на disco#info
					log.Debug("Sending keepalive whitespace ping")

					if _, err := talk.SendKeepAlive(); err != nil {
						gTomb.Kill(err)

						return
					}
				}
			}
		}
	}
}

// probeMUCLiveness Пингует MUC-и, нужно для проверки, что клиент ещё находится в MUC-е.
func probeMUCLiveness() { //nolint:gocognit
	for {
		select {
		case <-gTomb.Dying():
			return

		default:
			for {
				for _, room := range roomsConnected {
					var (
						exist          bool
						lastActivityTS interface{}
					)

					// Если записи про комнату нету, то пинговать её бессмысленно.
					if lastActivityTS, exist = lastMucActivity.Get(room); !exist {
						continue
					}

					// Если время последней активности в чятике не превысило
					// config.Jabber.ServerPingDelay + config.Jabber.PingSplayDelay, ничего не пингуем.
					if (time.Now().Unix() - lastActivityTS.(int64)) < (config.Jabber.ServerPingDelay + config.Jabber.PingSplayDelay) {
						continue
					}

					/* Пинг MUC-а по сценарию без серверной оптимизации мы реализовывать не будем. Это как-то не надёжно.
					go func(room string) {
						// Небольшая рандомная задержка перед пингом комнаты
						sleepTime := time.Duration(rand.Int63n(1000*config.Jabber.PingSplayDelay)) * time.Millisecond //nolint:gosec
						time.Sleep(sleepTime)

						if err := talk.PingS2S(talk.JID(), room+"/"+config.Jabber.Nick); err != nil {
							gTomb.Kill(err)

							return
						}
					}(room)
					*/

					var roomMap interface{}

					roomMap, exist = mucCapsList.Get(room)

					// Пинги комнаты проводим, только если она записана, как прошедшая disco#info и поддерживающая
					// Server Optimization.
					if exist && roomMap.(map[string]bool)["http://jabber.org/protocol/muc#self-ping-optimization"] {
						go func(room string) {
							// Небольшая рандомная задержка перед пингом комнаты.
							sleepTime := time.Duration(rand.Int63n(1000*config.Jabber.PingSplayDelay)) * time.Millisecond
							time.Sleep(sleepTime)

							log.Debugf("Sending MUC ping from %s to %s", talk.JID(), room)

							if err := talk.PingS2S(talk.JID(), room); err != nil {
								err := fmt.Errorf("unable to ping MUC %s: %w", room, err)
								gTomb.Kill(err)

								return
							}
						}(room)
					}
				}

				time.Sleep(time.Duration(config.Jabber.MucPingDelay) * time.Second)
			}
		}
	}
}

// RotateStatus периодически изменяет статус бота в MUC-е согласно настройкам из кофига.
func RotateStatus(room string) {
	defer gTomb.Done()

	for {
		select {
		case <-gTomb.Dying():
			return

		default:
			// TODO: Переделать на ticker-ы
			totalSleepTime := time.Duration(config.Jabber.RuntimeStatus.RotationTime) * time.Second
			totalSleepTime += time.Duration(config.Jabber.RuntimeStatus.RotationSplayTime) * time.Second

			for {
				status := randomPhrase(config.Jabber.RuntimeStatus.Text)
				log.Debugf("Set status for MUC: %s to: %s", room, status)

				var p xmpp.Presence

				if room != "" {
					p = xmpp.Presence{ //nolint:exhaustruct
						To:     room,
						Status: status,
					}
				} else {
					p = xmpp.Presence{ //nolint:exhaustruct
						Status: status,
					}
				}

				if _, err := talk.SendPresence(p); err != nil {
					log.Infof("Unable to send presence to MUC %s: %s", room, err)
					gTomb.Kill(err)

					return
				}

				// Если мы не хотим ротировать, то цикл нам тут не нужен, просто выходим.
				if config.Jabber.RuntimeStatus.RotationTime == 0 {
					gTomb.Done()

					return
				}

				time.Sleep(totalSleepTime)
			}
		}
	}
}

// randomPhrase Выдаёт одну рандомную фразу из даденного списка фраз.
func randomPhrase(list []string) string {
	phrase := ""

	if listLen := len(list); listLen > 0 {
		phrase = list[rand.Intn(listLen)]
	}

	return phrase
}

// interfaceToStringSlice превращает данный интерфейс в слайс строк.
// Если может, конечно :) .
func interfaceToStringSlice(iface interface{}) []string {
	var mySlice []string

	// А теперь мы начинаем дурдом, нам надо превратить ёбанный interface{} в []string
	// Поскольку interface{} может быть чем угодно, перестрахуемся
	if reflect.TypeOf(iface).Kind() == reflect.Slice {
		shit := reflect.ValueOf(iface)

		for i := 0; i < shit.Len(); i++ {
			mySlice = append(mySlice, fmt.Sprint(shit.Index(i)))
		}
	}

	return mySlice
}

// getRealJIDfromNick достаёт из запомненных presence-ов по даденному nick-у real jid с resource-ом. Nick должен
// содержать имя конфы, откуда участник.
func getRealJIDfromNick(fullNick string) string {
	var p xmpp.Presence

	room := (strings.SplitN(fullNick, "/", 2))[0]

	// Достанем presence участника
	presenceJSONInterface, present := roomPresences.Get(room)

	// Никого нет дома
	if !present {
		return ""
	}

	presenceJSONStrings := interfaceToStringSlice(presenceJSONInterface)

	for _, presepresenceJSONString := range presenceJSONStrings {
		_ = json.Unmarshal([]byte(presepresenceJSONString), &p)

		if p.From == fullNick {
			return p.JID
		}
	}

	return ""
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
