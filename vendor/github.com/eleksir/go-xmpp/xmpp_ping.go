package xmpp

import (
	"fmt"
)

func (c *Client) PingC2S(jid, server string) error {
	if jid == "" {
		jid = c.jid
	}

	if server == "" {
		server = c.domain
	}

	_, err := fmt.Fprintf(StanzaWriter, "<iq from='%s' to='%s' id='c2s1' type='get'>\n"+
		"<ping xmlns='urn:xmpp:ping'/>\n"+
		"</iq>",
		xmlEscape(jid), xmlEscape(server))

	return err
}

func (c *Client) PingS2S(fromServer, toServer string) error {
	_, err := fmt.Fprintf(StanzaWriter, "<iq from='%s' to='%s' id='s2s1' type='get'>\n"+
		"<ping xmlns='urn:xmpp:ping'/>\n"+
		"</iq>",
		xmlEscape(fromServer), xmlEscape(toServer))

	return err
}

func (c *Client) SendResultPing(id, toServer string) error {
	_, err := fmt.Fprintf(StanzaWriter, "<iq type='result' to='%s' id='%s'/>",
		xmlEscape(toServer), xmlEscape(id))

	return err
}

// PingResponse responding to ping query according to xep-0199.
func (c *Client) PingResponse(v IQ) (string, error) {
	return c.RawInformation(
		v.To,
		v.From,
		v.ID,
		IQTypeResult,
		"",
	)
}
