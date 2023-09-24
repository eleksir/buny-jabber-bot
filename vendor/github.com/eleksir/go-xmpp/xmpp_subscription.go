package xmpp

import (
	"fmt"
)

func (c *Client) ApproveSubscription(jid string) error {
	_, err := fmt.Fprintf(StanzaWriter, "<presence to='%s' type='subscribed'/>",
		xmlEscape(jid))

	return err
}

func (c *Client) RevokeSubscription(jid string) error {
	_, err := fmt.Fprintf(StanzaWriter, "<presence to='%s' type='unsubscribed'/>",
		xmlEscape(jid))

	return err
}

func (c *Client) RetrieveSubscription(jid string) error {
	_, err := fmt.Fprintf(c.conn, "<presence to='%s' type='unsubscribe'/>",
		xmlEscape(jid))

	return err
}

func (c *Client) RequestSubscription(jid string) error {
	_, err := fmt.Fprintf(StanzaWriter, "<presence to='%s' type='subscribe'/>",
		xmlEscape(jid))

	return err
}
