// Package config implements getting params from env/flags/config file
package config

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
)

type AppParams struct {
	Address              *string `env:"ADDRESS" json:"address"`                 //address to send metrics
	ReportInterval       *int    `env:"REPORT_INTERVAL" json:"report_interval"` //send report  interval
	PollInterval         *int    `env:"POLL_INTERVAL" json:"poll_interval"`     //poll metrics interval
	SecretKey            *string `env:"KEY" json:"-"`                           //secret key for calculate hash for request body
	RateLimit            *int    `env:"RATE_LIMIT" json:"-"`                    //parallel request limit
	ProfileServerAddress *string `env:"PROFILE_SERVER_ADDRESS" json:"-"`        //profile server address to run
	CryptoKey            *string `env:"CRYPTO_KEY" json:"crypto_key"`           //filepath to RSA public key
	Config               *string `env:"CONFIG" json:"-"`                        //filepath to config file
	UseGRPC              *bool   `env:"USE_GRPC"`                               //use gRPC client to send metrics (http client by default)
}

var Params *AppParams = &AppParams{}

func parseEnv() error {
	if err := env.Parse(Params); err != nil {
		return err
	}
	return nil
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

var serverAddressFlag *string
var pollIntervalFlag *int
var reportIntervalFlag *int
var secretKeyFlag *string
var rateLimitFlag *int
var profileServerAddressFlag *string
var cryptoKeyFlag *string
var configFlag *string
var useGRPCFlag *bool

func parseFlags() {
	serverAddressFlag = flag.String("a", "localhost:8080", "address and port to send reports")
	pollIntervalFlag = flag.Int("p", 2, "poll interval in seconds")
	reportIntervalFlag = flag.Int("r", 10, "report interval in seconds")
	secretKeyFlag = flag.String("k", "", "secret key for request signing")
	rateLimitFlag = flag.Int("l", 0, "max parallel request limit, 0 = send in batch")
	profileServerAddressFlag = flag.String("ad", "", "profile server address to listen")
	cryptoKeyFlag = flag.String("crypto-key", "", "Full filepath to RSA private key")
	useGRPCFlag = flag.Bool("g", false, "Use gRPC client to send metrics")
	configFlag = flag.String("config", "", "filepath to config file")
	flag.StringVar(configFlag, "c", *configFlag, "alias for -config")
	flag.Parse()
}

func SetFlags() {
	if Params.Address == nil {
		Params.Address = serverAddressFlag
	}
	if Params.Address == nil {
		panic("server address is not set")
	}
	if Params.PollInterval == nil {
		Params.PollInterval = pollIntervalFlag
	}
	if Params.PollInterval == nil {
		panic("poll interval is not set")
	}
	if Params.ReportInterval == nil {
		Params.ReportInterval = reportIntervalFlag
	}
	if Params.SecretKey == nil {
		Params.SecretKey = secretKeyFlag
	}
	if Params.RateLimit == nil {
		Params.RateLimit = rateLimitFlag
	}
	if Params.ProfileServerAddress == nil {
		Params.ProfileServerAddress = profileServerAddressFlag
	}
	if Params.CryptoKey == nil {
		Params.CryptoKey = cryptoKeyFlag
	}
	if Params.UseGRPC == nil {
		Params.UseGRPC = useGRPCFlag
	}
	if Params.Config == nil {
		Params.Config = configFlag
	}
}

func Initialize() {
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
