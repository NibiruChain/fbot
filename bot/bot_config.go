package fbot

import (
	"fmt"
	"io"
	"os"
	"path"

	"reflect"

	"github.com/Unique-Divine/gonibi"
	"github.com/joho/godotenv"
)

const ENV_FILENAME = ".env.bot"

type BotConfig struct {
	MNEMONIC       string
	CHAIN_ID       string
	GRPC_ENDPOINT  string
	TMRPC_ENDPOINT string
}

// Initiliaze fields in file and/or struct
//		- Need to be able to pass in mnemonic
//		- Info should stay between calls
//		- Check if inputs are valid

// Initialize Config from .env.bot

func EnvFilePath() string {
	cwd, _ := os.Getwd()
	envFilePath := path.Join(cwd, ENV_FILENAME)
	return envFilePath
}

func Load() (*BotConfig, error) {

	vars, err := godotenv.Read(EnvFilePath())

	if err != nil {
		return nil, err
	}
	var newConfig = &BotConfig{
		MNEMONIC:       vars["MNEMONIC"],
		CHAIN_ID:       vars["CHAIN_ID"],
		GRPC_ENDPOINT:  vars["GRPC_ENDPOINT"],
		TMRPC_ENDPOINT: vars["TMRPC_ENDPOINT"],
	}

	return newConfig, err
}

func LoadDefaultNetwork(mnemonic string) *BotConfig {

	var newConfig = &BotConfig{
		MNEMONIC:       mnemonic,
		CHAIN_ID:       gonibi.DefaultNetworkInfo.ChainID,
		GRPC_ENDPOINT:  gonibi.DefaultNetworkInfo.GrpcEndpoint,
		TMRPC_ENDPOINT: gonibi.DefaultNetworkInfo.TmRpcEndpoint,
	}

	return newConfig
}

func (config *BotConfig) CheckConfig() error {

	reflectConfig := reflect.ValueOf(*config)

	for i := 0; i < reflectConfig.NumField(); i++ {
		field := reflectConfig.Field(i)

		if field.Interface() == reflect.Zero(field.Type()).Interface() {
			return fmt.Errorf("Undefined Bot Config Field")
		}
	}

	kring, _, err := gonibi.CreateSigner(config.MNEMONIC,
		gonibi.NewKeyring(), "test")

	if err != nil {
		return err
	}

	_, err = kring.GetAddress()

	return err
}

func (config *BotConfig) Save() (*BotConfig, error) {

	envPath := EnvFilePath()

	newConfig, err := Load()

	if err != nil {
		return nil, err
	}

	var envFile *os.File

	_, err = os.Stat(envPath)
	if os.IsNotExist(err) {
		envFile, _ = os.Create(envPath)
	} else if err == nil {
		envFile, _ = os.OpenFile(envPath, os.O_WRONLY|os.O_TRUNC, 0644)
		envFile.Seek(0, io.SeekStart)
	}

	newConfigReflect := reflect.ValueOf(*newConfig)
	oldConfigReflect := reflect.ValueOf(*config)

	configStruct := reflect.TypeOf(BotConfig{})

	configFieldsLen := configStruct.NumField()

	updatedFields := make([]string, configFieldsLen)
	fieldNames := make([]string, configFieldsLen)

	for i := 0; i < configFieldsLen; i++ {

		fieldNewConf := newConfigReflect.Field(i)
		fieldOldConf := oldConfigReflect.Field(i)
		fieldNames[i] = configStruct.Field(i).Name

		if !reflect.DeepEqual(fieldNewConf.Interface(), fieldOldConf.Interface()) && len(fieldNewConf.String()) != 0 {
			updatedFields[i] = fieldNewConf.String()
		} else {
			if len(fieldNewConf.String()) != 0 {
				updatedFields[i] = fieldNewConf.String()
			} else if len(fieldOldConf.String()) != 0 {
				updatedFields[i] = fieldOldConf.String()
			} else {
				updatedFields[i] = ""
			}
		}

		textField := fmt.Sprintf("%s=\"%s\"\n", configStruct.Field(i).Name, updatedFields[i])

		godotenv.Load(envPath)

		_, err = envFile.WriteString(fmt.Sprintf(textField))

	}

	config, err = Load()

	return config, err
	// call load to get old config
	// if current config has new vals, use to make config struct
	// save new config struct to .env
}

func (config *BotConfig) ToMap() map[string]string {

	configMap := make(map[string]string)

	configStruct := reflect.TypeOf(BotConfig{})

	confligReflect := reflect.ValueOf(*config)
	configFieldsLen := configStruct.NumField()

	for i := 0; i < configFieldsLen; i++ {
		configMap[configStruct.Field(i).Name] = confligReflect.Field(i).String()
	}

	return configMap
}

func DeleteConfigFile() error {
	err := os.Remove(EnvFilePath())
	return err
}
