package jabber

import (
	"fmt"
	"time"

	"github.com/eleksir/go-xmpp"
)

// Squash банит указанный jid в указанной комнате.
// reasonEnable указывает, надо ли писать дату автобана в банлисте в поле reason (это единственная причина, в которую
// умеет бот).
func (j *Jabber) Squash(room, jid string, reasonEnable bool, vType string) (string, error) {
	var (
		id  string
		err error
	)

	if j.C.Jabber.BanPhrasesEnable {
		phrase := RandomPhrase(j.C.Jabber.BanPhrases)

		if _, err = j.Talk.Send(
			xmpp.Chat{ //nolint:exhaustruct
				Remote: room,
				Text:   phrase,
				Type:   vType,
			},
		); err != nil {
			err = fmt.Errorf("unable to send phrase to room %s: %w", room, err)

			// Здесь возвращаем nil, т.к. за нас ошибку залоггирует код выше.
			return id, err
		}
	}

	// https://xmpp.org/extensions/xep-0045.html#ban баним вот таким сообщением
	ban := "<item affiliation='outcast' jid='" + jid + "'>"

	if reasonEnable {
		var t = time.Now()
		ban += fmt.Sprintf(
			"<reason>autoban at %04d.%02d.%02d %02d:%02d:%02d</reason>",
			t.Year(),
			t.Month(),
			t.Day(),
			t.Hour(),
			t.Minute(),
			t.Second(),
		)
	} else {
		ban += "<reason />"
	}

	ban += "</item>"

	// Выжидаем некоторое время перед баном. А то можно настолько рано забанить, что сервер не внесёт злодея в банлист
	// комнаты и пришлёт affiliation: none вместо affiliation: outcast.
	if j.C.Jabber.BanDelay > 0 {
		time.Sleep(time.Duration(j.C.Jabber.BanDelay) * time.Millisecond)
	}

	if id, err = j.Talk.RawInformationQuery(
		j.Talk.JID(),
		room,
		"ban1",
		xmpp.IQTypeSet,
		"http://jabber.org/protocol/muc#admin",
		ban,
	); err != nil {
		err = fmt.Errorf(
			"unable to ban user: id=%s, err=%w",
			id,
			err,
		)
	}

	return id, err
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
