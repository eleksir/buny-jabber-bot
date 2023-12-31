package main

import (
	"os"

	"github.com/eleksir/go-xmpp"
	"github.com/jbrukh/bayesian"
	"gopkg.in/tomb.v1"
)

const (
	Bad  bayesian.Class = "Bad"
	Good bayesian.Class = "Good"
)

// Config - это у нас глобальная штука :) .
var config myConfig

// Списки тех, кого мы не баним никогда.
var whiteList myWhiteList

// Списки злодеев, которых мы баним.
var blackList myBlackList

// Ставится в true, если мы получили сигнал на выключение.
var shutdown = false

// Чтобы не организовывать драку за установку коннекта.
var connecting = false

// Глобальное состояние соединения.
var isConnected = false

// Канал, в который приходят уведомления для хэндлера сигналов от траппера сигналов.
var sigChan = make(chan os.Signal, 1)

// Основной инстанс xmpp-клиента.
var talk *xmpp.Client

// Опции подключения к xmpp-серверу.
var options *xmpp.Options

// Список комнат, в которых находится бот.
var roomsConnected []string

// Время последней активности, нужно для jabber:iq:last.
var lastActivity int64

// Время последней активности, нужно для c2s пингов - посылаем пинги, только если давненько ничего не приходило с
// сервера.
var lastServerActivity int64

// Время последней активности MUC-ов, нужно для пингов - посылаем пинги, только если давненько ничего не приходило из
// muc-ов.
var lastMucActivity *Collection

// Получен ли ответ на запрос disco#info к серверу.
var serverCapsQueried bool

// sync.Map-ка с капабилити сервера.
var serverCapsList *Collection

// sync.Map-ка с комнатами и их capability.
var mucCapsList *Collection

// Время, когда был отправлен c2s ping.
var serverPingTimestampTx int64

// Время, когда был принят s2c pong.
var serverPingTimestampRx int64

// Объектик для хранения стейта утилизатора горутинок.
var gTomb tomb.Tomb

// sync.Map-ка со списком участников конференций (в json-формате, согласно структуре xmpp.Presence, "room".[]json).
var roomPresences *Collection

// Переменные для простенького нормализатора текста.
var (
	// Знаки препинания раз.
	pMarks = []string{".", ",", "!", "@", "#", "$", "%", "^", "&", "*", "(", ")", "{", "}", "<", ">", "[", "]", "\\"}

	// Знаки препинания два-с.
	pMarks2 = []string{"-", "_", "+", "=", ":", ";", "'", "`", "~", "\""}

	// Символы новой строки.
	newLines = []string{"\n", "\r", "\n\r", "\r\n"}
)

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
