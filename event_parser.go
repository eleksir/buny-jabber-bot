package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/eleksir/go-xmpp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

// parseEvent парсит ивенты, прилетающие из модуля xmpp (изменения presence, фразы участников, ошибки).
func parseEvent(e interface{}) { //nolint:maintidx,gocognit,gocyclo
	lastServerActivity = time.Now().Unix()

	switch v := e.(type) {
	// Сообщение в чяти
	case xmpp.Chat:
		log.Debugf("Looks like message, ChatType: %s, From: %s, Subject: %s Text: %s",
			v.Type, v.Remote, v.Subject, v.Text)

		// Топик чятика присылается в виде сообщения с subject, но без text
		// В то же время сообщения от людей приходят с пустым subject, но с заполненным text
		if v.Text != "" {
			// Чятики бывают групповые и не групповые, от этого зависит Remote, куда направлять сообщение
			switch v.Type { //nolint:wsl,whitespace

			// Групповой чятик
			case "groupchat":
				log.Debugf("Message from public chat: %s", v.Text)
				// dest := strings.SplitN(v.Remote, "/", 2)[0]
				nick := strings.SplitN(v.Remote, "/", 2)[1]

				if nick == config.Jabber.Nick {
					log.Debug("Skipping message from myself")

					return
				}

				if err := cmd(v); err != nil {
					gTomb.Kill(err)

					return
				}

				lastActivity = lastServerActivity

				if muc, _ := strings.CutSuffix(v.Remote, "/"); muc != "" {
					lastMucActivity.Set(muc, lastServerActivity)
				}

			// Приватный чятик
			case "chat":
				// Здесь у нас может быть 2 вида From:
				//  slackware-current@conference.jabber.ru/eleksir - это если сообщение из публичного чятика
				//  eleksir@jabber.ru/array.lan - это если мы работаем через ростер
				log.Debugf("Private message: %s", v.Text)

				if err := cmd(v); err != nil {
					gTomb.Kill(err)

					return
				}

				lastActivity = lastServerActivity

			// Внезапно, ошибки. По идее они должны ассоциироваться с отправляемыми сообщениями, но по ходу это
			// не реализовано, поэтому мы получаем ошибки в форме отдельного чятика
			case "error":
				// Это ошибка со стороны сервера, видимо, в ответ на наши действия и поскольку такие вещи происходят
				// асинхронно, связать их с конкретными нашими действиями довольно сложно.
				log.Warn(spew.Sdump(e))
			}
		}

	// IQ-сообщение - пинг, понг, что-то ещё...
	case xmpp.IQ:
		// По правилам IQ обязательно должна содержать ID, причём не пустой
		if v.ID == "" {
			log.Info("Got an IQ stanza with empty id, discarding")

			return
		}

		switch v.Type {
		case xmpp.IQTypeGet:
			log.Debug("Looks like IQ get query")

			if muc, _ := strings.CutSuffix(v.To, "/"); muc != "" {
				if slices.Contains(roomsConnected, muc) {
					lastMucActivity.Set(muc, lastServerActivity)
				}
			}

			var (
				iqStruct   jabberSimpleIqGetQuery
				parseError bool
			)

			if err := xml.Unmarshal(v.Query, &iqStruct); err == nil {
				switch iqStruct.Xmlns { //nolint:wsl,whitespace

				// Запрос номера версии приложения
				case "jabber:iq:version":
					log.Infof("Got IQ get request for version from %s", v.From)

					if id, err := talk.IqVersionResponse(
						v,
						"buny-jabber-bot",
						"1.с_чем-то",
						"линупс",
					); err != nil {
						err := fmt.Errorf(
							"unable to send version info to jabber server: id=%s, err=%w",
							id,
							err,
						)

						gTomb.Kill(err)

						return
					}

					parseError = false

				// Нам прислали попингуй
				case "urn:xmpp:ping":
					log.Infof("Got IQ get request for pong from %s", v.From)

					if id, err := talk.PingResponse(v); err != nil {
						err := fmt.Errorf(
							"unable to send pong to jabber server: id=%s, err=%w",
							id,
							err,
						)

						gTomb.Kill(err)

						return
					}

					parseError = false

				// У нас запросили время последней активности. В нашем случае это либо время запуска клиента, либо время
				// последней фразы в чяти. Xep-0012.
				case "jabber:iq:last":
					log.Infof("Got IQ get last activity time request from %s", v.From)

					if id, err := talk.JabberIqLastResponse(v, lastActivity); err != nil {
						err := fmt.Errorf(
							"unable to send last activity time to jabber server: id=%s, err=%w",
							id,
							err,
						)

						gTomb.Kill(err)

						return
					}

					parseError = false

				// Запрос на список поддерживаемых фич - версия, локальное время, итд
				case xmpp.XMPPNS_DISCO_INFO:
					log.Infof("Got IQ get disco#info request from %s", v.From)

					answer := "<query xmlns=\"http://jabber.org/protocol/disco#info\">"
					answer += "<feature var=\"jabber:iq:version\" />"
					answer += "<feature var=\"urn:xmpp:time\" />"
					answer += "<feature var=\"urn:xmpp:ping\" />"
					answer += "<feature var=\"jabber:iq:last\" />"
					answer += "<feature var=\"http://jabber.org/protocol/disco#info\" />"
					answer += "</query>"

					if id, err := talk.RawInformation(
						v.To,
						v.From,
						v.ID,
						xmpp.IQTypeResult,
						answer,
					); err != nil {
						err := fmt.Errorf(
							"unable to send disco#info to jabber server: id=%s, err=%w",
							id,
							err,
						)

						gTomb.Kill(err)

						return
					}

					parseError = false

				// Спрашивают список команд, чтобы поуправлять этим клиентом (xep-0030)
				case xmpp.XMPPNS_DISCO_ITEMS:
					log.Infof("Got IQ get disco#items request from %s, answer service unavailable", v.From)

					if _, err := talk.ErrorServiceUnavailable(
						v,
						"http://jabber.org/protocol/disco#info",
						"http://jabber.org/protocol/commands",
					); err != nil {
						err := fmt.Errorf(
							"unable to send disco#items to jabber server: id=%s, err=%w",
							v.ID,
							err,
						)

						gTomb.Kill(err)

						return
					}

					parseError = false

				// Запрашивают что-то, о чём мы не имеем представления
				default:
					log.Info("Got an unknown IQ get request, discarding")
					log.Info(spew.Sdump(e))

					return
				}
			} else {
				parseError = true
			}

			// Попробуем распарсить входящий запрос как urn:xmpp:time
			if parseError {
				var iqStruct jabberTimeIqGetQuery

				if err := xml.Unmarshal(v.Query, &iqStruct); err == nil {
					switch iqStruct.Xmlns {
					case "urn:xmpp:time":
						log.Infof("Got IQ get time request from %s", v.From)

						if id, err := talk.UrnXMPPTimeResponse(v, "+00:00"); err != nil {
							err := fmt.Errorf(
								"unable to send urn:xmpp:time to jabber server: id=%s, err=%w",
								id,
								err,
							)

							gTomb.Kill(err)

							return
						}

						parseError = false

					default:
						log.Info("Got bogus IQ get time request, discarding")
						log.Info(spew.Sdump(e))

						return
					}
				} else {
					parseError = true
				}
			}

			// Попробуем распарсить входящий запрос как http://jabber.org/protocol/pubsub
			if parseError {
				var iqStruct jabberPubsubIQGetQuery

				if err := xml.Unmarshal(v.Query, &iqStruct); err == nil {
					switch iqStruct.Xmlns {
					case xmpp.XMPPNS_PUBSUB:
						log.Infof("Got IQ get pubsub request from %s, answer feature unimplemented", v.From)

						if id, err := talk.ErrorNotImplemented(
							v,
							"http://jabber.org/protocol/pubsub#errors",
							"subscribe",
						); err != nil {
							err := fmt.Errorf("unable to send pubsub feature unimplemented to jabber server: id=%s, err=%w",
								id,
								err,
							)

							gTomb.Kill(err)

							return
						}

						parseError = false

					default:
						log.Info("Got unknown IQ get pubsub something request, discarding")
						log.Info(spew.Sdump(e))
						parseError = true //nolint:wsl
					}
				} else {
					parseError = true
				}
			}

			// Попробуем распарсить входящий запрос как urn:xmpp:ping он приходит при MUC Self-Ping (Schrödinger's Chat)
			// Предполагается, что такие респонсы должны приходить только для пинга без серверной оптимизации.
			// Хотя идейно мы не поддерживаем работу без серверной оптимизации, но на пинг ответим, нам несложно.
			if parseError {
				var iqStruct jabberIqPing

				if err := xml.Unmarshal(v.Query, &iqStruct); err == nil {
					log.Debugf("Got IQ get request (actually, response) for MUC Self-Ping from %s", v.From)

					if id, err := talk.PingResponse(v); err != nil {
						err := fmt.Errorf(
							"unable to send pong to jabber server: id=%s, err=%w",
							id,
							err,
						)

						gTomb.Kill(err)

						return
					}
				}

				parseError = false
			}

			// Не знаю, как парсить... залоггируем это дело и пойдём дальше
			if parseError {
				log.Infof("Does not look like parsable via iqStruct")
				log.Info(spew.Sdump(e))
			}

		case xmpp.IQTypeResult:
			if muc, _ := strings.CutSuffix(v.To, "/"); muc != "" {
				if slices.Contains(roomsConnected, muc) {
					lastMucActivity.Set(muc, lastServerActivity)
				}
			}

			switch {
			// Похоже на pong от сервера (по стандарту в ответе нету query, но go-xmpp нам подсовывает это)
			case v.From == config.Jabber.Server && v.To == talk.JID() && string(v.Query) == "<XMLElement></XMLElement>":
				log.Debugf("Got S2C pong answer from %s to %s", v.From, v.To)
				serverPingTimestampRx = time.Now().Unix() //nolint:wsl

			// Похоже на понг второй стадии xep-0410 MUC-Ping-а, который у нас не реализован
			case v.To == talk.JID() && string(v.Query) == "<XMLElement></XMLElement>":
				mucNameMatch := slices.Contains(roomsConnected, v.From)

				if mucNameMatch {
					log.Debugf("Got server-optimized MUC pong answer (xep-0410) from %s to %s", v.From, v.To)
				} else {
					mucNickMatch := false

					for _, room := range roomsConnected {
						mucNick := fmt.Sprintf("%s/%s", room, config.Jabber.Nick)

						if v.From == mucNick {
							mucNickMatch = true

							break
						}
					}

					if mucNickMatch {
						log.Debugf("Got 2-nd stage MUC pong answer (xep-0410) from %s to %s", v.From, v.To)
						// Поскольку никакой логики у нас на этот счёт не предусмотрено, то просто пропускаем ответ
					} else {
						log.Infof("Got unknown pong from %s to %s", v.From, v.To)
						log.Debug(spew.Sdump(e))
					}
				}
			default:
				log.Info("Got an IQ result. Dunno how deal with it, discarding")
				log.Debug(spew.Sdump(e))
			}

		// Этот бот не управляется со стороны сервера, поэтому все попытки порулить игнорируем
		case xmpp.IQTypeSet:
			if muc, _ := strings.CutSuffix(v.To, "/"); muc != "" {
				if slices.Contains(roomsConnected, muc) {
					lastMucActivity.Set(muc, lastServerActivity)
				}
			}

			log.Info("Got an IQ request for set something. Answer not implemented")

			if id, err := talk.ErrorNotImplemented(
				v,
				"http://jabber.org/protocol/commands",
				xmpp.IQTypeSet,
			); err != nil {
				err := fmt.Errorf(
					"unable to send set feature unimplemented to jabber server: id=%s, err=%w",
					id,
					err,
				)

				gTomb.Kill(err)

				return
			}

			log.Debug(spew.Sdump(e))

		// Нам прилетело сообщение об ошибке
		case xmpp.IQTypeError:
			// Если сервер не хочет пинговаться и отвечает ошибкой на пинг, то наверно он не умеет в пинги,
			// хотя если мы его пингуем, значит он анонсировал такой capability. Вот, засранец!
			var (
				iqPingStruct jabberIqPing
				parseError   = true
			)

			if err := xml.Unmarshal(v.Query, &iqPingStruct); err == nil {
				if iqPingStruct.Xmlns == "urn:xmpp:ping" {
					if v.From == config.Jabber.Server {
						msg := "Server announced that it can answer c2s ping, but gives us an error to such query, "
						msg += "fallback to keepalive whitespace pings"
						log.Error(msg)

						serverCapsList.Set("urn:xmpp:ping", false)
					} else {
						log.Errorf("Got 'ping unsupported' message from: %s to: %s", v.From, v.To)
					}
				} else {
					log.Error("Iq parsed as ping, but does not belong to xmlns urn:xmpp:ping")
					log.Error(spew.Sdump(e))
				}

				parseError = false
			}

			// Это у нас пинг xep-0410 и мы не в комнате, предполагается, что надо бы заджойниться
			if parseError {
				var iqErrorCancelNotAcceptable jabberIqErrorCancelNotAcceptable

				if err := xml.Unmarshal(v.Query, &iqErrorCancelNotAcceptable); err == nil {
					if v.To == talk.JID() {
						nick := strings.SplitN(v.From, "/", 2)[1]

						if slices.Contains(roomsConnected, iqErrorCancelNotAcceptable.By) && nick == config.Jabber.Nick {
							log.Errorf(
								"Got Iq error message from: %s to: %s. Looks like i'm not in MUC anymore",
								v.From, v.To,
							)

							time.Sleep(time.Duration(config.Jabber.MucRejoinDelay) * time.Second)

							if _, err := talk.JoinMUCNoHistory(iqErrorCancelNotAcceptable.By, nick); err != nil {
								err := fmt.Errorf(
									"looks like connection to server also lost err=%w",
									err,
								)

								gTomb.Kill(err)

								return
							}
						} else {
							log.Error(
								"looks like message parsed as jabberIqErrorCancelNotAcceptable but we're not in given room",
							)

							log.Error(spew.Sdump(e))
						}
					} else {
						log.Error(
							"Looks like message parsed as jabberIqErrorCancelNotAcceptable but addressed not to us",
						)

						log.Error(spew.Sdump(e))
					}

					parseError = false
				}
			}

			if parseError {
				log.Error("Unhandled IQ Error message")
				log.Error(spew.Sdump(e))
			}
		// Нам прилетело что-то неизвестное из семейства IQ stanza
		default:
			log.Info("Got an unknown IQ request. Dunno how deal with it, discarding")
			log.Info(spew.Sdump(e))
		}

	// Смена статуса участника
	case xmpp.Presence:
		if muc, _ := strings.CutSuffix(v.To, "/"); muc != "" {
			if slices.Contains(roomsConnected, muc) {
				lastMucActivity.Set(muc, lastServerActivity)
			}
		}

		if v.Type == xmpp.IQTypeError {
			// Это событие происходит, когда из чятика выходит другой инстанс клиента.
			// Такое бывает, когда 2 инстанса с одинаковым ресурсом, например, начинают "драться" за возможность
			// остаться на сервере. Ситуация, которую допускать нельзя, на самом деле, потому что рано или поздно такого
			// клиента забанят из-за спама "вошёл-вышел".
			// Однако, это не значит, что такую ситуацию мы не должны корректным образом обрабатывать.
			if v.Type == "unavailable" {
				// Считаем, что мы больше не в комнате, поэтому не знаем, кто там есть
				roomPresences.Delete(v.From)

				log.Error("Presence notification - looks like another instance of client leaves room")

				if slices.Contains(roomsConnected, v.From) {
					go joinMuc(v.From)
				}
			} else {
				log.Errorf("Presence notification, Type: %s, From: %s, Show: %s, Status: %s",
					v.Type, v.From, v.Show, v.Status)
				log.Errorf(spew.Sdump(v))
			}
		} else {
			log.Infof(
				"Presence notification, Type: %s, From: %s, To: %s Show: %s, Status: %s, Affiliation: %s, Role: %s, JID: %s",
				v.Type, v.From, v.To, v.Show, v.Status, v.Affiliation, v.Role, v.JID,
			)

			room := strings.SplitN(v.From, "/", 2)[0]
			nick := strings.SplitN(v.From, "/", 2)[1]

			if nick == "" {
				log.Infof("Presence stanza contains incorrect from attribute: %s", v.From)

				return
			}

			// Это наш собственный Presence
			if v.Show == "" && v.Status == "" {
				if nick == config.Jabber.Nick {
					roomsConnected = append(roomsConnected, room)
					// На всякий случай дедуплицируем список комнат, к которым мы заджойнились.
					sort.Strings(roomsConnected)
					slices.Compact(roomsConnected)
				}
			}

			switch v.Role {
			// Участник ушёл
			case "none":
				if presenceJSONInterface, present := roomPresences.Get(room); present {
					presenceJSONStrings := interfaceToStringSlice(presenceJSONInterface)
					var newPresenceJSONStrings []string

					for _, presenceJSONstring := range presenceJSONStrings {
						var p xmpp.Presence
						_ = json.Unmarshal([]byte(presenceJSONstring), &p)

						if p.From == v.From {
							continue
						}

						newPresenceJSONStrings = append(newPresenceJSONStrings, presenceJSONstring)
					}

					roomPresences.Set(room, newPresenceJSONStrings)
				}
			// Участник пришёл
			default:
				var presenceJSONStrings []string
				var newPresenceJSONStrings []string
				var presenceJSONBytes []byte

				if presenceJSONInterface, present := roomPresences.Get(room); present {
					presenceJSONStrings = interfaceToStringSlice(presenceJSONInterface)
				}

				for _, presenceJSONString := range presenceJSONStrings {
					var p xmpp.Presence
					_ = json.Unmarshal([]byte(presenceJSONString), &p)

					// Если находим, что у нас уже есть клиент с таким же From, то есть полным nick-ом (для grouchat)
					// просто замещаем его.
					if p.From == v.From {
						continue
					}

					newPresenceJSONStrings = append(newPresenceJSONStrings, presenceJSONString)
				}

				presenceJSONBytes, _ = json.Marshal(v) //nolint:errchkjson
				newPresenceJSONStrings = append(newPresenceJSONStrings, string(presenceJSONBytes))
				roomPresences.Set(room, newPresenceJSONStrings)
			}

			// Проверяем, а не злодей ли зашёл? Сделать это мы можем, только если мы находимся в комнате.
			// По правилам, мы можем что-то делать, только после того, как нам прилетит наш собственный presence, это
			// значит, что мы вошли в комнату.
			if slices.Contains(roomsConnected, room) {
				if err := buny(v); err != nil {
					gTomb.Kill(err)

					return
				}
			}
		}

	// Ответ на запрос поддерживаемых фич, который "http://jabber.org/protocol/disco#info"
	case xmpp.DiscoResult:
		if muc, _ := strings.CutSuffix(v.To, "/"); muc != "" {
			if slices.Contains(roomsConnected, muc) {
				lastMucActivity.Set(muc, lastServerActivity)
			}
		}

		// Я видел 2 типа disco result и они отличались только []identities. Попробуем вытащить известный identity
		// TODO: проверять адресата v.To
		for _, ident := range v.Identities {
			switch ident.Category {
			case "server":
				// Конкретно сейчас нас интересует только поддержка c2s ping
				for _, feature := range v.Features {
					log.Debugf("Server %s announced that it supports feature: %s", v.From, feature)
					serverCapsList.Set(feature, true)
				}

				serverCapsQueried = true

			case "conference":
				mucCaps := make(map[string]bool)

				for _, feature := range v.Features {
					log.Debugf("MUC %s announced that it supports feature: %s", v.From, feature)
					mucCaps[feature] = true //nolint:wsl
				}

				mucCapsList.Set(v.From, mucCaps)

			case "pubsub":
				log.Debugf("PubSub component %s reply to disco#info, skipping", v.From)

			default:
				log.Debug("Got unknown reply to disco#info")
				log.Debug(spew.Sdump(e))
			}
		}

	// Это что-то неизвестное, подампим событие в лог
	default:
		log.Info(spew.Sdump(e))
	}
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
