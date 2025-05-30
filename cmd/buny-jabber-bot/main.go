package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"time"

	_ "embed"

	"buny-jabber-bot/internal/jabber"

	"github.com/eleksir/go-xmpp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/tomb.v2"
)

//go:embed version
var version string

// main - фактичеcки, начало и основное тело программы.
func main() {
	var err error

	for {
		var j = jabber.Jabber{ //nolint:exhaustruct
			SigChan:        make(chan os.Signal, 1),
			GTomb:          tomb.Tomb{},
			RoomsConnected: make([]string, 1),
		}

		log.SetFormatter(&log.TextFormatter{ //nolint:exhaustruct
			DisableQuote:           true,
			DisableLevelTruncation: false,
			DisableColors:          true,
			FullTimestamp:          true,
			TimestampFormat:        "2006-01-02 15:04:05",
		})

		if err := j.ReadConfig(); err != nil {
			log.Error(err)

			os.Exit(1)
		}

		j.C.Version = version

		// no panic
		switch j.C.Loglevel {
		case "fatal":
			log.SetLevel(log.FatalLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "info":
			log.SetLevel(log.InfoLevel)
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "trace":
			log.SetLevel(log.TraceLevel)
		default:
			log.SetLevel(log.InfoLevel)
		}

		// Откроем лог и скормим его логгеру.
		if j.C.Log != "" {
			logfile, err := os.OpenFile(j.C.Log, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

			if err != nil {
				log.Fatalf("Unable to open log file %s: %s", j.C.Log, err)
			}

			log.SetOutput(logfile)
		}

		j.C.ExeName, err = os.Executable()

		if err != nil {
			log.Panicf("Unable to find my executable: %s", err)
		}

		// handle pricky windows case. But honestly i do not expect it runs on windows.
		if regexp.MustCompile(".[Ee][Xx][Ee]$").MatchString(j.C.ExeName) {
			j.C.ExeName = j.C.ExeName[:len(j.C.ExeName)-4]

			if j.C.ExeName == "" {
				j.C.ExeName = "buny-jabber-bot"
			}
		}

		log.Infof("Service %s v%s starting", j.C.ExeName, j.C.Version)

		myLogLevel := log.GetLevel()
		log.Warnf("Loglevel set to %v", myLogLevel)

		verboseClient := false

		if myLogLevel == log.TraceLevel {
			verboseClient = true
		}

		// Самое время вгрузить белые и чёрные списки, если что - получим ошибку в лог.
		if err := j.ReadWhitelist(); err != nil {
			log.Error(err)

			os.Exit(1)
		}

		if err := j.ReadBlacklist(); err != nil {
			log.Error(err)

			os.Exit(1)
		}

		// github.com/mattn/go-xmpp пишет в stdio, нам этого не надо, ловим выхлоп его в logrus с уровнем trace.
		xmpp.DebugWriter = log.WithFields(log.Fields{"logger": "stdlib"}).WriterLevel(log.TraceLevel)

		// Хэндлер сигналов.
		j.GTomb.Go(func() error { return j.SigHandler() }) //nolint: gocritic

		signal.Notify(j.SigChan, os.Interrupt)

		j.Options = &xmpp.Options{ //nolint:exhaustruct
			Host:     fmt.Sprintf("%s:%d", j.C.Jabber.Server, j.C.Jabber.Port),
			User:     j.C.Jabber.User,
			Password: j.C.Jabber.Password,
			Resource: j.C.Jabber.Resource,
			NoTLS:    !j.C.Jabber.Ssl,
			StartTLS: j.C.Jabber.StartTLS,
			TLSConfig: &tls.Config{ //nolint:exhaustruct
				ServerName:         j.C.Jabber.Server,
				InsecureSkipVerify: !j.C.Jabber.SslVerify, //nolint:gosec
			},
			InsecureAllowUnencryptedAuth: j.C.Jabber.InsecureAllowUnencryptedAuth,
			Debug:                        verboseClient,
			Session:                      false,
			Status:                       "xa",
			StatusMessage:                jabber.RandomPhrase(j.C.Jabber.StartupStatus),
			DialTimeout:                  time.Duration(j.C.Jabber.ConnectionTimeout) * time.Second,
		}

		// Враппер основной программы. Фактически на каждой серьёзной ошибке, например, сетевой, mainLoop и некоторые
		// вспомогательные корутины прекращает свою работу. Здесь мы это ловим и запускаем их снова. Задача в том, чтобы
		// запустить все асинхронные процессы заново, с чистого листа, но уже с распарсенным конфигом.

		// Устанавливаем соединение и гребём события, посылаемые сервером - основной и вспомогательные циклы программы.
		j.GTomb.Go(func() error { return j.MyLoop() }) //nolint: gocritic

		// Ловим первый же kill и не дождаемся остальных, хотя формально надо бы.
		<-j.GTomb.Dying()

		// Если у нас wire error, то вызов .Close() повлечёт за собой ошибку. А если у нас не wire error, то по ходу мы
		// получим утечку сокетов.

		time.Sleep(time.Duration(j.C.Jabber.ReconnectDelay) * time.Second)
	}
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
