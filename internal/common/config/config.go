package config

import (
	"encoding/json"
	"os"

	"github.com/sirupsen/logrus"
)

var _Conf Config

func SetConfig(c Config) {
	_Conf = c
}

func GetConfig() *Config {
	return &_Conf
}

type Config struct {
	DeploymentEnv          string                `json:"deployment_env"`
	ServiceName            string                `json:"service_name"`
	ServiceGroupName       string                `json:"service_group_name"`
	ServiceWsEndpoint      string                `json:"service_ws_endpoint"`
	ServiceMetricsEndpoint string                `json:"service_metrics_endpoint"`
	LogLevel               string                `json:"log_level"`
	LogSentryDSN           string                `json:"log_sentry_dsn"`
	LogPrinter             string                `json:"log_printer"`
	LogPrinterFilePath     string                `json:"log_printer_filepath"`
	ServiceInternalConfig  ServiceInternalConfig `json:"service_internal_config"`
}

type Storage struct {
	Endpoint    string `json:"endpoint"`
	RootUsr     string `json:"root_usr"`
	RootPwd     string `json:"root_pwd"`
	EnableSSL   bool   `json:"enable_ssl"`
	DB          string `json:"db"`
	ConnTimeout int    `json:"conn_timeout"`
}

type Cache struct {
	Endpoint    string `json:"endpoint"`
	Pwd         string `json:"pwd"`
	EnableSSL   bool   `json:"enable_ssl"`
	DB          int    `json:"db"`
	ConnTimeout int    `json:"conn_timeout"`
}

type ServiceInternalConfig struct {
	Storage Storage `json:"storage"`
	Cache   Cache   `json:"cache"`
}

func (conf *Config) UnmarshalJSON(data []byte) error {
	type Alias Config
	aux := &struct {
		*Alias
	}{Alias: (*Alias)(conf)}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	if aux.ServiceInternalConfig.Storage.RootPwd == "STORAGE_PWD" {
		conf.ServiceInternalConfig.Storage.RootPwd = os.Getenv("STORAGE_PWD")
	}
	if aux.ServiceInternalConfig.Cache.Pwd == "CACHE_PWD" {
		conf.ServiceInternalConfig.Cache.Pwd = os.Getenv("CACHE_PWD")
	}

	return nil
}

func loadConfigFile(fn string) error {
	data, err := os.ReadFile(fn)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &_Conf)
}

func LoadConfigFileOrPanic(fn string) *Config {
	if err := loadConfigFile(fn); err != nil {
		logrus.WithError(err).Fatalf("Failed to load config file:%s.", fn)
	} else {
		logrus.Debugf("Loaded config file:%s.", fn)
	}
	// print.PrettyPrintStruct(_Conf, 1, 4)
	return &_Conf
}
