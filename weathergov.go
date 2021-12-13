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
func NwsWriteWeather(
	config ConfigRoot,
	influxClient influxdb2.Client) {

	// Weather - parse json data for value and unit
	type Weather struct {
		Value          float32 `json:"value"`
		UnitCode       string  `json:"unitCode"`
		QualityControl string  `json:"qualityControl"`
	}

	// Response - root of JSON object returned by weather.gov
	type Response struct {
		Properties struct {
			Timestamp   string  `json:"timestamp"`
			Temperature Weather `json:"temperature"`
			Humidity    Weather `json:"relativeHumidity"`
			Pressure    Weather `json:"barometricPressure"`
			Windspeed   Weather `json:"windSpeed"`
		} `json:"properties"`
	}

	url := fmt.Sprintf(
		"https://api.weather.gov/stations/%s/observations/latest",
		config.WeatherGovConfig.Station,
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

			var fields = make(map[string]interface{})

			// weather.gov sometimes reports a value of 0 when it doesn't have data.
			// Given that 0 humidity never happens, 0 pressure means we all die,
			// and a floating point value being exactly 0 for temperature is rare,
			// it's better to pass null values instead.
			timestamp, _ := time.Parse(time.RFC3339, weather.Properties.Timestamp)
			if weather.Properties.Temperature.Value != 0 {
				fields["temperature"] = weather.Properties.Temperature.Value
			}
			if weather.Properties.Humidity.Value > 0 {
				fields["humidity"] = weather.Properties.Humidity.Value
			}
			if weather.Properties.Pressure.Value > 0 {
				// Convert Pa to hPa for consistency with other apps
				fields["pressure"] = weather.Properties.Pressure.Value * 0.01
			}
			if weather.Properties.Windspeed.Value > 0 {
				fields["windspeed"] = weather.Properties.Windspeed.Value
			}

			p := influxdb2.NewPoint(
				"weathergov",
				map[string]string{
					"station": config.WeatherGovConfig.Station,
				},
				fields,
				timestamp,
			)

			err := writer.WritePoint(context.Background(), p)
			if err != nil {
				log.Println("ERROR: Could not write data point for NWS!")
				log.Print(err)
			} else {
				log.Printf("Wrote weather metrics from NWS. Sleeping for %d minute(s).\n", config.WeatherGovConfig.Interval)
			}
		}
		time.Sleep(time.Minute * time.Duration(config.WeatherGovConfig.Interval))
	}
}
