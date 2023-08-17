package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/eleksir/go-xmpp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/tomb.v1"
)

func init() {
	log.SetFormatter(&log.TextFormatter{ //nolint:exhaustruct
		DisableQuote:           true,
		DisableLevelTruncation: false,
		DisableColors:          true,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
	})

	readConfig()

	// no panic
	switch config.Loglevel {
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
}

func main() {
	// Откроем лог и скормим его логгеру
	if config.Log != "" {
		logfile, err := os.OpenFile(config.Log, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

		if err != nil {
			log.Fatalf("Unable to open log file %s: %s", config.Log, err)
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
	readWhitelist()
	readBlacklist()

	// github.com/mattn/go-xmpp пишет в stdio, нам этого не надо, ловим выхлоп его в logrus с уровнем trace
	xmpp.DebugWriter = log.WithFields(log.Fields{"logger": "stdlib"}).WriterLevel(log.TraceLevel)

	// Хэндлер сигналов не надо трогать, он нужен для завершения программы целиком
	go sigHandler()
	signal.Notify(sigChan, os.Interrupt)

	for {
		options = &xmpp.Options{ //nolint:exhaustruct
			Host:     fmt.Sprintf("%s:%d", config.Jabber.Server, config.Jabber.Port),
			User:     config.Jabber.User,
			Password: config.Jabber.Password,
			Resource: config.Jabber.Resource,
			NoTLS:    !config.Jabber.Ssl,
			StartTLS: config.Jabber.StartTLS,
			TLSConfig: &tls.Config{ //nolint:exhaustruct
				ServerName:         config.Jabber.Server,
				InsecureSkipVerify: !config.Jabber.SslVerify, //nolint:gosec
			},
			InsecureAllowUnencryptedAuth: config.Jabber.InsecureAllowUnencryptedAuth,
			Debug:                        verboseClient,
			Session:                      false,
			Status:                       "xa",
			StatusMessage:                randomPhrase(config.Jabber.StartupStatus),
			DialTimeout:                  time.Duration(config.Jabber.ConnectionTimeout) * time.Second,
		}

		// Через tomb попробуем сделать выход горутинок управляемым
		gTomb = tomb.Tomb{}

		// Устанавливаем соединение и гребём события, посылаемые сервером
		myLoop()

		log.Error(gTomb.Wait())
		time.Sleep(time.Duration(config.Jabber.ReconnectDelay) * time.Second)
	}
}

func myLoop() {
	defer gTomb.Done()

	for {
		select {
		case <-gTomb.Dying():
			return
		default:
			// Зададим начальное значение глобальным переменным
			serverPingTimestampRx = 0
			serverPingTimestampTx = 0
			roomsConnected = make([]string, 1)
			lastActivity = 0
			lastServerActivity = 0
			lastMucActivity = NewCollection()
			serverCapsQueried = false
			serverCapsList = NewCollection()
			mucCapsList = NewCollection()
			serverPingTimestampTx = 0
			serverPingTimestampRx = 0

			// Установим коннект
			establishConnection()

			serverPingTimestampRx = time.Now().Unix() // Считаем, что если коннект запустился, то первый пинг успешен

			// Тыкаем сервер палочкой, проверяем, что коннект жив и переустанавливаем его, если он не жив
			go probeServerLiveness()

			// Тыкаем muc-и палочкой, проверяем, что они живы и пере-заходим в них, если пинги пропали
			go probeMUCLiveness()

			// Гребём сообщения...
			for {
				// Стриггерилось завершение работы приложения, или соединение не установлено (порвалось, например)
				// грести не надо
				if shutdown {
					break
				}

				if !isConnected {
					continue
				}

				chat, err := talk.Recv()

				if err != nil {
					log.Errorf("Unable to get events from server: %s", err)

					switch {
					// Стрим не читается, он закрылся с той стороны во время чтения
					case errors.Is(err, io.EOF):
						err := fmt.Errorf("tcp stream closed while reading, err=%w", err)
						gTomb.Kill(err)

						continue

					// Пытаемся читать закрытый сокет
					case errors.Is(err, net.ErrClosed):
						err := fmt.Errorf("unable to read closed socket, err=%w", err)
						gTomb.Kill(err)

						continue

					// Не смогли записать в сокет
					case errors.Is(err, net.ErrWriteToConnected):
						err := fmt.Errorf("unable to write to socket, err=%w", err)
						gTomb.Kill(err)

						continue

					// Не сетевая проблема
					default:
						// Это уже что-то странное.
						// Вероятно, ошибка парсинга xml. Собственно, баг сервера, тут мы ничего поделать не можем
						err := fmt.Errorf("error during parsing received message, err=%w", err)
						gTomb.Kill(err)

						continue
					}
				}

				parseEvent(chat)
			}
		}
	}
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
