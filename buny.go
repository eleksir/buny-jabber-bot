package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/eleksir/go-xmpp"
	log "github.com/sirupsen/logrus"
)

// buny производит проверку по бело-чёрным спискам. Если presence пришёл от злодея (из чёрного списка), то отправляет
// его в бан.
func buny(v xmpp.Presence) error { //nolint:gocognit,gocyclo
	var err error

	// Если у presence-а есть JID и presence из одной из комнат, в которой мы есть и если его домен в чёрном
	// списке, заносим пидора в список outcast-ов
	if v.JID != "" {
		// На всякий случай: себя никогда не баним, явным образом
		if v.JID == talk.JID() {
			return err
		}

		room := strings.SplitN(v.From, "/", 2)[0]

		if room == "" {
			log.Infof("We got empty room field in presence event, which kinda strange: %s", spew.Sdump(v))

			return err
		}

		goodJid := strings.SplitN(v.JID, "/", 2)[0]

		// Обрабатываем правила белого списка
		for _, good := range whiteList.Whitelist {
			// Глобальный белый список
			if good.RoomName == "" {
				for _, person := range good.Jid {
					if person == goodJid {
						return err
					}
				}
			}

			// Белый список для конкретных каналов
			if good.RoomName == room {
				for _, person := range good.Jid {
					if person == goodJid {
						return err
					}
				}
			}
		}

		evilJid := strings.SplitN(v.JID, "/", 2)[0]
		evilNick := ""
		evilNicks := strings.SplitN(v.From, "/", 2)

		// Давайте-ка похэндлим нонсенс - когда у нас в строке нету разделителя
		if len(evilNicks) > 1 {
			evilNick = evilNicks[1]
		}

		// Обрабатываем правила чёрного списка
		for _, cRoom := range roomsConnected {
			if cRoom == room {
				for _, bEntry := range blackList.Blacklist {
					// Обработаем правила глобального чёрного списка
					if bEntry.RoomName == "" {
						for _, jidRegexp := range bEntry.JidRe {
							if jidRegexp == "" {
								continue
							}

							re, err := regexp.Compile(jidRegexp)

							if err != nil {
								log.Errorf("Incorrect regexp in global blacklist: %s, skipping", jidRegexp)

								continue
							}

							log.Debugf("Checking jid %s vs global blacklist regex %s", evilJid, jidRegexp)

							if re.MatchString(evilJid) {
								if id, err := squash(room, evilJid, bEntry.ReasonEnable, v.Type); err != nil {
									err := fmt.Errorf(
										"unable to ban user: id=%s, err=%w",
										id,
										err,
									)

									gTomb.Kill(err)

									continue
								}

								return err
							}
						}

						for _, nickRegexp := range bEntry.NickRe {
							if nickRegexp == "" {
								continue
							}

							re, err := regexp.Compile(nickRegexp)

							if err != nil {
								log.Errorf("Incorrect regexp in global blacklist: %s, skipping", nickRegexp)

								continue
							}

							log.Debugf("Checking nick %s vs global blacklist regex %s", evilNick, nickRegexp)

							if re.MatchString(evilNick) {
								// Баним именно jid
								if id, err := squash(room, evilJid, bEntry.ReasonEnable, v.Type); err != nil {
									err := fmt.Errorf(
										"unable to ban user: id=%s, err=%w",
										id,
										err,
									)

									gTomb.Kill(err)

									continue
								}

								return err
							}
						}

						continue
					}

					// Обработаем правила конкретного канала, комнаты конференции
					if bEntry.RoomName == room {
						for _, jidRegexp := range bEntry.JidRe {
							if jidRegexp == "" {
								continue
							}

							re, err := regexp.Compile(jidRegexp)

							if err != nil {
								log.Errorf("Incorrect regexp in room %s blacklist: %s, skipping", room, jidRegexp)

								continue
							}

							log.Debugf("Checking jid %s vs room %s blacklist regex %s", evilJid, room, jidRegexp)

							if re.MatchString(evilJid) {
								if id, err := squash(room, evilJid, bEntry.ReasonEnable, v.Type); err != nil {
									err := fmt.Errorf(
										"unable to ban user: id=%s, err=%w",
										id,
										err,
									)

									gTomb.Kill(err)

									continue
								}

								return err
							}
						}

						if bEntry.RoomName == room {
							for _, nickRegexp := range bEntry.NickRe {
								if nickRegexp == "" {
									continue
								}

								re, err := regexp.Compile(nickRegexp)

								if err != nil {
									log.Errorf("Incorrect regexp in room %s blacklist: %s, skipping", room, nickRegexp)

									continue
								}

								log.Debugf("Checking nick %s vs room %s blacklist regex %s", evilJid, room, nickRegexp)

								if re.MatchString(evilNick) {
									// Баним именно jid
									if id, err := squash(room, evilJid, bEntry.ReasonEnable, v.Type); err != nil {
										err := fmt.Errorf(
											"unable to ban user: id=%s, err=%w",
											id,
											err,
										)

										gTomb.Kill(err)

										continue
									}

									return err
								}
							}
						}

						continue
					}
				}
			}
		}
	}

	return err
}

// squash банит указанный jid в указанной комнате.
// reasonEnable указывает, надо ли писать дату автобана в банлисте в поле reason (это единственная причина, в которую
// умеет бот).
func squash(room, jid string, reasonEnable bool, vType string) (string, error) {
	var (
		id  string
		err error
	)

	log.Infof("Hammer falls on %s", jid)

	if config.Jabber.BanPhrasesEnable {
		phrase := randomPhrase(config.Jabber.BanPhrases)

		if _, err = talk.Send(
			xmpp.Chat{ //nolint:exhaustruct
				Remote: room,
				Text:   phrase,
				Type:   vType,
			},
		); err != nil {
			err = fmt.Errorf("unable to send phrase to room %s: %w", room, err)

			// Здесь возвращаем nil, т.к. за нас ошибку залоггирует код выше
			return id, err
		}
	}

	// https://xmpp.org/extensions/xep-0045.html#ban баним вот таким сообщением
	ban := "<item affiliation='outcast' jid='" + jid + "'>"

	if reasonEnable {
		var t = time.Now()
		ban += fmt.Sprintf(
			"<reason>autoban at %04d.%02d.%02d %02d:%02d:%02d</reason>",
			t.Year(),
			t.Month(),
			t.Day(),
			t.Hour(),
			t.Minute(),
			t.Second(),
		)
	} else {
		ban += "<reason />"
	}

	ban += "</item>"

	if id, err = talk.RawInformationQuery(
		talk.JID(),
		room,
		"ban1",
		xmpp.IQTypeSet,
		"http://jabber.org/protocol/muc#admin",
		ban,
	); err != nil {
		err = fmt.Errorf(
			"unable to ban user: id=%s, err=%w",
			id,
			err,
		)
	}

	return id, err
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
