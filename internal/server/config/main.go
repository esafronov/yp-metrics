// Package config implements getting params from env/flags/config file
package config

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
)

type AppParams struct {
	Address              *string `env:"ADDRESS" json:"address"`              //server address to listen
	StoreInterval        *int    `env:"STORE_INTERVAL" json:"restore"`       //store interval
	FileStoragePath      *string `env:"FILE_STORAGE_PATH" json:"store_file"` //file storage path
	Restore              *bool   `env:"RESTORE"`                             //restore or not data on start
	DatabaseDsn          *string `env:"DATABASE_DSN" json:"database_dsn"`    //db connection dsn
	SecretKey            *string `env:"KEY"`                                 //secret key for signature check
	ProfileServerAddress *string `env:"PROFILE_SERVER_ADDRESS"`              //profile serveraddress to listen
	CryptoKey            *string `env:"CRYPTO_KEY" json:"crypto_key"`        //Full filepath to RSA private key
	Config               *string `env:"CONFIG" json:"-"`                     //filepath to config file
}

var Params *AppParams = &AppParams{}

func parseEnv() error {
	if err := env.Parse(Params); err != nil {
		return err
	}
	return nil
}

var serverAddressFlag *string
var storeIntervalFlag *int
var fileStoragePathFlag *string
var restoreDataFlag *bool
var databaseDsnFlag *string
var secretKeyFlag *string
var profileServerAddressFlag *string
var cryptoKeyFlag *string
var configFlag *string

func parseFlags() {
	serverAddressFlag = flag.String("a", "localhost:8080", "address and port to run server")
	storeIntervalFlag = flag.Int("i", 300, "interval for backuping data")
	fileStoragePathFlag = flag.String("f", "", "filepath to store backup data")
	restoreDataFlag = flag.Bool("r", true, "restore data on server start")
	databaseDsnFlag = flag.String("d", "", "database dsn")
	secretKeyFlag = flag.String("k", "", "secret key for signature check")
	profileServerAddressFlag = flag.String("ad", "", "profile server address to listen")
	cryptoKeyFlag = flag.String("crypto-key", "", "Full filepath to RSA private key")
	configFlag = flag.String("config", "", "filepath to config file")
	flag.StringVar(configFlag, "c", *configFlag, "alias for -config")
	flag.Parse()
}

func SetFlags() {
	if Params.Address == nil {
		Params.Address = serverAddressFlag
	}
	if Params.StoreInterval == nil {
		Params.StoreInterval = storeIntervalFlag
	}
	if Params.FileStoragePath == nil {
		Params.FileStoragePath = fileStoragePathFlag
	}
	if Params.Restore == nil {
		Params.Restore = restoreDataFlag
	}
	if Params.DatabaseDsn == nil {
		Params.DatabaseDsn = databaseDsnFlag
	}
	if Params.SecretKey == nil {
		Params.SecretKey = secretKeyFlag
	}
	if Params.ProfileServerAddress == nil {
		Params.ProfileServerAddress = profileServerAddressFlag
	}
	if Params.CryptoKey == nil {
		Params.CryptoKey = cryptoKeyFlag
	}
	if Params.Config == nil {
		Params.Config = configFlag
	}
}

func parseConfigFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	if err = json.NewDecoder(f).Decode(Params); err != nil {
		return err
	}
	return nil
}

func init() {
	parseFlags()
	configFileName, existsInEnv := os.LookupEnv("CONFIG")
	if !existsInEnv {
		if f := flag.Lookup("config"); f.Value != nil {
			configFileName = f.Value.String()
		} else if f := flag.Lookup("c"); f != nil {
			configFileName = f.Value.String()
		}
	}
	if configFileName != "" {
		err := parseConfigFile(configFileName)
		if err != nil {
			panic(err.Error())
		}
	}
	err := parseEnv()
	if err != nil {
		panic(err.Error())
	}
	SetFlags()
}
