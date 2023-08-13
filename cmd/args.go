package cmd

import "github.com/uselagoon/lagoon-sync/synchers"

// This file contains the shared argument variables used by the various commands.

var ProjectName string
var sourceEnvironmentName string
var targetEnvironmentName string
var SyncerType string
var ServiceName string
var configurationFile string
var SSHHost string
var SSHPort string
var SSHKey string
var SSHVerbose bool
var CmdSSHKey string
var noCliInteraction bool
var dryRun bool
var verboseSSH bool
var RsyncArguments string
var runSyncProcess synchers.RunSyncProcessFunctionType
var skipSourceCleanup bool
var skipTargetCleanup bool
var skipTargetImport bool
var localTransferResourceName string
var namedTransferResource string
