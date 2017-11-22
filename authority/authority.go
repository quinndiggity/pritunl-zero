package authority

import (
	"crypto/rand"
	"github.com/dropbox/godropbox/container/set"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-zero/database"
	"github.com/pritunl/pritunl-zero/errortypes"
	"github.com/pritunl/pritunl-zero/user"
	"golang.org/x/crypto/ssh"
	"gopkg.in/mgo.v2/bson"
	"hash/fnv"
	"strings"
	"time"
)

type Authority struct {
	Id         bson.ObjectId `bson:"_id,omitempty" json:"id"`
	Name       string        `bson:"name" json:"name"`
	Type       string        `bson:"type" json:"type"`
	Roles      []string      `bson:"roles" json:"roles"`
	PrivateKey string        `bson:"private_key" json:"private_key"`
}

func (a *Authority) GenerateRsaPrivateKey() (err error) {
	keyBytes, err := GenerateRsaKey()
	if err != nil {
		return
	}

	a.PrivateKey = strings.TrimSpace(string(keyBytes))

	return
}

func (a *Authority) GenerateEcPrivateKey() (err error) {
	keyBytes, err := GenerateEcKey()
	if err != nil {
		return
	}

	a.PrivateKey = strings.TrimSpace(string(keyBytes))

	return
}

func (a *Authority) CreateCertificate(usr *user.User, sshPubKey string) (
	certMarshaled string, err error) {

	privateKey, err := ParsePemKey(a.PrivateKey)
	if err != nil {
		return
	}

	pubKey, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(sshPubKey))
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to parse ssh public key"),
		}
		return
	}

	serialHash := fnv.New64a()
	serialHash.Write([]byte(bson.NewObjectId().Hex()))
	serial := serialHash.Sum64()

	validAfter := time.Now().Add(-5 * time.Minute).Unix()
	validBefore := time.Now().Add(24 * time.Hour).Unix()

	cert := &ssh.Certificate{
		Key:             pubKey,
		Serial:          serial,
		CertType:        ssh.UserCert,
		KeyId:           usr.Id.Hex(),
		ValidPrincipals: usr.Roles,
		ValidAfter:      uint64(validAfter),
		ValidBefore:     uint64(validBefore),
		Permissions: ssh.Permissions{
			Extensions: map[string]string{
				"permit-X11-forwarding":   "",
				"permit-agent-forwarding": "",
				"permit-port-forwarding":  "",
				"permit-pty":              "",
				"permit-user-rc":          "",
			},
		},
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return
	}

	err = cert.SignCert(rand.Reader, signer)
	if err != nil {
		return
	}

	certMarshaled = string(MarshalCertificate(cert, comment))

	return
}

func (a *Authority) Validate(db *database.Database) (
	errData *errortypes.ErrorData, err error) {

	if a.Type == "" {
		a.Type = Local
	}

	return
}

func (a *Authority) Commit(db *database.Database) (err error) {
	coll := db.Authorities()

	err = coll.Commit(a.Id, a)
	if err != nil {
		return
	}

	return
}

func (a *Authority) CommitFields(db *database.Database, fields set.Set) (
	err error) {

	coll := db.Authorities()

	err = coll.CommitFields(a.Id, a, fields)
	if err != nil {
		return
	}

	return
}

func (a *Authority) Insert(db *database.Database) (err error) {
	coll := db.Authorities()

	if a.Id != "" {
		err = &errortypes.DatabaseError{
			errors.New("authority: Authority already exists"),
		}
		return
	}

	err = coll.Insert(a)
	if err != nil {
		err = database.ParseError(err)
		return
	}

	return
}
