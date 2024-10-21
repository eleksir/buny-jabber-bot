package jabber

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/eleksir/go-xmpp"
	log "github.com/sirupsen/logrus"
)

// BunyPresense производит проверку по бело-чёрным спискам. Если presence пришёл от злодея (из чёрного списка), то
// отправляет его в бан.
func (j *Jabber) BunyPresense(v xmpp.Presence) error { //nolint:gocognit,gocyclo
	var err error

	// Если у presence-а есть JID и presence из одной из комнат, в которой мы есть и если его домен в чёрном
	// списке, заносим пидора в список outcast-ов
	if v.JID != "" {
		// На всякий случай: себя никогда не баним, явным образом
		if v.JID == j.Talk.JID() {
			return err
		}

		room := strings.SplitN(v.From, "/", 2)[0]

		if room == "" {
			log.Infof("We got empty room field in presence event, which kinda strange: %s", spew.Sdump(v))

			return err
		}

		goodJid := strings.SplitN(v.JID, "/", 2)[0]

		// Обрабатываем правила белого списка
		for _, good := range j.WhiteList.Whitelist {
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
		for _, cRoom := range j.RoomsConnected {
			if cRoom == room {
				for _, bEntry := range j.BlackList.Blacklist {
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

							log.Debugf("Checking jid %s vs global blacklist regex %s", v.JID, jidRegexp)

							if re.MatchString(v.JID) {
								log.Warnf(
									"Hammer falls on %s (%s): jid matches with global blacklist entry: %s",
									v.From,
									evilJid,
									jidRegexp,
								)

								id, err := j.Squash(room, evilJid, bEntry.ReasonEnable, v.Type)

								if err != nil {
									err := fmt.Errorf(
										"unable to ban user: id=%s, err=%w",
										id,
										err,
									)

									j.GTomb.Kill(err)
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
								log.Warnf(
									"Hammer falls on %s (%s): nick matches with global blacklist entry: %s",
									v.From,
									evilJid,
									nickRegexp,
								)

								// Баним именно jid
								id, err := j.Squash(room, evilJid, bEntry.ReasonEnable, v.Type)

								if err != nil {
									err := fmt.Errorf(
										"unable to ban user: id=%s, err=%w",
										id,
										err,
									)

									j.GTomb.Kill(err)
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

							log.Debugf("Checking jid %s vs room %s blacklist regex %s", v.JID, room, jidRegexp)

							if re.MatchString(v.JID) {
								log.Warnf(
									"Hammer falls on %s (%s): jid matches with room blacklist entry: %s",
									v.From,
									evilJid,
									jidRegexp,
								)

								id, err := j.Squash(room, evilJid, bEntry.ReasonEnable, v.Type)

								if err != nil {
									err := fmt.Errorf(
										"unable to ban user: id=%s, err=%w",
										id,
										err,
									)

									j.GTomb.Kill(err)
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
									log.Warnf(
										"Hammer falls on %s (%s): nick matches with room blacklist entry: %s",
										v.From,
										evilJid,
										nickRegexp,
									)

									// Баним именно jid
									id, err := j.Squash(room, evilJid, bEntry.ReasonEnable, v.Type)

									if err != nil {
										err := fmt.Errorf(
											"unable to ban user: id=%s, err=%w",
											id,
											err,
										)

										j.GTomb.Kill(err)
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

// BunyChat производит проверку сообщений участников чата по списку забаненных фраз и в случае нахождения запрещённого
// шаблона банит участника чата.
func (j *Jabber) BunyChat(v xmpp.Chat) error {
	var (
		room = (strings.SplitN(v.Remote, "/", 2))[0]
		// nick = (strings.SplitN(v.Remote, "/", 2))[1]
		err error
	)

	// Действовать мы можем только в рамках тех комнат, где явно присуствуем.
	for _, cRoom := range j.RoomsConnected {
		if cRoom == room {
			// Перебирём правила чёрных списков.
			for _, bEntry := range j.BlackList.Blacklist {
				// Обработаем правила глобального чёрного списка
				if bEntry.RoomName == "" {
					for _, phraseRegexp := range bEntry.PhraseRe {
						if phraseRegexp == "" {
							continue
						}

						re, err := regexp.Compile(phraseRegexp)

						if err != nil {
							log.Errorf("Incorrect regexp in room %s blacklist: %s, skipping", room, phraseRegexp)

							continue
						}

						log.Debugf("Checking phrase %s vs room %s blacklist regex %s", v.Text, room, phraseRegexp)

						if re.MatchString(v.Text) {
							realJID := j.GetRealJIDfromNick(v.Remote)

							log.Warnf(
								"Hammer falls on %s (%s): phrase matches with global blacklist entry: %s vs %s",
								v.Remote,
								realJID,
								v.Text,
								phraseRegexp,
							)

							if id, err := j.Squash(room, realJID, bEntry.ReasonEnable, v.Type); err != nil {
								err := fmt.Errorf(
									"unable to ban user: id=%s, err=%w",
									id,
									err,
								)

								j.GTomb.Kill(err)
							}

							return err
						}
					}
				}

				// Обработаем правила чёрного списка для конкретной комнаты
				if bEntry.RoomName == room {
					for _, phraseRegexp := range bEntry.PhraseRe {
						if phraseRegexp == "" {
							continue
						}

						re, err := regexp.Compile(phraseRegexp)

						if err != nil {
							log.Errorf("Incorrect regexp in room %s blacklist: %s, skipping", room, phraseRegexp)

							continue
						}

						log.Debugf("Checking phrase %s vs room %s blacklist regex %s", v.Text, room, phraseRegexp)

						if re.MatchString(v.Text) {
							realJID := j.GetRealJIDfromNick(v.Remote)

							log.Warnf(
								"Hammer falls on %s (%s): phrase matches with room blacklist entry: %s vs %s",
								v.Remote,
								realJID,
								v.Text,
								phraseRegexp,
							)

							if id, err := j.Squash(room, realJID, bEntry.ReasonEnable, v.Type); err != nil {
								err := fmt.Errorf(
									"unable to ban user: id=%s, err=%w",
									id,
									err,
								)

								j.GTomb.Kill(err)
							}

							return err
						}
					}
				}
			}

			// Если включено, проверяем фразу на КАПС.
			for _, channel := range j.C.Jabber.Channels {
				if channel.Name == room && channel.AllCaps.Enabled {
					// Нормализуем строку и вырежем из неё пробелы
					normPhrase := strings.ReplaceAll(nString(v.Text), " ", "")

					// Проверяем согласно тому, что длина фразы более чем сколько-то символов
					if len(normPhrase) >= channel.AllCaps.MinLength {
						normPhraseUpper := strings.ReplaceAll(nStringUpper(v.Text), " ", "")

						if normPhrase == normPhraseUpper {
							realJID := j.GetRealJIDfromNick(v.Remote)
							id, err := j.Squash(room, realJID, false, v.Type)

							if err != nil {
								err := fmt.Errorf(
									"unable to ban user: id=%s, err=%w",
									id,
									err,
								)

								j.GTomb.Kill(err)
							}
						}
					}
				}
			}

			break
		}
	}

	return err
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
