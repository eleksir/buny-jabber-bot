package xmpp

import (
	"fmt"
	"strconv"
	"time"
)

const IQTypeGet = "get"
const IQTypeSet = "set"
const IQTypeResult = "result"
const IQTypeError = "error"

func (c *Client) Discovery() (string, error) {
	// use getCookie for a pseudo random id.
	reqID := strconv.FormatUint(uint64(getCookie()), 10)
	return c.RawInformationQuery(c.jid, c.domain, reqID, IQTypeGet, XMPPNS_DISCO_ITEMS, "")
}

// DiscoverNodeInfo discovers information about a node. Empty node queries info about server itself.
func (c *Client) DiscoverNodeInfo(node string) (string, error) {
	query := fmt.Sprintf("<query xmlns='%s' node='%s'/>", XMPPNS_DISCO_INFO, node)
	return c.RawInformation(c.jid, c.domain, "info3", IQTypeGet, query)
}

// DiscoverInfo discovers information about given item from given jid.
func (c *Client) DiscoverInfo(from string, to string) (string, error) {
	query := fmt.Sprintf("<query xmlns='%s'/>", XMPPNS_DISCO_INFO)
	return c.RawInformation(from, to, "info3", IQTypeGet, query)
}

// DiscoverServerItems discover items that the server exposes
func (c *Client) DiscoverServerItems() (string, error) {
	return c.DiscoverEntityItems(c.domain)
}

// DiscoverEntityItems discovers items that an entity exposes.
func (c *Client) DiscoverEntityItems(jid string) (string, error) {
	query := fmt.Sprintf("<query xmlns='%s'/>", XMPPNS_DISCO_ITEMS)
	return c.RawInformation(c.jid, jid, "info1", IQTypeGet, query)
}

// RawInformationQuery sends an information query request to the server.
func (c *Client) RawInformationQuery(from, to, id, iqType, requestNamespace, body string) (string, error) {
	const xmlIQ = "<iq from='%s' to='%s' id='%s' type='%s'><query xmlns='%s'>%s</query></iq>"
	_, err := fmt.Fprintf(StanzaWriter, xmlIQ, xmlEscape(from), xmlEscape(to), id, iqType, requestNamespace, body)
	return id, err
}

// RawInformation sends a IQ request with the payload body to the server.
func (c *Client) RawInformation(from, to, id, iqType, body string) (string, error) {
	const xmlIQ = "<iq from='%s' to='%s' id='%s' type='%s'>%s</iq>"
	_, err := fmt.Fprintf(StanzaWriter, xmlIQ, xmlEscape(from), xmlEscape(to), id, iqType, body)
	return id, err
}

// IqVersionResponse responding with software version, according to xep-0092.
func (c *Client) IqVersionResponse(v IQ, name, version, os string) (string, error) {
	if name == "" {
		name = "go-xmpp"
	}

	if version == "" {
		version = "undefined"
	}

	query := "<query xmlns=\"jabber:iq:version\">" //nolint:wsl
	query += fmt.Sprintf("<name>%s</name>", name)
	query += fmt.Sprintf("<version>%s</version>", version)

	if os != "" {
		query += fmt.Sprintf("<os>%s</os>", os)
	}

	query += "</query>"

	return c.RawInformation(
		v.To,
		v.From,
		v.ID,
		IQTypeResult,
		query,
	)
}

// JabberIqLastResponse responding with relative time since last activity.
// Here lastActivity is unix time stamp when last activity took place since.
func (c *Client) JabberIqLastResponse(v IQ, lastActivity int64) (string, error) {
	query := fmt.Sprintf(
		"<query xmlns=\"jabber:iq:last\" seconds=\"%d\" />",
		time.Now().Unix()-lastActivity,
	)

	return c.RawInformation(
		v.To,
		v.From,
		v.ID,
		IQTypeResult,
		query,
	)
}

// UrnXmppTimeResponse implements response to query entity's current time (xep-0202).
// TimezoneOffset have format +HH:MM or -HH:MM
func (c *Client) UrnXmppTimeResponse(v IQ, timezoneOffset string) (string, error) {
	query := fmt.Sprintf(
		"<time xmlns=\"urn:xmpp:time\"><tzo>%s</tzo><utc>%s</utc></time>",
		timezoneOffset,
		time.Now().UTC().Format(time.RFC3339),
	)

	return c.RawInformation(
		v.To,
		v.From,
		v.ID,
		IQTypeResult,
		query,
	)
}
