package cmd

import (
	"bufio"
	"crypto"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/inconshreveable/go-update"
	"github.com/spf13/cobra"
	"github.com/uselagoon/lagoon-sync/utils"
	"golang.org/x/crypto/openpgp"
)

const selfUpdateDownloadURL = "https://api.github.com/repos/uselagoon/lagoon-sync/releases/latest"

var osArch, downloadPath, checkSumFileURL, sigFileURL, checksumStr string

var publicKey = strings.NewReader(`
-----BEGIN PGP PUBLIC KEY BLOCK-----

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
=xcNP
-----END PGP PUBLIC KEY BLOCK-----
`)

// selfUpdateCmd represents the selfUpdate command
var selfUpdateCmd = &cobra.Command{
	Use:   "selfUpdate",
	Short: "Update this tool to the latest version",
	Long:  "Update this tool to the latest version.",
	Run: func(cmd *cobra.Command, args []string) {
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

	defer resp.Body.Close()
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading response", err)
	}

	result := make(map[string]interface{})
	json.Unmarshal(bodyText, &result)

	results := make([]interface{}, 0)
	for _, asset := range result["assets"].([]interface{}) {
		results = append(results, asset.(map[string]interface{})["browser_download_url"])
	}

	osArch = fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	for _, res := range results {
		str := res.(string)
		if strings.Contains(str, osArch) {
			downloadPath = str
		}
		if strings.Contains(str, "checksums.txt") && !strings.Contains(str, "checksums.txt.sig") {
			checkSumFileURL = str
		}
		if strings.Contains(str, "checksums.txt.sig") {
			sigFileURL = str
		}
	}

	return downloadPath, nil
}

func doUpdate(url string) error {
	utils.LogProcessStep("Downloading binary from", url)
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

	checksumRespBody, err := getChecksum(checkSumFileURL)
	if err != nil {
		fmt.Println(err)
		return err
	}
	checkSumOut, err := os.Create("/tmp/checksum.txt")
	if err != nil {
		return err
	}
	defer checkSumOut.Close()
	_, err = checkSumOut.Write(checksumRespBody)

	// Open Checksum file
	checkSumfile, err := os.Open("/tmp/checksum.txt")
	if err != nil {
		log.Fatal(err)
	}

	// Pull out the inidivdual checksum from the list for given binary version
	scanner := bufio.NewScanner(checkSumfile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, osArch) {
			checksumOSArch := osArch + ":" + line[0:64]
			fmt.Printf("\x1b[37mChecksum for: %s\x1b[0m\n", checksumOSArch)
			checksumStr = line[0:64]
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	// Checksum hex into string
	checksum, err := hex.DecodeString(checksumStr)
	if err != nil {
		return err
	}

	sigFileResp, err := http.Get(sigFileURL)
	if err != nil {
		panic(err)
	}
	if sigFileResp.StatusCode != 200 {
		fmt.Printf(sigFileResp.Status)
		os.Exit(2)
	}
	defer sigFileResp.Body.Close()
	sigOut, err := os.Create("/tmp/checksum.txt.sig")
	if err != nil {
		return err
	}
	defer sigOut.Close()
	// Write signature body to file
	_, err = io.Copy(sigOut, sigFileResp.Body)

	// Verification step
	// Note: We need to verify the complete signed checksum using openpgp (as go-updater requires public key in PEM format, but
	// goreleaser uses GPG).

	// Load from public key reader abover, instead of os.Open
	keyring, err := openpgp.ReadArmoredKeyRing(publicKey)
	if err != nil {
		utils.LogFatalError("Public key error", err.Error())
	}
	// Target checksum we need to verify that it hasn't been manipulated
	verificationTarget, err := os.Open("/tmp/checksum.txt")
	if err != nil {
		utils.LogFatalError("Can't open checksum target", err)
	}
	// Signature of the new executable, signed by the private key during GH release
	sig, err := os.Open("/tmp/checksum.txt.sig")
	if err != nil {
		utils.LogFatalError("Can't open signature", err)
	}

	// When the signature is binary instead of armored, the error was EOF.
	// e.g. entity, err := openpgp.CheckArmoredDetachedSignature(keyring, verificationTarget, sig)
	// So using the binary signature method instead
	entity, err := openpgp.CheckDetachedSignature(keyring, verificationTarget, sig)
	if err == io.EOF {
		// If signature has EOF issues, the client failure is just "EOF", which is not helpful
		cleanUpSelfUpdate("/tmp/checksum.txt.sig", "/tmp/checksum.txt")
		utils.LogFatalError("No valid signatures found in target checksum file", err)
	}
	if err != nil {
		cleanUpSelfUpdate("/tmp/checksum.txt.sig", "/tmp/checksum.txt")
		utils.LogFatalError("Verifcation error", err)
	}

	for _, identity := range entity.Identities {
		sigYear, sigMonth, sigDay := identity.SelfSignature.CreationTime.Date()
		sigInfo := fmt.Sprintf("\"%s\" (created %v %s %v)", identity.UserId.Name, sigDay, sigMonth, sigYear)

		fmt.Fprintf(os.Stderr, "\x1b[32mGood signature from %s\x1b[0m\n", sigInfo)
	}

	fmt.Printf("\x1b[36;1mApplying update...\x1b[0m\n")

	// We have now verified at this step using opengpg, so we only define additional go-update options.
	opts := update.Options{
		TargetPath: exec,
		Hash:       crypto.SHA256,
		Checksum:   checksum,
	}

	// If we get this far, then we can apply the update
	err = update.Apply(resp.Body, opts)
	if err != nil {
		fmt.Println(err)
		cleanUpSelfUpdate("/tmp/checksum.txt.sig", "/tmp/checksum.txt")
		os.Exit(2)
	}
	fmt.Printf("\x1b[32mSuccessfully updated binary at:\x1b[0m %s\n", exec)
	cleanUpSelfUpdate("/tmp/checksum.txt.sig", "/tmp/checksum.txt")
	return err
}

func getChecksum(url string) ([]byte, error) {
	checkSumFileResp, err := http.Get(checkSumFileURL)
	if err != nil {
		panic(err)
	}

	if checkSumFileResp.StatusCode != 200 {
		fmt.Printf(checkSumFileResp.Status)
		os.Exit(2)
	}
	defer checkSumFileResp.Body.Close()
	checkSumFileRespBodyText, err := ioutil.ReadAll(checkSumFileResp.Body)
	if err != nil {
		log.Fatal("Error reading response", err)
	}

	return checkSumFileRespBodyText, err
}

func cleanUpSelfUpdate(signatureFile string, checksumFile string) error {
	fmt.Printf("\x1b[37mCleanup signature and checksum files\x1b[0m\n")
	files := []string{signatureFile, checksumFile}

	if !dryRun {
		for _, f := range files {
			if err := os.Remove(f); err != nil {
				utils.LogFatalError(err.Error(), nil)
			}
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(selfUpdateCmd)
}
