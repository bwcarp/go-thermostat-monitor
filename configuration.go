package main

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// ConfigRoot - load yaml data from config file
type ConfigRoot struct {
	InfluxConfig         InfluxConfig         `yaml:"influx"`
	NestConfig           NestConfig           `yaml:"nest"`
	EcobeeConfig         EcobeeConfig         `yaml:"ecobee"`
	AccuWeatherConfig    AccuWeatherConfig    `yaml:"accuweather"`
	OpenWeatherMapConfig OpenWeatherMapConfig `yaml:"openweathermap"`
	WeatherGovConfig     WeatherGovConfig     `yaml:"NWS"`
}

// InfluxConfig - InfluxDB configuration
type InfluxConfig struct {
	Url    string `yaml:"url"`
	Bucket string `yaml:"bucket"`
	Token  string `yaml:"token"`
	Org    string `yaml:"org"`
}

// EcobeeConfig - Ecobee configuration
type EcobeeConfig struct {
	Enabled      bool   `yaml:"enable"`
	Interval     int    `yaml:"interval"`
	APIKey       string `yaml:"api_key"`
	RefreshToken string `yaml:"refresh_token"`
}

// NestConfig - Google Nest configuration
type NestConfig struct {
	Enabled      bool   `yaml:"enable"`
	Interval     int    `yaml:"interval"`
	ProjectID    string `yaml:"project_id"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	RefreshToken string `yaml:"refresh_token"`
	RedirectUri  string `yaml:"redirect_uri"`
}

// AccuWeatherConfig - AccuWeather configuration
type AccuWeatherConfig struct {
	Enabled  bool   `yaml:"enable"`
	Interval int    `yaml:"interval"`
	APIKey   string `yaml:"api_key"`
	Location int    `yaml:"location_key"`
}

// OpenWeatherMapConfig - OpenWeatherMap configuration
type OpenWeatherMapConfig struct {
	Enabled  bool   `yaml:"enable"`
	Interval int    `yaml:"interval"`
	AppID    string `yaml:"app_id"`
	CityID   int    `yaml:"city_id"`
}

// WeatherGovConfig - weather.gov configuration
type WeatherGovConfig struct {
	Enabled  bool   `yaml:"enable"`
	Interval int    `yaml:"interval"`
	Station  string `yaml:"station"`
}

// GetConfig - read config file
func GetConfig(filePath string) ConfigRoot {
	configFile, _ := ioutil.ReadFile(filePath)
	configRoot := ConfigRoot{}
	err := yaml.Unmarshal(configFile, &configRoot)
	if err != nil {
		log.Print(err)
		log.Fatal("Could not parse config file.")
	}
	return configRoot
}
