package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/eleksir/go-xmpp"
	log "github.com/sirupsen/logrus"
)

func cmd(v xmpp.Chat) error {
	var err error

	switch {
	case v.Text == fmt.Sprintf("%shelp", config.CSign) || v.Text == fmt.Sprintf("%sпомощь", config.CSign):
		var (
			answer         string
			chosenOneTalks = false
		)

		realJID := getRealJIDfromNick(v.Remote)

		for _, master := range config.Jabber.BotMasters {
			if (strings.SplitN(realJID, "/", 2))[0] == master {
				chosenOneTalks = true
			}
		}

		if chosenOneTalks {
			answer = fmt.Sprintf("%sпомощь - этот список команд\n", config.CSign)
			answer += fmt.Sprintf("%shelp   - this commands list\n", config.CSign)
			answer += fmt.Sprintf("%srehash - reload white and black lists", config.CSign)
		} else {
			answer = "Ничем помочь не могу. Луна не светит на тебя."
		}

		dest := v.Remote

		if v.Type == "groupchat" {
			dest = (strings.SplitN(v.Remote, "/", 2))[0]
		}

		if _, err := talk.Send(
			xmpp.Chat{ //nolint:exhaustruct
				Remote: dest,
				Text:   strings.TrimSpace(answer),
				Type:   v.Type,
			},
		); err != nil {
			err = fmt.Errorf("unable to send message to %s: %w", v.Remote, err)

			return err
		}

	case v.Text == fmt.Sprintf("%srehash", config.CSign):
		/* Эта команда может прилетать из приватной беседы с realjid-ом, из приватной беседы с chat-nick-ом, а также из
		 * чятика.
		 * Groupchat мы можем определить по v.Type (groupchat) и тогда понятно что делать.
		 * А вот, например, приватные притязания определить несколько сложно-вато. Но можно попробовать, например,
		 * пытаться по списку presence-ов находить персонажа и его real jid и уже из этого определять master он или нет.
		 */

		// Попробуем определить каков алгоритм наших действий, в зависимости от типа сообщения
		switch v.Type {
		case "groupchat":
			// Проверим, является ли "командир" избранным или это самозванец?
			var (
				chosenOneTalks = false
				listsLoaded    = true
			)

			realJID := getRealJIDfromNick(v.Remote)

			for _, master := range config.Jabber.BotMasters {
				if (strings.SplitN(realJID, "/", 2))[0] == master {
					chosenOneTalks = true
				}
			}

			if !chosenOneTalks {
				log.Infof(
					"Command %srehash given by non-bot_master user %s(%s), ignoring",
					config.CSign,
					realJID,
					v.Remote,
				)

				return err
			}

			if err := readWhitelist(); err != nil {
				listsLoaded = false

				var msg xmpp.Chat
				msg.Remote = v.Remote
				msg.Type = "chat" // private message
				msg.Text = fmt.Sprint(err)

				if _, err := talk.Send(msg); err != nil {
					return err
				}
			}

			if err := readBlacklist(); err != nil {
				listsLoaded = false

				var msg xmpp.Chat
				msg.Remote = v.Remote
				msg.Type = "chat" // private message
				msg.Text = fmt.Sprint(err)

				if _, err := talk.Send(msg); err != nil {
					return err
				}
			}

			if listsLoaded {
				room := (strings.SplitN(v.Remote, "/", 2))[0]

				var msg xmpp.Chat
				msg.Remote = room
				msg.Type = v.Type
				msg.Text = "Сделано"

				if _, err := talk.Send(msg); err != nil {
					return err
				}
			}
		case "chat":
			room := (strings.SplitN(v.Remote, "/", 2))[0]

			// Реагировать на приватную команду "из комнаты", только если бот в комнате
			if slices.Contains(roomsConnected, room) {
				var (
					chosenOneTalks = false
					listsLoaded    = true
				)

				// Реагировать на rehash только если собеседник является bot master-ом
				realJID := getRealJIDfromNick(v.Remote)

				for _, master := range config.Jabber.BotMasters {
					if (strings.SplitN(realJID, "/", 2))[0] == master {
						chosenOneTalks = true
					}
				}

				if !chosenOneTalks {
					log.Infof(
						"Command %srehash given by non-bot_master user %s(%s), ignoring",
						config.CSign,
						realJID,
						v.Remote,
					)

					return err
				}

				if err := readWhitelist(); err != nil {
					listsLoaded = false

					var msg xmpp.Chat
					msg.Remote = v.Remote
					msg.Type = v.Type
					msg.Text = fmt.Sprint(err)

					if _, err := talk.Send(msg); err != nil {
						return err
					}
				}

				if err := readBlacklist(); err != nil {
					listsLoaded = false

					var msg xmpp.Chat
					msg.Remote = v.Remote
					msg.Type = v.Type
					msg.Text = fmt.Sprint(err)

					if _, err := talk.Send(msg); err != nil {
						return err
					}
				}

				if listsLoaded {
					var msg xmpp.Chat
					msg.Remote = v.Remote
					msg.Type = v.Type
					msg.Text = "Сделано"

					if _, err := talk.Send(msg); err != nil {
						return err
					}
				}
			}

		default:
			log.Infof("Rehash command from outer space: %s", spew.Sdump(v))
		}

		return err
	default:
		return err
	}

	return err
}
