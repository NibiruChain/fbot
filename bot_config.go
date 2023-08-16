package fbot

import (
	"fmt"
	"io"
	"os"
	"strings"

	"reflect"

	"github.com/Unique-Divine/gonibi"
	"github.com/joho/godotenv"
)

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

func Load() *BotConfig {
	godotenv.Load(".env")

	var newConfig = &BotConfig{
		MNEMONIC:       os.Getenv("MNEMONIC"),
		CHAIN_ID:       os.Getenv("CHAIN_ID"),
		GRPC_ENDPOINT:  os.Getenv("GRPC_ENDPOINT"),
		TMRPC_ENDPOINT: os.Getenv("TMRPC_ENDPOINT"),
	}

	return newConfig
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

func (config *BotConfig) CheckConfig() (err error) {

	reflectConfig := reflect.ValueOf(*config)

	for i := 0; i < reflectConfig.NumField(); i++ {
		field := reflectConfig.Field(i)

		if field.Interface() == reflect.Zero(field.Type()).Interface() {
			return fmt.Errorf("Undefined Bot Config Field")
		}
	}

	_, _, err = gonibi.CreateSigner(config.MNEMONIC,
		gonibi.NewKeyring(), "test")

	return err
}

func (config *BotConfig) Save() (*BotConfig, error) {

	newConfig := Load()

	var envFile *os.File

	_, err := os.Stat(".env")
	if os.IsNotExist(err) {
		envFile, _ = os.Create(".env")
	} else if err == nil {
		// .env exists, append or update existing
		envFile, _ = os.OpenFile(".env", os.O_WRONLY|os.O_TRUNC, 0644)
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

		if !reflect.DeepEqual(fieldNewConf.Interface(), fieldOldConf.Interface()) {
			updatedFields[i] = fieldNewConf.String()
		} else {
			if fieldNewConf.String() != "" {
				updatedFields[i] = fieldNewConf.String()
			} else if fieldOldConf.String() != "" {
				updatedFields[i] = fieldOldConf.String()
			} else {
				updatedFields[i] = "MISSING"
			}
		}

		textField := strings.ToUpper(configStruct.Field(i).Name) + " = " + updatedFields[i] + "\n"
		godotenv.Load(".env")

		_, err = envFile.WriteString(fmt.Sprintf(textField))

	}

	config = Load()

	return config, nil
	// call load to get old config
	// if current config has new vals, use to make config struct
	// save new config struct to .env
}
