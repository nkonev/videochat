package service

import (
	"github.com/pion/logging"
	"github.com/pion/turn/v2"
	"net"
	"nkonev.name/video/config"
	"os"
	"regexp"
)

func GetCompositeTurnAuth(conf config.ExtendedConfig) turn.AuthHandler {
	turnConstantLogger := logging.NewDefaultLeveledLoggerForScope("constant-creds", logging.LogLevelTrace, os.Stdout)
	usersMap := map[string][]byte{}
	for _, kv := range regexp.MustCompile(`(\w+)=(\w+)`).FindAllStringSubmatch(conf.Config.Turn.Auth.Credentials, -1) {
		usersMap[kv[1]] = turn.GenerateAuthKey(kv[1], conf.Config.Turn.Realm, kv[2])
	}
	var constantAuth turn.AuthHandler = func(username string, realm string, srcAddr net.Addr) ([]byte, bool) {
		turnConstantLogger.Tracef("Authentication with constant username=%q realm=%q srcAddr=%v\n", username, realm, srcAddr)
		if key, ok := usersMap[username]; ok {
			turnConstantLogger.Tracef("Successful authentication with constant username=%q realm=%q srcAddr=%v\n", username, realm, srcAddr)
			return key, true
		}
		turnConstantLogger.Tracef("Failed authentication with constant username=%q realm=%q srcAddr=%v, will try next longterm creds AuthHandler\n", username, realm, srcAddr)
		return nil, false
	}

	turnLtCredsLogger := logging.NewDefaultLeveledLoggerForScope("lt-creds", logging.LogLevelTrace, os.Stdout)
	var longtermAuth turn.AuthHandler = turn.NewLongTermAuthHandler(conf.Config.Turn.Auth.Secret, turnLtCredsLogger)
	compositeAuth := func(username string, realm string, srcAddr net.Addr) ([]byte, bool) {
		auth, success := constantAuth(username, realm, srcAddr)
		if success {
			return auth, true
		} else {
			return longtermAuth(username, realm, srcAddr)
		}
	}
	return compositeAuth
}

