package jabber

import (
	"fmt"

	"github.com/eleksir/go-xmpp"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// QuerySoftwareVersion запрошивает версию и название клиента.
func (j *Jabber) QuerySoftwareVersion(jid string) (string, error) {
	var (
		id  string
		err error
	)

	if id, err = j.Talk.RawInformationQuery(
		j.Talk.JID(),
		jid,
		uuid.New().String(),
		xmpp.IQTypeGet,
		"jabber:iq:version",
		"",
	); err != nil {
		err = fmt.Errorf(
			"unable to query software version of jid=%s err=%w",
			jid,
			err,
		)
	} else {
		log.Infof("Query software version of %s", jid)
	}

	return id, err
}
