package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/newrelic/go-agent/v3/newrelic"
	"golang.org/x/net/context/ctxhttp"
)

// WeatherReport defines a weather report
type WeatherReport struct {
	Latitude  float64            `json:"latitude"`
	Longitude float64            `json:"longitude"`
	Currently WeatherDataPoint   `json:"currently"`
	Daily     WeatherDataBlock   `json:"daily"`
	Hourly    []WeatherDataPoint `json:"hourly"`
	Minutely  []MinuteDataPoint  `json:"minutely"`
	APICalls  int                `json:"apicalls"`
	Code      int                `json:"code"`
}

// WeatherDataBlock defines a group of data points
type WeatherDataBlock struct {
	Summary string             `json:"summary"`
	Icon    string             `json:"icon"`
	Data    []WeatherDataPoint `json:"data"`
}

// WeatherDataPoint defines a weather data point
type WeatherDataPoint struct {
	Time                int64   `json:"time"`
	Summary             string  `json:"summary"`
	Icon                string  `json:"icon"`
	PrecipIntensity     float64 `json:"precipIntensity"`
	PrecipIntensityMax  float64 `json:"precipIntensityMax"`
	PrecipProbability   float64 `json:"precipProbability"`
	PrecipType          string  `json:"precipType"`
	PrecipAccumulation  float64 `json:"precipAccumulation"`
	Temperature         float64 `json:"temperature"`
	TemperatureMin      float64 `json:"temperatureMin"`
	TemperatureMax      float64 `json:"temperatureMax"`
	ApparentTemperature float64 `json:"apparentTemperature"`
	WindSpeed           float64 `json:"windSpeed"`
	WindGust            float64 `json:"windGust"`
	WindBearing         float64 `json:"windBearing"`
	CloudCover          float64 `json:"cloudCover"`
	Humidity            float64 `json:"humidity"`
	Pressure            float64 `json:"pressure"`
	Visibility          float64 `json:"visibility"`
	Ozone               float64 `json:"ozone"`
	UVIndex             float64 `json:"uvindex"`
}

type MinuteDataPoint struct {
	DateTime      int64   `json:"dt"`
	Precipitation float64 `json:"precipitation"`
}

type WeatherRequest struct {
	Latitude  string `json:"lat"`
	Longitude string `json:"long"`
}

type OpenWeatherRequest struct {
	Lat     string `json:"lat"`     // Latitude
	Long    string `json:"lon"`     // Longitude
	Exclude string `json:"exclude"` // Exclude data from response
	Units   string `json:"units"`   // Units to use
	AppId   string `json:"appid"`   // Application id / token
}

