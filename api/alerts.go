package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"golang.org/x/net/context/ctxhttp"
)

type AlertsRequest struct {
	Latitude  string `json:"lat"`
	Longitude string `json:"long"`
}

// NWSPointsResponse defines the expected response format from the NWS points service
type NWSPointsResponse struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Geometry struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		ID                  string `json:"@id"`
		Type                string `json:"@type"`
		Cwa                 string `json:"cwa"`
		ForecastOffice      string `json:"forecastOffice"`
		GridID              string `json:"gridId"`
		GridX               int    `json:"gridX"`
		GridY               int    `json:"gridY"`
		Forecast            string `json:"forecast"`
		ForecastHourly      string `json:"forecastHourly"`
		ForecastGridData    string `json:"forecastGridData"`
		ObservationStations string `json:"observationStations"`
		RelativeLocation    struct {
			Type     string `json:"type"`
			Geometry struct {
				Type        string    `json:"type"`
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties struct {
				City     string `json:"city"`  // City name
				State    string `json:"state"` // State name
				Distance struct {
					Value    float64 `json:"value"`
					UnitCode string  `json:"unitCode"`
				} `json:"distance"`
				Bearing struct {
					Value    int    `json:"value"`
					UnitCode string `json:"unitCode"`
				} `json:"bearing"`
			} `json:"properties"`
		} `json:"relativeLocation"`
		ForecastZone    string `json:"forecastZone"`
		County          string `json:"county"` // County
		FireWeatherZone string `json:"fireWeatherZone"`
		TimeZone        string `json:"timeZone"`
		RadarStation    string `json:"radarStation"`
	} `json:"properties"`
}

// NWSAlertsResponse defines the expected response from the NWS alerts service (for a specific zone)
type NWSAlertsResponse struct {
	Type     string `json:"type"`
	Features []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Properties struct {
			ID       string `json:"@id"`
			Type     string `json:"@type"`
			AreaDesc string `json:"areaDesc"`
			Geocode  struct {
				SAME []string `json:"SAME"`
				UGC  []string `json:"UGC"`
			} `json:"geocode"`
			AffectedZones []string `json:"affectedZones"`
			References    []struct {
				ID         string `json:"@id"`
				Identifier string `json:"identifier"`
				Sender     string `json:"sender"`
				Sent       string `json:"sent"`
			} `json:"references"`
			Sent        time.Time `json:"sent"`
			Effective   time.Time `json:"effective"`
			Onset       time.Time `json:"onset"`
			Expires     time.Time `json:"expires"`
			Ends        time.Time `json:"ends"`
			Status      string    `json:"status"`
			MessageType string    `json:"messageType"`
			Category    string    `json:"category"`
			Severity    string    `json:"severity"`
			Certainty   string    `json:"certainty"`
			Urgency     string    `json:"urgency"`
			Event       string    `json:"event"`
			Sender      string    `json:"sender"`
			SenderName  string    `json:"senderName"`
			Headline    string    `json:"headline"`
			Description string    `json:"description"`
			Instruction string    `json:"instruction"`
			Response    string    `json:"response"`
			Parameters  struct {
				PIL               []string    `json:"PIL"`
				NWSheadline       []string    `json:"NWSheadline"`
				BLOCKCHANNEL      []string    `json:"BLOCKCHANNEL"`
				EASORG            []string    `json:"EAS-ORG"`
				VTEC              []string    `json:"VTEC"`
				EventEndingTime   []time.Time `json:"eventEndingTime"`
				ExpiredReferences []string    `json:"expiredReferences"`
			} `json:"parameters"`
		} `json:"properties"`
	} `json:"features"`
	Title   string    `json:"title"`
	Updated time.Time `json:"updated"`
}

// AlertReport defines an alert report
type AlertReport struct {
	Latitude                 float64     `json:"latitude"`  // Latitude
	Longitude                float64     `json:"longitude"` // Longitude
	City                     string      `json:"city"`      // City name
	State                    string      `json:"state"`     // State name
	NWSCounty                string      `json:"county"`    // National weather service county
	ActiveAlertsForCountyURL string      `json:"alertsurl"` // URL to see active alerts on the NWS website for the current NWS zone
	Alerts                   []AlertItem `json:"alerts"`    // Active alerts
	Version                  string      `json:"version"`   // Service version
}

// AlertItem defines an individual alert item in a report
type AlertItem struct {
	Event           string    `json:"event"`            // Short event summary
	Headline        string    `json:"headline"`         // Full headline description
	Description     string    `json:"description"`      // Long description of the event
	Severity        string    `json:"severity"`         // Severity of the event
	Urgency         string    `json:"urgency"`          // Urgency of the event
	AreaDescription string    `json:"area_description"` // Affected Area description
	Sender          string    `json:"sender"`           // Sender (email) of the event
	SenderName      string    `json:"sendername"`       // Sender name of the event
	Start           time.Time `json:"start"`            // Event start time
	End             time.Time `json:"end"`              // Event end time
}

