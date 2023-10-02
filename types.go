package main

import (
	"encoding/xml"
)

// myConfig прототип структурки с конфигом.
type myConfig struct {
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
		BotMasters                   []string `json:"bot_masters"`
		Channels                     []string `json:"channels"`
		StartupStatus                []string `json:"startup_status,omitempty"`
		RuntimeStatus                struct {
			Text              []string `json:"text,omitempty"`
			RotationTime      int64    `json:"rotation_time,omitempty"`
			RotationSplayTime int64    `json:"rotation_splay_time,omitempty"`
		} `json:"runtime_status,omitempty"`
		BanPhrasesEnable bool     `json:"ban_phrases_enable,omitempty"`
		BanPhrases       []string `json:"ban_phrases,omitempty"`
	} `json:"jabber,omitempty"`

	CSign    string `json:"csign,omitempty"`
	Loglevel string `json:"loglevel,omitempty"`
	Log      string `json:"log,omitempty"`
}

// myWhiteList прототип структурки с белым списком jid-ов.
type myWhiteList struct {
	Whitelist []struct {
		RoomName string   `json:"room_name,omitempty"`
		Jid      []string `json:"jid,omitempty"`
		WipeBans bool     `json:"wipe_bans,omitempty"`
	} `json:"whitelist,omitempty"`
}

// myBlackList прототип структурки с чёрным списком jid-ов.
type myBlackList struct {
	Blacklist []struct {
		RoomName     string   `json:"room_name,omitempty"`
		ReasonEnable bool     `json:"reason_enable,omitempty"`
		JidRe        []string `json:"jid_re,omitempty"`
		NickRe       []string `json:"nick_re,omitempty"`
		PhraseRe     []string `json:"phrase_re,omitempty"`
	} `json:"blacklist,omitempty"`
}

// jabberSimpleIqGetQuery прототип структурки для разбора запросов xmpp discovery query, например,
// https://xmpp.org/extensions/xep-0030.html#example-18 .
type jabberSimpleIqGetQuery struct {
	XMLName xml.Name `xml:"query"`
	Text    string   `xml:",chardata"`
	Xmlns   string   `xml:"xmlns,attr"`
	Node    string   `xml:"node,attr,omitempty"` // для xmlns="http://jabber.org/protocol/disco#items"
}

// jabberPubsubIQGetQuery прототип структурки для разбора запросов xmpp pubsub.
type jabberPubsubIQGetQuery struct {
	XMLName xml.Name `xml:"pubsub"`
	Text    string   `xml:",chardata"`
	Xmlns   string   `xml:"xmlns,attr"`
	Items   struct {
		Text     string `xml:",chardata"`
		Node     string `xml:"node,attr"`
		MaxItems string `xml:"max_items,attr"`
	} `xml:"items"`
}

// jabberTimeIqGetQuery прототип структурки для разбора IQ запросов на локальное время клиента,
// https://xmpp.org/extensions/xep-0202.html
type jabberTimeIqGetQuery struct {
	// <time xmlns="urn:xmpp:time"/>
	XMLName xml.Name `xml:"time"`
	Text    string   `xml:",chardata"`
	Xmlns   string   `xml:"xmlns,attr"`
}

// jabberIqPing прототип структурки для разбора IQ запросов на пинг клиента, https://xmpp.org/extensions/xep-0199.html
type jabberIqPing struct {
	XMLName xml.Name `xml:"ping"`
	Text    string   `xml:",chardata"`
	Xmlns   string   `xml:"xmlns,attr"`
}

// jabberIqErrorCancelNotAcceptable прототип структурки для разбора IQ ответов, когда сервис (сервер, клиент, etc) не
// может или не хочет принимать наш iq-запрос https://xmpp.org/extensions/xep-0099.html
type jabberIqErrorCancelNotAcceptable struct {
	XMLName       xml.Name `xml:"error"`
	Text          string   `xml:",chardata"`
	Type          string   `xml:"type,attr"`
	By            string   `xml:"by,attr"`
	NotAcceptable struct {
		Text  string `xml:",chardata"`
		Xmlns string `xml:"xmlns,attr"`
	} `xml:"not-acceptable"`
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
