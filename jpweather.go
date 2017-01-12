package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"time"

	yaml "gopkg.in/yaml.v2"

	forecast "github.com/mlbright/forecast/v2"
	"github.com/olekukonko/tablewriter"
)

type config struct {
	ForecaseApiKey string `yaml:"forecastApiKey"`
	Home           struct {
		Lat string `yaml:lat`
		Lng string `yaml:lng`
	} `yaml:home`
}

func loadConfig() (*config, error) {
	home := os.Getenv("HOME")
	if home == "" && runtime.GOOS == "windows" {
		home = os.Getenv("APPDATA")
	}

	fname := filepath.Join(home, ".config", "jpweather", "config.yml")
	buf, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	var cfg config
	err = yaml.Unmarshal(buf, &cfg)
	return &cfg, err
}

func showDay(time time.Time, w io.Writer) {
	fmt.Fprintf(w, "┌──────────────┐\n")
	fmt.Fprintf(w, "│ %d-%02d-%d   │\n", time.Year(), time.Month(), time.Day())
	fmt.Fprintf(w, "└──────────────┘\n")
}

func convertMilePerHourToMS(windSpeed float64) float64 {
	return windSpeed * 0.447
}

func convertDegToCompass(deg float64) string {
	compasses := []string{"北", "北北東", "北東", "東北東", "東", "東南東", "南東", "南南東", "南", "南南西", "南西", "西南西", "西", "西北西", "北西", "北北西"}
	val := int((deg / 22.5) + .5)
	return compasses[(val % 16)]
}

func main() {
	config, err := loadConfig()

	if err != nil {
		fmt.Printf("Config file load Error: %v\nPlease create a config file.\n", err)
		os.Exit(1)
	}

	times := []string{"時間"}
	weathers := []string{"天気"}
	temperatures := []string{"気温"}
	windBearings := []string{"風向"}
	windSpeeds := []string{"風速(m/s)"}
	precipProbabilitys := []string{"降水確率(%)"}
	var showDays int

	needShowDay := true
	w := os.Stdout
	table := tablewriter.NewWriter(w)
	weatherConversionTable := map[string]string{
		"clear-day":           "☀",
		"clear-night":         "🌙",
		"rain":                "☔",
		"snow":                "☃",
		"sleet":               "❄",
		"wind":                "🍃",
		"fog":                 "🌁",
		"cloudy":              "☁",
		"partly-cloudy-day":   "☀/☁",
		"partly-cloudy-night": "🌙/☁",
		"hail":                "❅",
		"thunderstorm":        "☇",
	}

	f, err := forecast.Get(config.ForecaseApiKey, config.Home.Lat, config.Home.Lng, "now", forecast.AUTO)
	if err != nil {
		fmt.Printf("API Error: %v\n", err)
		os.Exit(1)
	}

	for _, data := range f.Hourly.Data {
		weatherTime := time.Unix(int64(data.Time), 0)
		if needShowDay {
			showDay(weatherTime, w)
			needShowDay = false
		}

		times = append(times, fmt.Sprintf("%02d", weatherTime.Hour()))
		weathers = append(weathers, weatherConversionTable[data.Icon])
		temperatures = append(temperatures, fmt.Sprintf("%.1f", float32(data.Temperature)))
		precipProbabilitys = append(precipProbabilitys, fmt.Sprintf("%v", float32(data.PrecipProbability)*1000))
		windBearings = append(windBearings, convertDegToCompass(data.WindBearing))
		windSpeeds = append(windSpeeds, fmt.Sprintf("%.1f", convertMilePerHourToMS(float64(data.WindSpeed))))

		if weatherTime.Hour() == 23 {
			table.SetHeader(times)
			table.Append(weathers)
			table.Append(temperatures)
			table.Append(precipProbabilitys)
			table.Append(windBearings)
			table.Append(windSpeeds)
			table.SetBorder(false)
			table.Render()
			table = tablewriter.NewWriter(w)

			fmt.Fprintf(w, "\n\n")

			needShowDay = true
			times = []string{"時間"}
			weathers = []string{"天気"}
			temperatures = []string{"気温"}
			windBearings = []string{"風向"}
			windSpeeds = []string{"風速(m/s)"}
			precipProbabilitys = []string{"降水確率(%)"}
			showDays += 1
		}

		if showDays == 2 {
			break
		}
	}
}
