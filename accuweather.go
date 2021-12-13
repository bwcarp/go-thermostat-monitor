package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// WriteWeather - write weather metrics to InfluxDB
func AccuWeatherWriteWeather(
	config ConfigRoot,
	influxClient influxdb2.Client) {

	// Weather - parse weather details
	type Weather struct {
		Timestamp   string `json:"LocalObservationDateTime"`
		Temperature struct {
			Metric struct {
				Value float32 `json:"Value"`
			} `json:"Metric"`
		} `json:"Temperature"`
		Humidity int `json:"RelativeHumidity"`
		Pressure struct {
			Metric struct {
				Value float32 `json:"Value"`
			} `json:"Metric"`
		} `json:"Pressure"`
		Wind struct {
			Speed struct {
				Metric struct {
					Value float32 `json:"Value"`
				} `json:"Metric"`
			} `json:"Speed"`
		} `json:"Wind"`
	}

	writer := influxClient.WriteAPIBlocking(config.InfluxConfig.Org, config.InfluxConfig.Bucket)

	url := fmt.Sprintf(
		"https://dataservice.accuweather.com/currentconditions/v1/%d?apikey=%s&details=true",
		config.AccuWeatherConfig.Location,
		config.AccuWeatherConfig.APIKey,
	)
	for {
		httpClient := &http.Client{Timeout: time.Second * 10}
		res, err := httpClient.Get(url)
		if err != nil {
			log.Print(err)
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)

		var response []Weather
		jsonErr := json.Unmarshal(body, &response)
		if jsonErr != nil {
			log.Print(string(body))
			log.Println("ERROR: Could not unmarshal json!")
			log.Print(jsonErr)
		} else {
			weather := response[0]
			timestamp, _ := time.Parse(time.RFC3339, weather.Timestamp)

			p := influxdb2.NewPoint(
				"accuweather",
				map[string]string{
					"locationKey": fmt.Sprint(config.AccuWeatherConfig.Location),
				},
				map[string]interface{}{
					"temperature": weather.Temperature.Metric.Value,
					"humidity":    weather.Humidity,
					"pressure":    weather.Pressure.Metric.Value,
					"windspeed":   weather.Wind.Speed.Metric.Value,
				},
				timestamp,
			)

			err := writer.WritePoint(context.Background(), p)
			if err != nil {
				log.Println("ERROR: Could not write data point for AccuWeather!")
				log.Print(err)
			} else {
				log.Printf("Wrote weather metrics from AccuWeather. Sleeping for %d minute(s).\n", config.AccuWeatherConfig.Interval)
			}
		}
		time.Sleep(time.Minute * time.Duration(config.AccuWeatherConfig.Interval))
	}
}
