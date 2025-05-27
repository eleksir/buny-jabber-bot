package jabber

import (
	"os"
	"syscall"

	"github.com/eleksir/go-xmpp"
	log "github.com/sirupsen/logrus"
)

func (j *Jabber) SigHandler() error {
	log.Debug("Installing signal handler")

	for s := range j.SigChan {
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
		j.Shutdown = true

		if j.IsConnected && !j.Shutdown {
			log.Debug("Try to set our presence to Unavailable and status to Offline")

			// Вот тут понадобится коллекция известных пользователей, чтобы им разослать presence, что бот свалил в
			// offline. Пока за неимением лучшего сообщим об этом самим себе.
			for _, room := range j.RoomsConnected {
				if _, err := j.Talk.SendPresence(
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
			log.Infoln("Closing connection to jabber server")

			if err := j.Talk.Close(); err != nil {
				log.Infof("Unable to close connection to jabber server: %s", err)
			}
		}

		os.Exit(0)
	}

	return nil
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
