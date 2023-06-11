package main

import (
	"encoding/xml"
)

// Конфиг
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
		Channels                     []string `json:"channels"`
		OutcastDomains               []string `json:"outcast_domains,omitempty"`
	}

	Loglevel string `json:"loglevel,omitempty"`
	Log      string `json:"log,omitempty"`
	NoEcho   bool   `json:"noecho,omitempty"`
}

// Белый список
type myWhiteList struct {
	Whitelist []struct {
		RoomName string   `json:"room_name,omitempty"`
		Jid      []string `json:"jid,omitempty"`
		WipeBans bool     `json:"wipe_bans,omitempty"`
	} `json:"whitelist,omitampty"`
}

type myBlackList struct {
	Blacklist []struct {
		RoomName string   `json:"room_name,omitempty"`
		JidRe    []string `json:"jid_re,omitempty"`
	} `json:"blacklist,omitempty"`
}

type jabberSimpleIqGetQuery struct {
	XMLName xml.Name `xml:"query"`
	Text    string   `xml:",chardata"`
	Xmlns   string   `xml:"xmlns,attr"`
	Node    string   `xml:"node,attr,omitempty"` // для xmlns="http://jabber.org/protocol/disco#items"
}

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

type jabberTimeIqGetQuery struct {
	// <time xmlns="urn:xmpp:time"/>
	XMLName xml.Name `xml:"time"`
	Text    string   `xml:",chardata"`
	Xmlns   string   `xml:"xmlns,attr"`
}

type jabberIqPing struct {
	XMLName xml.Name `xml:"ping"`
	Text    string   `xml:",chardata"`
	Xmlns   string   `xml:"xmlns,attr"`
}

type jabberIqErrorCancelNotAccepatble struct {
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
