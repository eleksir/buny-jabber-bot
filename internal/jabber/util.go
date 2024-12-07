package jabber

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/eleksir/go-xmpp"
	log "github.com/sirupsen/logrus"
)

// RandomPhrase выдаёт одну рандомную фразу из даденного списка фраз.
func RandomPhrase(list []string) string {
	phrase := ""

	if listLen := len(list); listLen > 0 {
		phrase = list[rand.Intn(listLen)]
	}

	return phrase
}

// InterfaceToStringSlice превращает данный интерфейс в слайс строк.
// Если может, конечно :) .
func InterfaceToStringSlice(iface interface{}) []string {
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

// GetRealJIDfromNick достаёт из запомненных presence-ов по даденному nick-у real jid с resource-ом. Nick должен
// содержать имя конфы, откуда участник.
func (j *Jabber) GetRealJIDfromNick(fullNick string) string {
	var p xmpp.Presence

	room := (strings.SplitN(fullNick, "/", 2))[0]

	// Достанем presence участника
	presenceJSONInterface, present := j.RoomPresences.Get(room)

	// Никого нет дома
	if !present {
		return ""
	}

	presenceJSONStrings := InterfaceToStringSlice(presenceJSONInterface)

	for _, presepresenceJSONString := range presenceJSONStrings {
		_ = json.Unmarshal([]byte(presepresenceJSONString), &p)

		if p.From == fullNick {
			return p.JID
		}
	}

	return ""
}

// GetBotNickFromRoomConfig достаёт из настроек комнаты короткий ник бота, либо берёт значение из конфига, если ник в
// настройках комнаты не задан. Короткий ник не содержит название комнаты.
func (j *Jabber) GetBotNickFromRoomConfig(room string) string {
	for _, roomStruct := range j.C.Jabber.Channels {
		if roomStruct.Name == room {
			return roomStruct.Nick
		}
	}

	return j.C.Jabber.Nick
}

// RotateStatus периодически изменяет статус бота в MUC-е согласно настройкам из кофига.
func (j *Jabber) RotateStatus(room string) error {
	for {
		// TODO: Переделать на ticker-ы
		totalSleepTime := time.Duration(j.C.Jabber.RuntimeStatus.RotationTime) * time.Second
		totalSleepTime += time.Duration(j.C.Jabber.RuntimeStatus.RotationSplayTime) * time.Second

		for {
			status := RandomPhrase(j.C.Jabber.RuntimeStatus.Text)
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

			if _, err := j.Talk.SendPresence(p); err != nil {
				return fmt.Errorf("Unable to send presence to MUC %s: %s", room, err)
			}

			// Если мы не хотим ротировать, то цикл нам тут не нужен, просто выходим.
			if j.C.Jabber.RuntimeStatus.RotationTime == 0 {
				return nil
			}

			time.Sleep(totalSleepTime)
		}
	}
}

// JoinMuc джойнится к конференциям/каналам/комнатам в джаббере.
func (j *Jabber) JoinMuc(room string) error {
	log.Debugf("Sending disco#info from %s to %s", j.Talk.JID(), room)

	if _, err := j.Talk.DiscoverInfo(j.Talk.JID(), room); err != nil {
		return fmt.Errorf("Unable to send disco#info to MUC %s: %s", room, err)
	}

	// Ждём, пока muc нам вернёт список фичей.
	for i := 0; i < (20 * int(j.C.Jabber.ConnectionTimeout)); i++ {
		var (
			myRoom    interface{}
			supported bool
			exist     bool
		)

		time.Sleep(50 * time.Millisecond)

		if myRoom, exist = j.MucCapsList.Get(room); !exist {
			// Пока не задискаверилась.
			continue
		}

		if supported, exist = myRoom.(map[string]bool)["muc_unsecured"]; exist {
			if supported {
				break
			}

			log.Infof("Unable to join to password-protected room. Don't know how to enter passwords :)")

			return nil
		}
	}

	// Заходим в конфу, наконец-то
	if _, err := j.Talk.JoinMUCNoHistory(room, j.GetBotNickFromRoomConfig(room)); err != nil {
		return fmt.Errorf("Unable to join to MUC: %s, %w", room, err)
	}

	log.Infof("Joining to MUC: %s", room)

	// Ждём, когда прилетит presence из комнаты, тогда мы точно знаем, что мы вошли.
	entered := false

	for i := 0; i < (20 * int(j.C.Jabber.ConnectionTimeout)); i++ {
		time.Sleep(50 * time.Millisecond)

		if slices.Contains(j.RoomsConnected, room) {
			entered = true

			break
		}
	}

	if !entered {
		log.Errorf(
			"Unable to enter to MUC %s, join timeout after %d seconds (server does not return my presence for this room)",
			room,
			20*int(j.C.Jabber.ConnectionTimeout)+1,
		)

		return nil
	}

	// Вот теперь точно можно слать статус.
	log.Infof("Joined to MUC: %s", room)

	j.GTomb.Go(func() error { return j.RotateStatus(room) })

	// Время проверить участников на предмет злобности
	namesInterface, present := j.RoomPresences.Get(room)

	// Если room есть в списке presence-ов, то фигачим. room там должен быть, просто обязан.
	if present {
		for _, name := range InterfaceToStringSlice(namesInterface) {
			var v xmpp.Presence

			_ = json.Unmarshal([]byte(name), &v)
			log.Infof("Fake presence forged for %s just for on-enter check", name)
			// Оно там внутри всё само обработает, если вдруг возникнет wire error, то зарекконетится.
			_ = j.BunyPresense(v)
		}
	}

	return nil
}

// EstablishConnection устанавливает соединение с jabber-сервером.
func (j *Jabber) EstablishConnection() error {
	var err error

	if j.Connecting && !j.IsConnected {
		return nil
	}

	// Проставляем глобальные переменные.
	j.Connecting = true
	j.IsConnected = false
	j.RoomsConnected = make([]string, 0)

	log.Debugf("Establishing connection to %s", j.Options.Host)
	j.Talk, err = j.Options.NewClient()

	if err != nil {
		return fmt.Errorf("Unable to connect to %s: %w", j.Options.Host, err)
	}

	// По идее keepalive должен же проходить только, если мы уже на сервере, так?
	if _, err := j.Talk.SendKeepAlive(); err != nil {
		return fmt.Errorf("Try to send initial KeepAlive, got error: %w", err)
	}

	log.Info("Connected")

	// Джойнимся к чятикам, но делаем это в фоне, чтобы не блочиться на ошибках, например, если бота забанили
	for _, roomStruct := range j.C.Jabber.Channels {
		room := roomStruct.Name
		j.GTomb.Go(func() error { return j.JoinMuc(room) })
	}

	j.GTomb.Go(func() error { return j.RotateStatus("") })

	j.LastActivity = time.Now().Unix()
	j.Connecting = false
	j.IsConnected = true

	log.Debugf("Sending disco#info to %s", j.C.Jabber.Server)

	_, err = j.Talk.DiscoverInfo(j.Talk.JID(), j.C.Jabber.Server)

	if err != nil {
		return fmt.Errorf("Unable to send disco#info to jabber server: %s", err)
	}

	return nil
}

// ProbeServerLiveness проверяет живость соединения с сервером. Для многих серверов обязательная штука, без которой
// они выкидывают (дисконнектят) клиента через некоторое время неактивности.
func (j *Jabber) ProbeServerLiveness() error { //nolint:gocognit
	for {
		for {
			if j.Shutdown {
				return nil
			}

			sleepTime := time.Duration(j.C.Jabber.ServerPingDelay) * 1000 * time.Millisecond
			sleepTime += time.Duration(rand.Int63n(1000*j.C.Jabber.PingSplayDelay)) * time.Millisecond
			time.Sleep(sleepTime)

			if !j.IsConnected {
				continue
			}

			// Пингуем, только если не было никакой активности в течение > config.Jabber.ServerPingDelay,
			// в худшем случе это будет ~ (config.Jabber.PingSplayDelay * 2) + config.Jabber.PingSplayDelay
			// if (time.Now().Unix() - lastServerActivity) < (config.Jabber.ServerPingDelay + config.Jabber.PingSplayDelay) {
			//	continue
			// }

			if j.ServerCapsQueried { // Сервер ответил на disco#info
				var (
					value interface{}
					exist bool
				)

				value, exist = j.ServerCapsList.Get("urn:xmpp:ping")

				switch {
				// Сервер анонсировал, что умеет в c2s пинги
				case exist && value.(bool):
					// Таймаут c2s пинга. Возьмём сумму задержки между пингами, добавим таймаут коннекта и добавим
					// максимальную корректировку разброса.
					txTimeout := j.C.Jabber.ServerPingDelay + j.C.Jabber.ConnectionTimeout
					txTimeout += j.C.Jabber.PingSplayDelay
					rxTimeout := txTimeout

					rxTimeAgo := time.Now().Unix() - j.ServerPingTimestampRx

					if j.ServerPingTimestampTx > 0 { // Первая пуля от нас ушла...
						switch {
						// Давненько мы не получали понгов от сервера, вероятно, соединение с сервером утеряно?
						case rxTimeAgo > (rxTimeout * 2):
							err := fmt.Errorf( //nolint:goerr113
								"stall connection detected. No c2s pong for %d seconds",
								rxTimeAgo,
							)

							return err

						// По-умолчанию, мы отправляем c2s пинг
						default:
							log.Debugf("Sending c2s ping from %s to %s", j.Talk.JID(), j.C.Jabber.Server)

							if err := j.Talk.PingC2S(j.Talk.JID(), j.C.Jabber.Server); err != nil {
								return err
							}

							j.ServerPingTimestampTx = time.Now().Unix()
						}
					} else { // Первая пуля пока не вылетела, отправляем
						log.Debugf("Sending first c2s ping from %s to %s", j.Talk.JID(), j.C.Jabber.Server)

						if err := j.Talk.PingC2S(j.Talk.JID(), j.C.Jabber.Server); err != nil {
							return err
						}

						j.ServerPingTimestampTx = time.Now().Unix()
					}

				// Сервер не анонсировал, что умеет в c2s пинги
				default:
					log.Debug("Sending keepalive whitespace ping")

					if _, err := j.Talk.SendKeepAlive(); err != nil {
						return err
					}
				}
			} else { // Сервер не ответил на disco#info
				log.Debug("Sending keepalive whitespace ping")

				if _, err := j.Talk.SendKeepAlive(); err != nil {
					return err
				}
			}
		}
	}
}

// ProbeMUCLiveness Пингует MUC-и, нужно для проверки, что клиент ещё находится в MUC-е.
func (j *Jabber) ProbeMUCLiveness() { //nolint:gocognit
	var err error

	for {
		for _, room := range j.RoomsConnected {
			var (
				exist          bool
				lastActivityTS interface{}
			)

			// Если записи про комнату нету, то пинговать её бессмысленно.
			if lastActivityTS, exist = j.LastMucActivity.Get(room); !exist {
				continue
			}

			// Если время последней активности в чятике не превысило
			// j.C.Jabber.ServerPingDelay + j.C.Jabber.PingSplayDelay, ничего не пингуем.
			if (time.Now().Unix() - lastActivityTS.(int64)) < (j.C.Jabber.ServerPingDelay + j.C.Jabber.PingSplayDelay) {
				continue
			}

			/* Пинг MUC-а по сценарию без серверной оптимизации мы реализовывать не будем. Это как-то не надёжно.
				j.GTomb.Go(
					func() {
						// Небольшая рандомная задержка перед пингом комнаты
						sleepTime := time.Duration(rand.Int63n(1000*j.C.Jabber.PingSplayDelay)) * time.Millisecond //nolint:gosec
						time.Sleep(sleepTime)

						if e := j.Talk.PingS2S(talk.JID(), room+"/"+j.GetBotNickFromRoomConfig(room)); e != nil {
							err = e
							return err
						}

						return nil
					},
				)

				if err != nil {
					break
				}
			)
			*/
			var roomMap interface{}

			roomMap, exist = j.MucCapsList.Get(room)

			// Пинги комнаты проводим, только если она записана, как прошедшая disco#info и поддерживающая
			// Server Optimization.
			if exist && roomMap.(map[string]bool)["http://jabber.org/protocol/muc#self-ping-optimization"] {
				j.GTomb.Go(
					func() error {
						// Небольшая рандомная задержка перед пингом комнаты.
						sleepTime := time.Duration(rand.Int63n(1000*j.C.Jabber.PingSplayDelay)) * time.Millisecond
						time.Sleep(sleepTime)

						log.Debugf("Sending MUC ping from %s to %s", j.Talk.JID(), room)

						if e := j.Talk.PingS2S(j.Talk.JID(), room); e != nil {
							err = fmt.Errorf("unable to ping MUC %s: %w", room, e)
							return err
						}

						return nil
					},
				)
			}

			if err != nil {
				break
			}
		}

		if err != nil {
			return
		}

		time.Sleep(time.Duration(j.C.Jabber.MucPingDelay) * time.Second)
	}
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
