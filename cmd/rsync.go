package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"

	"github.com/amazeeio/lagoon-sync/assets"
	"github.com/spf13/cobra"
)

var rsyncCmd = &cobra.Command{
	Use:   "rsync",
	Short: "remote syncing cmd test",
	Long:  "remote syncing cmd test",
	Run: func(cmd *cobra.Command, args []string) {
		testLocalRsync()
	},
}

func testLocalRsync() {
	err := ioutil.WriteFile("/tmp/rsync", assets.GetRSYNC(), 0774)
	if err != nil {
		log.Fatal(err)
	}

	// Test running local rsync binary
	localRsyncCommand := exec.Command("/tmp/rsync", "--version")
	// stdin, err := local_rsync_command.StdinPipe()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	stdout, err := localRsyncCommand.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := localRsyncCommand.Start(); err != nil {
		log.Fatal(err)
	}

	r := bufio.NewReader(stdout)
	b, err := r.ReadBytes('\n')
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Local rsync ran:", string(b))
}

func init() {
	rootCmd.AddCommand(rsyncCmd)
}
