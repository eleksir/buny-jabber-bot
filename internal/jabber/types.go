package jabber

import (
	"encoding/xml"
	"os"

	"github.com/eleksir/go-xmpp"
	"gopkg.in/tomb.v2"
)

// MyConfig прототип структурки с конфигом.
type MyConfig struct {
	Jabber struct {
		Server                       string   `json:"server,omitempty"`
		Port                         int      `json:"port,omitempty"`
		Ssl                          bool     `json:"ssl,omitempty"`
		StartTLS                     bool     `json:"starttls,omitempty"`
		SslVerify                    bool     `json:"ssl_verify,omitempty"`
		InsecureAllowUnencryptedAuth bool     `json:"insecureallowunencryptedauth,omitempty"`
		ConnectionTimeout            int64    `json:"connection_timeout,omitempty"`
		ReconnectDelay               int64    `json:"reconnect_delay,omitempty"`
		ServerPingDelay              int64    `json:"server_ping_delay,omitempty"`
		MucPingDelay                 int64    `json:"muc_ping_delay,omitempty"`
		MucRejoinDelay               int64    `json:"muc_rejoin_delay,omitempty"`
		PingSplayDelay               int64    `json:"ping_splay_delay,omitempty"`
		Nick                         string   `json:"nick,omitempty"`
		Resource                     string   `json:"resource,omitempty"`
		User                         string   `json:"user,omitempty"`
		Password                     string   `json:"password,omitempty"`
		BotMasters                   []string `json:"bot_masters,omitempty"`
		Channels                     []struct {
			Name     string `json:"name,omitempty"`
			Nick     string `json:"nick,omitempty"`
			Password string `json:"password,omitempty"`
			Bayes    struct {
				Enabled       bool   `json:"enabled,omitempty"`
				MinWords      int64  `json:"min_words,omitempty"`
				MinLength     int    `json:"min_length,omitempty"`
				DefaultAction string `json:"default_action,omitempty"`
			} `json:"bayes,omitempty"`
			AllCaps struct {
				Enabled       bool   `json:"enabled,omitempty"`
				MinLength     int    `json:"min_length,omitempty"`
				DefaultAction string `json:"default_action,omitempty"`
			} `json:"all_caps,omitempty"`
		} `json:"channels"`
		StartupStatus []string `json:"startup_status,omitempty"`
		RuntimeStatus struct {
			Text              []string `json:"text,omitempty"`
			RotationTime      int64    `json:"rotation_time,omitempty"`
			RotationSplayTime int64    `json:"rotation_splay_time,omitempty"`
		} `json:"runtime_status,omitempty"`
		BanDelay         int64    `json:"ban_delay,omitempty"`
		BanPhrasesEnable bool     `json:"ban_phrases_enable,omitempty"`
		BanPhrases       []string `json:"ban_phrases,omitempty"`
	} `json:"jabber,omitempty"`

	CSign    string `json:"csign,omitempty"`
	Loglevel string `json:"loglevel,omitempty"`
	Log      string `json:"log,omitempty"`
	Version  string `json:"version,omitempty"`
	ExeName  string `json:"exe_name,omitempty"`
}

// MyWhiteList прототип структурки с белым списком jid-ов.
type MyWhiteList struct {
	Whitelist []struct {
		RoomName string   `json:"room_name,omitempty"`
		Jid      []string `json:"jid,omitempty"`
		WipeBans bool     `json:"wipe_bans,omitempty"`
	} `json:"whitelist,omitempty"`
}

// MyBlackList прототип структурки с чёрным списком jid-ов.
type MyBlackList struct {
	Blacklist []struct {
		RoomName     string   `json:"room_name,omitempty"`
		ReasonEnable bool     `json:"reason_enable,omitempty"`
		JidRe        []string `json:"jid_re,omitempty"`
		NickRe       []string `json:"nick_re,omitempty"`
		PhraseRe     []string `json:"phrase_re,omitempty"`
		UserAgent    []struct {
			Name    string `json:"name,omitempty"`
			Version string `json:"version,omitempty"`
			Os      string `json:"os,omitempty"`
		} `json:"user_agent,omitempty"`
	} `json:"blacklist,omitempty"`
}

