package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Config struct {	
	DbUrl	string `json:"db_url"`
	CurrentUserName	string	`json:"current_user_name"`
}

const configFileName = ".gatorconfig.json"	//Name of config json file

func Read() (Config, error) {	//Reads config json file into Config struct
	jsonFile, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}
	file, err := os.Open(jsonFile)
	if err != nil {
		return Config{}, fmt.Errorf("error opening config file: %w", err)
	}
	defer file.Close()

	fileBytes, _ := io.ReadAll(file)

	var contents Config
	if err := json.Unmarshal(fileBytes, &contents); err != nil {
		return Config{}, fmt.Errorf("error retrieving json data: %w", err)
	}
	return contents, nil	
}

func (c *Config) SetUser(name string) error {	//Sets user of config struct
	c.CurrentUserName = name

	jsonFile, err := getConfigFilePath()
	if err != nil {
		return err
	}

	jsonData, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("error marshaling structure into json: %w", err)
	}

	if err = os.WriteFile(jsonFile, jsonData, 0666); err != nil {	//Permissions read/write
		return fmt.Errorf("error writing json file: %w", err)
	}
	return nil
}

func getConfigFilePath() (string, error) {	//Helper function to return the path for the config json file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error finding home directory: %w", err)
	}
	jsonFilePath := fmt.Sprintf("%s/%s", homeDir, configFileName)
	return jsonFilePath, nil
}