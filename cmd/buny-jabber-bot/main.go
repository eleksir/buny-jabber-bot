package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"time"

	"buny-jabber-bot/internal/jabber"

	"github.com/eleksir/go-xmpp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/tomb.v2"
)

// main - фактичеcки, начало и основное тело программы.
func main() {
	var j = jabber.Jabber{
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

	// Хэндлер сигналов не надо трогать, он нужен для завершения программы целиком.
	go j.SigHandler()
	signal.Notify(j.SigChan, os.Interrupt)

	// Враппер основной программы. Фактически на каждой серьёзной ошибке, например, сетевой, mainLoop и некоторые
	// вспомогательные корутины прекращает свою работу. Здесь мы это ловим и запускаем их снова. Задача в том, чтобы
	// запустить все асинхронные процессы заново, с чистого листа, но уже с распарсенным конфигом.
	for {
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

		// Устанавливаем соединение и гребём события, посылаемые сервером - основной и вспомогательные циклы программы.
		j.GTomb.Go(func() error { return j.MyLoop() })

		// Логгируем причину завершения mainLoop и вспомогательных циклов программы.
		log.Error(j.GTomb.Wait())

		// Если у нас wire error, то вызов .Close() повлечёт за собой ошибку, но мы вынуждены звать .Close(), чтоб
		// закрыть tls контекст и почистить всё что связанно с прерванным соединением.
		log.Infoln("Closing connection to jabber server")

		_ = j.Talk.Close()

		time.Sleep(time.Duration(j.C.Jabber.ReconnectDelay) * time.Second)
	}
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
