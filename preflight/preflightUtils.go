package preflight

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func MysqlConnectionCommand(database string, hostname string, port string, username string, password string) string {
	//@TODO: db password should be hidden from log output
	// A solution here could be to write the credentials to file and read from 'mysql --defaults-file' - passing in username and password
	// see CreateDBCredentialsTempFile()

	return fmt.Sprintf(
		"mysql -u%s -p%s --database=%s --host=%s --port=%s -A",
		username,
		password,
		database,
		hostname,
		port,
	)
}

func CreateDBCredentialsTempFile(username string, password string, dir string, debug bool) (string, error) {
	tmpFile, err := ioutil.TempFile(dir, fmt.Sprintf("%s-", filepath.Base(os.Args[0])))
	if err != nil {
		log.Fatal("Could not create temporary file", err)
	}
	defer tmpFile.Close()

	if (debug) {
		fmt.Println("Created temp file: ", tmpFile.Name())
	}
	if _, err = tmpFile.WriteString(fmt.Sprintf("#This file was written by lagoon-sync.\n[client]\n%s\n%s\n", os.Getenv(username), os.Getenv(password))); err != nil {
		if debug {
			log.Println("Unable to write to temporary file", err)
		}
	} else {
		if debug {
			fmt.Println("Credentials have been written to file")
		}
	}

	return tmpFile.Name(), nil
}

func StringIsWildcard (string string) bool {
	// regexp.MatchString(".*", table)
	return strings.Contains(string, "*")
}

func FindMatchingTablesFromWildcardPattern(option string, tablesList []string) []string {
	var tables []string
	var splitOption = strings.Split(option, "*")
	ignoreTableOption := splitOption[0]

	for i, table := range tablesList {
		// remove first value as this will be the table header
		if i == 0 {
			continue
		}

		//fmt.Println(table, splitOption[0], strings.Contains(table, ignoreTableOption))

		optionMatchesTable := strings.Contains(table, ignoreTableOption)
		if  optionMatchesTable {
			//fmt.Println("match; ", ignoreTableOption, table)
			tables = append(tables, table)
		}
	}
	return tables
}