package utils

import (
	"github.com/uselagoon/machinery/utils/sshtoken"
)

// sshportal.go contains the functionality we need for connecting to the ssh portal and grab a list of deploy targets and environments

func GetToken(sshkeyPath, sshHost, sshPort string) (string, error) {
	return sshtoken.RetrieveToken(sshkeyPath, sshHost, sshPort)
}