type OpenWeatherResponse struct {
	Lat            float64 `json:"lat"`
	Lon            float64 `json:"lon"`
	Timezone       string  `json:"timezone"`
	TimezoneOffset int     `json:"timezone_offset"`
	Current        struct {
		Dt        int     `json:"dt"`
		Sunrise   int     `json:"sunrise"`
		Sunset    int     `json:"sunset"`
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		Pressure  int     `json:"pressure"`
		Humidity  int     `json:"humidity"`
		DewPoint  float64 `json:"dew_point"`
		Uvi       float64 `json:"uvi"`
		Clouds    int     `json:"clouds"`
		Rain      struct {
			LastHour float64 `json:"1h"`
		}
		Visibility int     `json:"visibility"`
		WindSpeed  float64 `json:"wind_speed"`
		WindDeg    int     `json:"wind_deg"`
		Weather    []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
			Icon        string `json:"icon"`
		} `json:"weather"`
	} `json:"current"`
	Hourly []struct {
		Dt         int     `json:"dt"`
		Temp       float64 `json:"temp"`
		FeelsLike  float64 `json:"feels_like"`
		Pressure   int     `json:"pressure"`
		Humidity   int     `json:"humidity"`
		DewPoint   float64 `json:"dew_point"`
		Uvi        float64 `json:"uvi"`
		Clouds     int     `json:"clouds"`
		Visibility int     `json:"visibility"`
		WindSpeed  float64 `json:"wind_speed"`
		WindDeg    float64 `json:"wind_deg"`
		WindGust   float64 `json:"wind_gust"`
		Weather    []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
			Icon        string `json:"icon"`
		} `json:"weather"`
		Pop  float64 `json:"pop"`
		Rain struct {
			OneH float64 `json:"1h"`
		} `json:"rain,omitempty"`
	} `json:"hourly"`
	Daily []struct {
		Dt        int     `json:"dt"`
		Sunrise   int     `json:"sunrise"`
		Sunset    int     `json:"sunset"`
		Moonrise  int     `json:"moonrise"`
		Moonset   int     `json:"moonset"`
		MoonPhase float64 `json:"moon_phase"`
		Temp      struct {
			Day   float64 `json:"day"`
			Min   float64 `json:"min"`
			Max   float64 `json:"max"`
			Night float64 `json:"night"`
			Eve   float64 `json:"eve"`
			Morn  float64 `json:"morn"`
		} `json:"temp"`
		FeelsLike struct {
			Day   float64 `json:"day"`
			Night float64 `json:"night"`
			Eve   float64 `json:"eve"`
			Morn  float64 `json:"morn"`
		} `json:"feels_like"`
		Pressure  int     `json:"pressure"`
		Humidity  int     `json:"humidity"`
		DewPoint  float64 `json:"dew_point"`
		WindSpeed float64 `json:"wind_speed"`
		WindDeg   int     `json:"wind_deg"`
		WindGust  float64 `json:"wind_gust"`
		Weather   []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
			Icon        string `json:"icon"`
		} `json:"weather"`
		Clouds int     `json:"clouds"`
		Pop    float64 `json:"pop"`
		Rain   float64 `json:"rain"`
		Uvi    float64 `json:"uvi"`
	} `json:"daily"`
	Minutely []struct {
		Dt            int     `json:"dt"`
		Precipitation float64 `json:"precipitation"`
	} `json:"minutely"`
	Alerts []struct {
		SenderName  string   `json:"sender_name"`
		Event       string   `json:"event"`
		Start       float64  `json:"start"`
		End         float64  `json:"end"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	} `json:"alerts"`
}

// GetWeatherReport godoc
// @Summary Gets the current and forecasted weather for the given location
// @Description Gets the current and forecasted weather for the given location
// @Tags dashboard
// @Accept  json
// @Produce  json
// @Param config body api.WeatherRequest true "The location to fetch data for"
// @Success 200 {object} api.WeatherReport
// @Failure 400 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /weather [post]
func (s Service) GetWeatherReport(rw http.ResponseWriter, req *http.Request) {

	txn := newrelic.FromContext(req.Context())
	defer txn.StartSegment("Weather GetWeatherReport").End()

	//	The default return value
	retval := WeatherReport{}

	//	Parse the request
	request := WeatherRequest{}
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		zlog.Errorw(
			"Problem decoding request",
			"error", err,
		)
		txn.NoticeError(err)
		sendErrorResponse(rw, err, http.StatusBadRequest)
		return
	}

	//	Make sure we have minimum args:
	if request.Latitude == "" || request.Longitude == "" {
		sendErrorResponse(rw, fmt.Errorf("you must include a valid lat and long param"), http.StatusBadRequest)
		return
	}

	//	Get the api key:
	apikey := os.Getenv("OPENWEATHER_API_KEY")
	if apikey == "" {
		zlog.Errorw(
			"{OPENWEATHER_API_KEY} key is blank but shouldn't be",
		)
		sendErrorResponse(rw, fmt.Errorf("{OPENWEATHER_API_KEY} key is blank but shouldn't be"), http.StatusBadRequest)
		return
	}

	//	Create our request:
	clientRequest, err := http.NewRequest("GET", "https://api.openweathermap.org/data/2.5/onecall", nil)
	if err != nil {
		zlog.Errorw(
			"problem creating request to openweather",
			"error", err,
		)
		txn.NoticeError(err)
		return
	}

	q := clientRequest.URL.Query()
	q.Add("lat", request.Latitude)
	q.Add("lon", request.Longitude)
	q.Add("units", "imperial")
	q.Add("appid", apikey)
	clientRequest.URL.RawQuery = q.Encode()

	//	Set our headers
	clientRequest.Header.Set("Content-Type", "application/json; charset=UTF-8")
	clientRequest = newrelic.RequestWithTransactionContext(clientRequest, txn)

	//	Execute the request
	client := &http.Client{}
	client.Transport = newrelic.NewRoundTripper(client.Transport)
	clientResponse, err := ctxhttp.Do(req.Context(), client, clientRequest)
	if err != nil {
		zlog.Errorw(
			"error when sending request to OpenWeather API server",
			"err", err,
		)
		txn.NoticeError(err)
		return
	}
	defer clientResponse.Body.Close()

	//	Decode the response:
	owResponse := OpenWeatherResponse{}
	err = json.NewDecoder(clientResponse.Body).Decode(&owResponse)
	if err != nil {
		zlog.Errorw(
			"problem decoding the response from the OpenWeather API server",
			"err", err,
		)
		txn.NoticeError(err)
		return
	}

	//	Get the daily points:
	dailyPoints := []WeatherDataPoint{}

	//	Set the weather data points:
	for _, item := range owResponse.Daily {

		//	Grab the current daily point
		dailyPoint := WeatherDataPoint{
			ApparentTemperature: item.FeelsLike.Day,
			CloudCover:          float64(item.Clouds),
			Humidity:            float64(item.Humidity),
			Icon:                item.Weather[0].Icon,
			UVIndex:             item.Uvi,
			PrecipAccumulation:  item.Rain,
			PrecipProbability:   item.Pop,
			Pressure:            float64(item.Pressure),
			Summary:             item.Weather[0].Description,
			TemperatureMin:      item.Temp.Min,
			Temperature:         item.Temp.Day,
			TemperatureMax:      item.Temp.Max,
			Time:                int64(item.Dt),
			WindBearing:         float64(item.WindDeg),
			WindGust:            item.WindGust,
			WindSpeed:           item.WindSpeed,
		}

		//	Add our daily point:
		dailyPoints = append(dailyPoints, dailyPoint)
	}

	//	Get the hourly points:
	hourlyPoints := []WeatherDataPoint{}

	//	Set the weather data points:
	for _, item := range owResponse.Hourly {

		//	Grab the current hourly point
		hourlyPoint := WeatherDataPoint{
			ApparentTemperature: item.FeelsLike,
			CloudCover:          float64(item.Clouds),
			Humidity:            float64(item.Humidity),
			Icon:                item.Weather[0].Icon,
			UVIndex:             item.Uvi,
			PrecipAccumulation:  item.Rain.OneH,
			PrecipProbability:   item.Pop,
			Pressure:            float64(item.Pressure),
			Summary:             item.Weather[0].Description,
			Temperature:         item.Temp,
			TemperatureMax:      item.Temp,
			Time:                int64(item.Dt),
			WindBearing:         item.WindDeg,
			WindGust:            item.WindGust,
			WindSpeed:           item.WindSpeed,
		}

		//	Add our point:
		hourlyPoints = append(hourlyPoints, hourlyPoint)
	}

	//	Get the minutely points:
	minutelyPoints := []MinuteDataPoint{}

	//	Set the weather data points:
	for _, item := range owResponse.Minutely {

		//	Grab the current hourly point
		minutePoint := MinuteDataPoint{
			DateTime:      int64(item.Dt),
			Precipitation: item.Precipitation,
		}

		//	Add our point:
		minutelyPoints = append(minutelyPoints, minutePoint)
	}

	//	Format our weather report
	retval = WeatherReport{
		Latitude:  owResponse.Lat,
		Longitude: owResponse.Lon,
		Currently: WeatherDataPoint{
			ApparentTemperature: owResponse.Current.FeelsLike,
			CloudCover:          float64(owResponse.Current.Clouds),
			Humidity:            float64(owResponse.Current.Humidity),
			Icon:                owResponse.Current.Weather[0].Icon,
			UVIndex:             owResponse.Current.Uvi,
			PrecipAccumulation:  owResponse.Current.Rain.LastHour,
			Pressure:            float64(owResponse.Current.Pressure),
			Summary:             owResponse.Current.Weather[0].Description,
			Temperature:         owResponse.Current.Temp,
			Time:                int64(owResponse.Current.Dt),
			WindBearing:         float64(owResponse.Current.WindDeg),
			WindSpeed:           owResponse.Current.WindSpeed,
		},
		Hourly:   hourlyPoints,
		Minutely: minutelyPoints,
		Daily: WeatherDataBlock{
			Data: dailyPoints,
		},
	}

	//	Serialize to JSON & return the response:
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(rw).Encode(retval)
}
