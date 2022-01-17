package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// Will pass to functions as needed
var EcobeeAccessToken string

// RefreshLogin - Routinely fetch a new authentication token
func EcobeeRefreshLogin(config ConfigRoot) {

	// Authorization - unpack access_token
	type Authorization struct {
		AccessToken string `json:"access_token"`
	}

	httpClient := &http.Client{Timeout: time.Second * 10}
	authUrl := fmt.Sprintf("https://api.ecobee.com/token?grant_type=refresh_token&refresh_token=%s&client_id=%s", config.EcobeeConfig.RefreshToken, config.EcobeeConfig.APIKey)

	for {
		log.Println("Getting new ecobee access_token")
		res, err := httpClient.Post(
			authUrl,
			"application/json",
			nil,
		)
		if err != nil {
			log.Println("ERROR: Could not login to ecobee.")
			log.Fatal(err)
		}
		var authData Authorization
		body, _ := ioutil.ReadAll(res.Body)
		err = json.Unmarshal(body, &authData)
		res.Body.Close()
		if err != nil {
			log.Println("ERROR: Invalid response object from ecobee")
			log.Fatal(err)
		}
		EcobeeAccessToken = fmt.Sprintf("Bearer %s", authData.AccessToken)
		time.Sleep(time.Minute * 45)
	}
}

// WriteEcobee - write ecobee metrics to influx
func WriteEcobee(config ConfigRoot, influxClient influxdb2.Client) {

	type ThermostatSummary struct {
		StatusList []string `json:"statusList"`
	}

	type ThermostatRuntime struct {
		ActualHumidity     int    `json:"actualHumidity"`
		ActualTemperature  int    `json:"actualTemperature"`
		DesiredCool        int    `json:"desiredCool"`
		DesiredHeat        int    `json:"desiredHeat"`
		DesiredHumidity    int    `json:"desiredHumidity"`
		LastStatusModified string `json:"lastStatusModified"`
		RawTemperature     int    `json:"rawTemperature"`
	}

	type ThermostatSettings struct {
		HvacMode string `json:"hvacMode"`
	}

	type Thermostat struct {
		Identifier string             `json:"identifier"`
		Name       string             `json:"name"`
		Runtime    ThermostatRuntime  `json:"runtime"`
		Settings   ThermostatSettings `json:"settings"`
	}

	type Thermostats struct {
		ThermostatList []Thermostat `json:"thermostatList"`
	}

	writer := influxClient.WriteAPIBlocking(config.InfluxConfig.Org, config.InfluxConfig.Bucket)

	for {

		// Get main thermostat info
		url := `https://api.ecobee.com/1/thermostat?json=%7B%22selection%22%3A%20%7B%22selectionType%22%3A%20%22registered%22%2C%20%22selectionMatch%22%3A%20%22%22%2C%20includeRuntime%3A%20true%2C%20includeSettings%3A%20true%2C%20includeEvents%3A%20true%7D%7D`

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", EcobeeAccessToken)

		httpClient := &http.Client{Timeout: time.Second * 10}
		res, err := httpClient.Do(req)
		if err != nil {
			log.Println("ERROR: Could not get device info from ecobee API.")
			log.Fatal(err)
		}

		var ecobees Thermostats
		body, _ := ioutil.ReadAll(res.Body)
		err = json.Unmarshal(body, &ecobees)
		res.Body.Close()
		if err != nil {
			log.Print(string(body))
			log.Println("ERROR: Invalid json.")
			log.Fatal(err)
		}

		// Get summary list to see what components are running
		url = `https://api.ecobee.com/1/thermostatSummary?json=%7B%22selection%22%3A%20%7B%22selectionType%22%3A%20%22registered%22%2C%20%22selectionMatch%22%3A%20%22%22%2C%20%22includeEquipmentStatus%22%3A%20true%7D%7D`

		req, _ = http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", EcobeeAccessToken)

		res, err = httpClient.Do(req)
		if err != nil {
			log.Println("ERROR: Could not get device info from ecobee API.")
			log.Fatal(err)
		}

		var ecobeeSummary ThermostatSummary
		body, _ = ioutil.ReadAll(res.Body)
		err = json.Unmarshal(body, &ecobeeSummary)
		res.Body.Close()
		if err != nil {
			log.Print(string(body))
			log.Println("ERROR: Invalid json.")
			log.Fatal(err)
		}

		for _, ecobee := range ecobees.ThermostatList {

			var Tags = make(map[string]string)
			var Fields = make(map[string]interface{})

			// get whether hvac is running.
			var hvacStatus = map[string]int8{
				"heatPump":     0,
				"heatPump2":    0,
				"heatPump3":    0,
				"compCool1":    0,
				"compCool2":    0,
				"auxHeat1":     0,
				"auxHeat2":     0,
				"auxHeat3":     0,
				"fan":          0,
				"humidifier":   0,
				"dehumidifier": 0,
				"ventilator":   0,
				"economizer":   0,
				"compHotWater": 0,
				"auxHotWater":  0,
			}

			for _, summary := range ecobeeSummary.StatusList {
				parsedSummary := strings.Split(summary, ":")
				identifier := parsedSummary[0]
				if identifier == ecobee.Identifier {
					for _, mode := range strings.Split(parsedSummary[1], ",") {
						if mode != "" {
							hvacStatus[mode] = 1
						}
					}
				}
			}

			for mode, v := range hvacStatus {
				Fields[mode] = v
			}

			lastModified, _ := time.Parse("2006-01-02 15:04:05", ecobee.Runtime.LastStatusModified)

			Tags["name"] = ecobee.Name
			Tags["identifier"] = ecobee.Identifier

			Fields["humidity"] = ecobee.Runtime.ActualHumidity
			Fields["desiredHumidity"] = ecobee.Runtime.DesiredHumidity
			Fields["temperature"] = (float64(ecobee.Runtime.ActualTemperature) - 320) * 5 / 90
			Fields["rawTemperature"] = (float64(ecobee.Runtime.RawTemperature) - 320) * 5 / 90

			switch ecobee.Settings.HvacMode {
			case "heat":
				Fields["heat"] = (float64(ecobee.Runtime.DesiredHeat) - 320) * 5 / 90
			case "cool":
				Fields["cool"] = (float64(ecobee.Runtime.DesiredCool) - 320) * 5 / 90
			case "auto":
				Fields["heat"] = (float64(ecobee.Runtime.DesiredHeat) - 320) * 5 / 90
				Fields["cool"] = (float64(ecobee.Runtime.DesiredCool) - 320) * 5 / 90
			case "auxHeatOnly":
				Fields["heat"] = (float64(ecobee.Runtime.DesiredHeat) - 320) * 5 / 90
			}

			p := influxdb2.NewPoint(
				"ecobee",
				Tags,
				Fields,
				lastModified,
			)

			err := writer.WritePoint(context.Background(), p)
			if err != nil {
				log.Println("ERROR: Could not write data point!")
				log.Print(err)
			} else {
				log.Printf("Wrote ecobee thermostat metrics. Sleeping for %d minute(s).\n", config.EcobeeConfig.Interval)
			}

		}

		time.Sleep(time.Minute * time.Duration(config.EcobeeConfig.Interval))
	}
}
