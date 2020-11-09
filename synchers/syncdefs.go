package synchers

type Syncer interface {
	GetRemoteCommand() string
	GetLocalCommand() string
	GetTransferResourceName() string
}