// Jabber основная структура-объект, содержащая стейты и проч.
type Jabber struct {
	// C - конфиг, как он распарсился из конфиг-файла.
	C MyConfig

	// WhiteList - структурка с безусловно разрешёнными jid-ами
	WhiteList MyWhiteList

	// BlackList - структурка с запрещёнными по регуляркам фразами, никами, jid-ами.
	BlackList MyBlackList

	// Опции подключения к xmpp-серверу.
	Options *xmpp.Options

	// gTomb пул активных горутин.
	GTomb tomb.Tomb

	// Talk основная структурка xmpp-клиента.
	Talk *xmpp.Client

	// sync.Map-ка с капабилити сервера.
	ServerCapsList *Collection

	// ServerCapsQueried показывает, были ли запрошены capabilities сервера.
	ServerCapsQueried bool

	// Время последней активности, нужно для c2s пингов - посылаем пинги, только если давненько ничего не приходило с
	// сервера.
	LastServerActivity int64

	// Время, когда был отправлен c2s ping.
	ServerPingTimestampTx int64

	// Время, когда был принят s2c pong.
	ServerPingTimestampRx int64

	// Время последней активности, нужно для jabber:iq:last.
	LastActivity int64

	// sync.Map-ка с комнатами и их capability.
	MucCapsList *Collection

	// Время последней активности MUC-ов, нужно для пингов - посылаем пинги, только если давненько ничего не приходило из
	// muc-ов.
	LastMucActivity *Collection

	// Список комнат, в которых находится бот.
	RoomsConnected []string

	// sync.Map-ка со списком участников конференций (в json-формате, согласно структуре xmpp.Presence, "room".[]json).
	RoomPresences *Collection

	// Канал, по котором приходят сообщения о том, что ОС отправила некие сигналы процессу.
	SigChan chan os.Signal

	// Индиктор того, что процесс завершается.
	Shutdown bool

	// Индиктор того, что соединение установлено.
	IsConnected bool

	// Индикатор того, что соединение в процессе достукивания до сервера.
	Connecting bool
}

// SimpleIqGetQuery прототип структурки для разбора запросов xmpp discovery query, например,
// https://xmpp.org/extensions/xep-0030.html#example-18 .
type SimpleIqGetQuery struct {
	XMLName xml.Name `xml:"query"`
	Text    string   `xml:",chardata"`
	Xmlns   string   `xml:"xmlns,attr"`
	Node    string   `xml:"node,attr,omitempty"` // для xmlns="http://jabber.org/protocol/disco#items"
}

// PubsubIQGetQuery прототип структурки для разбора запросов xmpp pubsub.
type PubsubIQGetQuery struct {
	XMLName xml.Name `xml:"pubsub"`
	Text    string   `xml:",chardata"`
	Xmlns   string   `xml:"xmlns,attr"`
	Items   struct {
		Text     string `xml:",chardata"`
		Node     string `xml:"node,attr"`
		MaxItems string `xml:"max_items,attr"`
	} `xml:"items"`
}

// TimeIqGetQuery прототип структурки для разбора IQ запросов на локальное время клиента,
// https://xmpp.org/extensions/xep-0202.html
type TimeIqGetQuery struct {
	// <time xmlns="urn:xmpp:time"/>
	XMLName xml.Name `xml:"time"`
	Text    string   `xml:",chardata"`
	Xmlns   string   `xml:"xmlns,attr"`
}

// IqPing прототип структурки для разбора IQ запросов на пинг клиента, https://xmpp.org/extensions/xep-0199.html
type IqPing struct {
	XMLName xml.Name `xml:"ping"`
	Text    string   `xml:",chardata"`
	Xmlns   string   `xml:"xmlns,attr"`
}

// IqErrorCancelNotAcceptable прототип структурки для разбора IQ ответов, когда сервис (сервер, клиент, etc) не
// может или не хочет принимать наш iq-запрос https://xmpp.org/extensions/xep-0099.html
type IqErrorCancelNotAcceptable struct {
	XMLName       xml.Name `xml:"error"`
	Text          string   `xml:",chardata"`
	Type          string   `xml:"type,attr"`
	By            string   `xml:"by,attr"`
	NotAcceptable struct {
		Text  string `xml:",chardata"`
		Xmlns string `xml:"xmlns,attr"`
	} `xml:"not-acceptable"`
}

// IqResultSoftwareVersion прототип структурки для разбора IQ ответов на запрос о названии и версии клиентского ПО.
type IqResultSoftwareVersion struct {
	XMLName xml.Name `xml:"query"`
	Text    string   `xml:",chardata"`
	Xmlns   string   `xml:"xmlns,attr"`
	Name    string   `xml:"name"`
	Version string   `xml:"version"`
	Os      string   `xml:"os,omitempty"`
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
