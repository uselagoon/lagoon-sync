package cmd

import "io/ioutil"

func LoadLagoonConfig(lagoonYamlPath string) ([]byte, error) {
	var data, err = ioutil.ReadFile(lagoonYamlPath)
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}