// GetWeatherReport gets the weather report
func (s Service) GetWeatherAlerts(rw http.ResponseWriter, req *http.Request) {

	txn := newrelic.FromContext(req.Context())
	defer txn.StartSegment("Alerts GetWeatherAlerts").End()

	//	Our return value
	retval := AlertReport{}
	retval.Alerts = []AlertItem{} // Initialize the array

	//	Parse the request
	request := AlertsRequest{}
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		zlog.Errorw(
			"Problem decoding request",
			"error", err,
		)
		sendErrorResponse(rw, err, http.StatusBadRequest)
		return
	}

	//	Add the request
	txn.AddAttribute("request", request)

	//	First, call the points service for the lat/long specified
	pointsUrl := fmt.Sprintf("https://api.weather.gov/points/%s,%s", request.Latitude, request.Longitude)
	clientRequest, err := http.NewRequest("GET", pointsUrl, nil)
	if err != nil {
		zlog.Errorw(
			"problem creating request to the NWS points service",
			"error", err,
		)
		txn.NoticeError(err)
		return
	}

	//	Set our headers
	clientRequest.Header.Set("Content-Type", "application/geo+json; charset=UTF-8")
	clientRequest = newrelic.RequestWithTransactionContext(clientRequest, txn)

	//	Execute the request
	client := &http.Client{}
	client.Transport = newrelic.NewRoundTripper(client.Transport)
	pointClientResponse, err := ctxhttp.Do(req.Context(), client, clientRequest)
	if err != nil {
		zlog.Errorw(
			"error when sending request to the NWS points service",
			"error", err,
		)
		txn.NoticeError(err)
		return
	}
	defer pointClientResponse.Body.Close()

	//	Decode the response:
	pointsResponse := NWSPointsResponse{}
	err = json.NewDecoder(pointClientResponse.Body).Decode(&pointsResponse)
	if err != nil {
		zlog.Errorw(
			"problem decoding the response from the NWS points service",
			"error", err,
		)
		txn.NoticeError(err)
		return
	}

	//	Parse the zone information and add information to the returned report
	//	TODO: Update to use county
	retval.Longitude = pointsResponse.Geometry.Coordinates[0]
	retval.Latitude = pointsResponse.Geometry.Coordinates[1]
	retval.NWSCounty = pointsResponse.Properties.County
	countyCode := strings.Replace(retval.NWSCounty, "https://api.weather.gov/zones/county/", "", -1)
	retval.ActiveAlertsForCountyURL = fmt.Sprintf("https://alerts.weather.gov/cap/wwaatmget.php?x=%s&y=1", countyCode)
	retval.State = pointsResponse.Properties.RelativeLocation.Properties.State
	retval.City = pointsResponse.Properties.RelativeLocation.Properties.City

	//	Call the alerts service for the lat/long specified
	alertsServiceUrl := fmt.Sprintf("https://api.weather.gov/alerts?point=%s,%s", request.Latitude, request.Longitude)
	alertClientRequest, err := http.NewRequest("GET", alertsServiceUrl, nil)
	if err != nil {
		zlog.Errorw(
			"problem creating request to the NWS alerts service",
			"error", err,
		)
		txn.NoticeError(err)
		return
	}

	//	Set our headers
	alertClientRequest.Header.Set("Content-Type", "application/geo+json; charset=UTF-8")
	alertClientRequest = newrelic.RequestWithTransactionContext(alertClientRequest, txn)

	//	Execute the request
	alertClient := &http.Client{}
	alertClient.Transport = newrelic.NewRoundTripper(alertClient.Transport)
	alertClientResponse, err := ctxhttp.Do(req.Context(), alertClient, alertClientRequest)
	if err != nil {
		zlog.Errorw(
			"error when sending request to the NWS alerts service",
			"error", err,
		)
		txn.NoticeError(err)
		return
	}
	defer alertClientResponse.Body.Close()

	//	Decode the response:
	alertsResponse := NWSAlertsResponse{}
	err = json.NewDecoder(alertClientResponse.Body).Decode(&alertsResponse)
	if err != nil {
		zlog.Errorw(
			"problem decoding the response from the NWS alerts service",
			"error", err,
		)
		txn.NoticeError(err)
		return
	}

	//	Compile our report
	for _, item := range alertsResponse.Features {

		alertItem := AlertItem{
			Event:           item.Properties.Event,
			Headline:        item.Properties.Headline,
			Description:     item.Properties.Description,
			Severity:        item.Properties.Severity,
			Urgency:         item.Properties.Urgency,
			AreaDescription: item.Properties.AreaDesc,
			Sender:          item.Properties.Sender,
			SenderName:      item.Properties.SenderName,
			Start:           item.Properties.Effective,
			End:             item.Properties.Ends,
		}

		retval.Alerts = append(retval.Alerts, alertItem)
	}

	//	Add the report to the request metadata
	txn.AddAttribute("response", retval)

	//	Serialize to JSON & return the response:
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(rw).Encode(retval)
}
