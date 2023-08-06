package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/eleksir/go-xmpp"
	log "github.com/sirupsen/logrus"
)

func buny(v xmpp.Presence) error { //nolint:gocognit
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

		// Обрабатываем правила чёрного списка
		for _, cRoom := range roomsConnected {
			if cRoom == room {
				for _, bEntry := range blackList.Blacklist {
					// Обработаем правила глобального чёрного списка
					if bEntry.RoomName == "" {
						for _, jidRegexp := range bEntry.JidRe {
							re, err := regexp.Compile(jidRegexp)

							if err != nil {
								log.Errorf("Incorrect regexp in blacklist: %s, skipping", jidRegexp)

								continue
							}

							if re.Match([]byte(evilJid)) {
								log.Infof("Hammer falls on %s", v.JID)
								// https://xmpp.org/extensions/xep-0045.html#ban баним вот таким сообщением
								ban := "<item affiliation='outcast' jid='" + evilJid + "'>"
								ban += "<reason />"
								ban += "</item>"

								if id, err := talk.RawInformationQuery(
									talk.JID(),
									room,
									"ban1",
									xmpp.IQTypeSet,
									"http://jabber.org/protocol/muc#admin",
									ban,
								); err != nil {
									err := errors.New(
										fmt.Sprintf(
											"Unable to ban user: id=%s, err=%s",
											id,
											err,
										),
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
							re, err := regexp.Compile(jidRegexp)

							if err != nil {
								log.Errorf("Incorrect regexp in blacklist: %s, skipping", jidRegexp)

								continue
							}

							if re.Match([]byte(evilJid)) {
								log.Infof("Hammer falls on %s", v.JID)
								// https://xmpp.org/extensions/xep-0045.html#ban баним вот таким сообщением
								ban := "<item affiliation='outcast' jid='" + evilJid + "'>"
								ban += "<reason />"
								ban += "</item>"

								if id, err := talk.RawInformationQuery(
									talk.JID(),
									room,
									"ban1",
									xmpp.IQTypeSet,
									"http://jabber.org/protocol/muc#admin",
									ban,
								); err != nil {
									err := errors.New(
										fmt.Sprintf(
											"Unable to ban user: id=%s, err=%s",
											id,
											err,
										),
									)

									gTomb.Kill(err)

									continue
								}

								return err
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

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
