/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"crypto"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/inconshreveable/go-update"
	"github.com/spf13/cobra"
)

const selfUpdateDownloadURL = "https://github.com/amazeeio/lagoon-sync/releases/latest/download/lagoon-sync"

// selfUpdateCmd represents the selfUpdate command
var selfUpdateCmd = &cobra.Command{
	Use:   "selfUpdate",
	Short: "Update this tool to the latest version",
	Long:  "Update this tool to the latest version.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("selfUpdate called")
		finalDLUrl, err := followRedirectsToActualFile(selfUpdateDownloadURL)
		if err != nil {
			log.Printf("There was an error resolving the self-update url : %v", err.Error())
			return
		}
		doUpdate(finalDLUrl)
	},
}

func followRedirectsToActualFile(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("http.Get => %v", err.Error())
		return "", err
	}
	return resp.Request.URL.String(), nil
}

func doUpdate(url string) error {
	fmt.Printf("Downloading binary from %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		fmt.Printf(resp.Status)
		os.Exit(2)
	}
	defer resp.Body.Close()

	exec, err := os.Executable()
	if err != nil {
		return err
	}

	fmt.Printf("Applying update...\n")
	//TODO: add support for gpg verification
	err = update.Apply(resp.Body, update.Options{
		TargetPath: exec,
		Hash:       crypto.SHA256,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Successfully updated binary at: %s\n", exec)
	return err
}

func init() {
	rootCmd.AddCommand(selfUpdateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	selfUpdateCmd.PersistentFlags().String("version", "", "Define version to update")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// selfUpdateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
