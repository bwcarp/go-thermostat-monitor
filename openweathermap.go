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
func OpenWeatherMapWriteWeather(
	config ConfigRoot,
	influxClient influxdb2.Client) {

	// Response - Parse openweathermap json output
	type Response struct {
		Timestamp int64  `json:"dt"`
		City      string `json:"name"`
		Sys       struct {
			Country string `json:"country"`
		} `json:"sys"`
		Weather struct {
			Humidity    int     `json:"humidity"`
			Pressure    int     `json:"pressure"`
			Temperature float32 `json:"temp"`
		} `json:"main"`
		Wind struct {
			Speed float32 `json:"speed"`
		} `json:"wind"`
	}

	url := fmt.Sprintf(
		"https://api.openweathermap.org/data/2.5/weather?id=%d&appid=%s&units=metric",
		config.OpenWeatherMapConfig.CityID,
		config.OpenWeatherMapConfig.AppID,
	)

	writer := influxClient.WriteAPIBlocking(config.InfluxConfig.Org, config.InfluxConfig.Bucket)

	for {
		httpClient := &http.Client{Timeout: time.Second * 10}
		res, err := httpClient.Get(url)
		if err != nil {
			log.Printf("ERROR: Could not fetch %s", url)
			log.Print(err)
			continue
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)

		var weather Response
		jsonErr := json.Unmarshal(body, &weather)
		if jsonErr != nil {
			log.Println("ERROR: Could not unmarshal json!")
			log.Print(jsonErr)
		} else {
			p := influxdb2.NewPoint(
				"openweathermap",
				map[string]string{
					"city":    weather.City,
					"country": weather.Sys.Country,
				},
				map[string]interface{}{
					"temperature": weather.Weather.Temperature,
					"humidity":    weather.Weather.Humidity,
					"pressure":    weather.Weather.Pressure,
					"windspeed":   weather.Wind.Speed * 3.6, // Convert to kilometers per hour
				},
				time.Unix(weather.Timestamp, 0),
			)

			err := writer.WritePoint(context.Background(), p)
			if err != nil {
				log.Println("ERROR: Could not write data point for OpenWeatherMap!")
				log.Print(err)
			} else {
				log.Printf("Wrote weather metrics from OpenWeatherMap. Sleeping for %d minute(s).\n", config.OpenWeatherMapConfig.Interval)
			}
		}
		time.Sleep(time.Minute * time.Duration(config.OpenWeatherMapConfig.Interval))
	}
}
