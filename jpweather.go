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

var weatherConversionTable = map[string]string{
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

type config struct {
	ForecaseApiKey string `yaml:"forecastApiKey"`
	Home           struct {
		Lat string `yaml:lat`
		Lng string `yaml:lng`
	} `yaml:home`
}

type weatherData struct {
	date               time.Time
	times              []string
	weathers           []string
	temperatures       []string
	windBearings       []string
	windSpeeds         []string
	precipProbabilitys []string
}

func (wd *weatherData) initialize() {
	wd.times = []string{"時間"}
	wd.weathers = []string{"天気"}
	wd.temperatures = []string{"気温"}
	wd.windBearings = []string{"風向"}
	wd.windSpeeds = []string{"風速(m/s)"}
	wd.precipProbabilitys = []string{"降水確率(%)"}
}

func (wd *weatherData) setForecaseData(data forecast.DataPoint) {
	weatherTime := time.Unix(int64(data.Time), 0)
	wd.date = weatherTime
	wd.times = append(wd.times, fmt.Sprintf("%02d", weatherTime.Hour()))
	wd.weathers = append(wd.weathers, weatherConversionTable[data.Icon])
	wd.temperatures = append(wd.temperatures, fmt.Sprintf("%.1f", float32(data.Temperature)))
	wd.precipProbabilitys = append(wd.precipProbabilitys, fmt.Sprintf("%v", float32(data.PrecipProbability)*1000))
	wd.windBearings = append(wd.windBearings, convertDegToCompass(data.WindBearing))
	wd.windSpeeds = append(wd.windSpeeds, fmt.Sprintf("%.1f", convertMilePerHourToMS(float64(data.WindSpeed))))
}

func (wd *weatherData) showDate(w io.Writer) {
	fmt.Fprintf(w, "┌──────────────┐\n")
	fmt.Fprintf(w, "│ %d-%02d-%d   │\n", wd.date.Year(), wd.date.Month(), wd.date.Day())
	fmt.Fprintf(w, "└──────────────┘\n")
}

func (wd *weatherData) showWeather(w io.Writer) {
	wd.showDate(w)
	table := tablewriter.NewWriter(w)
	table.SetHeader(wd.times)
	table.Append(wd.weathers)
	table.Append(wd.temperatures)
	table.Append(wd.precipProbabilitys)
	table.Append(wd.windBearings)
	table.Append(wd.windSpeeds)
	table.SetBorder(false)
	table.Render()
}

func convertMilePerHourToMS(windSpeed float64) float64 {
	return windSpeed * 0.447
}

func convertDegToCompass(deg float64) string {
	compasses := []string{"北", "北北東", "北東", "東北東", "東", "東南東", "南東", "南南東", "南", "南南西", "南西", "西南西", "西", "西北西", "北西", "北北西"}
	val := int((deg / 22.5) + .5)
	return compasses[(val % 16)]
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

func main() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Config file load Error: %v\nPlease create a config file.\n", err)
		os.Exit(1)
	}

	var showDays int
	var wd weatherData
	wd.initialize()

	w := os.Stdout

	f, err := forecast.Get(config.ForecaseApiKey, config.Home.Lat, config.Home.Lng, "now", forecast.SI)
	if err != nil {
		fmt.Printf("API Error: %v\n", err)
		os.Exit(1)
	}

	for _, data := range f.Hourly.Data {
		wd.setForecaseData(data)

		weatherTime := time.Unix(int64(data.Time), 0)
		if weatherTime.Hour() == 23 {
			wd.showWeather(w)
			wd.initialize()
			fmt.Fprintf(w, "\n\n")
			showDays += 1
		}

		if showDays == 2 {
			break
		}
	}
}
