package main

import (
	"flag"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

func main() {

	var configFile = flag.String("c", "./config.yaml", "Specify path to config.json")
	flag.Parse()

	config := GetConfig(*configFile)
	influxClient := influxdb2.NewClient(config.InfluxConfig.Url, config.InfluxConfig.Token)

	// Weather agents
	if config.AccuWeatherConfig.Enabled {
		go AccuWeatherWriteWeather(config, influxClient)
	}
	if config.OpenWeatherMapConfig.Enabled {
		go OpenWeatherMapWriteWeather(config, influxClient)
	}
	if config.WeatherGovConfig.Enabled {
		go NwsWriteWeather(config, influxClient)
	}
	// Thermostat agents
	if config.NestConfig.Enabled {
		go NestRefreshLogin(config)
		time.Sleep(time.Second * 10)
		go WriteNest(config, influxClient)
	}

	for {
		time.Sleep(time.Hour * 1)
	}
}
