package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"golang.org/x/net/context/ctxhttp"
)

// PollencomService is a pollen service for Pollen.com formatted data
type PollencomService struct{}

// PollencomForecastResponse is the native service return format for the extended forecast (includes pollen indices)
type PollencomForecastResponse struct {
	Type         string `json:"Type"`
	ForecastDate string `json:"ForecastDate"`
	Location     struct {
		ZIP     string `json:"ZIP"`
		City    string `json:"City"`
		State   string `json:"State"`
		Periods []struct {
			Period string  `json:"Period"`
			Index  float64 `json:"Index"`
		} `json:"periods"`
		DisplayLocation string `json:"DisplayLocation"`
	} `json:"Location"`
}

// PollencomCurrentResponse is the native service return format for the current conditions (includes predominant pollen)
type PollencomCurrentResponse struct {
	Type         string `json:"Type"`
	ForecastDate string `json:"ForecastDate"`
	Location     struct {
		ZIP     string `json:"ZIP"`
		City    string `json:"City"`
		State   string `json:"State"`
		Periods []struct {
			Triggers []struct {
				LGID      int    `json:"LGID"`
				Name      string `json:"Name"`
				Genus     string `json:"Genus"`
				PlantType string `json:"PlantType"`
			} `json:"Triggers"`
			Period string  `json:"Period"`
			Type   string  `json:"Type"`
			Index  float64 `json:"Index"`
		} `json:"periods"`
		DisplayLocation string `json:"DisplayLocation"`
	} `json:"Location"`
}

// GetPollenReport gets the pollen report
func (s PollencomService) GetPollenReport(ctx context.Context, zipcode string) (PollenReport, error) {

	txn := newrelic.FromContext(ctx)
	segment := txn.StartSegment("Pollen GetPollenReport")
	defer segment.End()

	//	Our return value
	retval := PollenReport{}

	//	Format the extended forecast url (to get the pollen indices):
	apiurl := fmt.Sprintf("https://www.pollen.com/api/forecast/extended/pollen/%s", zipcode)

	req, _ := http.NewRequest("GET", apiurl, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.146 Safari/537.36")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Referer", apiurl)
	req = newrelic.RequestWithTransactionContext(req, txn)

	client := &http.Client{}
	client.Transport = newrelic.NewRoundTripper(client.Transport)
	resp, err := ctxhttp.Do(ctx, client, req)

	if err != nil {
		zlog.Errorw(
			"There was a problem calling Pollen.com extended forecast API",
			"error", err,
		)
		txn.NoticeError(err)
		apperr := fmt.Errorf("problem calling Pollen.com extended forecast API: %s", err)
		return retval, apperr
	}
	defer resp.Body.Close()

	//	If the HTTP status code indicates an error, report it and get out
	if resp.StatusCode >= 400 {
		apperr := fmt.Errorf("error getting information from Pollen.com extended forecast API: %s", resp.Status)
		zlog.Errorw(
			"There was an error getting information from Pollen.com extended forecast API",
			"error", apperr,
		)
		txn.NoticeError(apperr)
		return retval, apperr
	}

	//	Decode the return object
	serviceResponse := PollencomForecastResponse{}
	err = json.NewDecoder(resp.Body).Decode(&serviceResponse)
	if err != nil {
		zlog.Errorw(
			"There was a problem decoding the response from Pollen.com extended forecast API",
			"error", err,
		)
		txn.NoticeError(err)
		apperr := fmt.Errorf("problem decoding the response from Pollen.com extended forecast API: %s", err)
		return retval, apperr
	}

	//	Parse the data items:
	dataitems := []float64{}

	if len(serviceResponse.Location.Periods) > 0 {
		parsedToday := serviceResponse.Location.Periods[0].Index
		dataitems = append(dataitems, parsedToday)

		parsedTomorrow := serviceResponse.Location.Periods[1].Index
		dataitems = append(dataitems, parsedTomorrow)

		parsedAfterTomorrrow := serviceResponse.Location.Periods[2].Index
		dataitems = append(dataitems, parsedAfterTomorrrow)

		parsedDay4 := serviceResponse.Location.Periods[3].Index
		dataitems = append(dataitems, parsedDay4)
	}

	//	Format the current conditions url (to get predominant pollen):
	currentapiurl := fmt.Sprintf("https://www.pollen.com/api/forecast/current/pollen/%s", zipcode)

	currreq, _ := http.NewRequest("GET", currentapiurl, nil)
	currreq.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.146 Safari/537.36")
	currreq.Header.Add("Accept", "application/json")
	currreq.Header.Add("Referer", currentapiurl)
	currreq = newrelic.RequestWithTransactionContext(currreq, txn)

	curclient := &http.Client{}
	curclient.Transport = newrelic.NewRoundTripper(curclient.Transport)
	currresp, err := ctxhttp.Do(ctx, curclient, currreq)

	if err != nil {
		zlog.Errorw(
			"There was a problem calling Pollen.com current forecast API",
			"error", err,
		)
		txn.NoticeError(err)
		apperr := fmt.Errorf("problem calling Pollen.com current forecast API: %s", err)
		return retval, apperr
	}
	defer resp.Body.Close()

	//	If the HTTP status code indicates an error, report it and get out
	if currresp.StatusCode >= 400 {
		apperr := fmt.Errorf("error getting information from Pollen.com current forecast API: %s", resp.Status)
		zlog.Errorw(
			"There was an error getting information from Pollen.com current forecast API",
			"error", apperr,
		)
		txn.NoticeError(apperr)
		return retval, apperr
	}

	//	Decode the return object
	serviceCurrentResponse := PollencomCurrentResponse{}
	err = json.NewDecoder(currresp.Body).Decode(&serviceCurrentResponse)
	if err != nil {
		zlog.Errorw(
			"There was a problem decoding the response from Pollen.com current forecast API",
			"error", err,
		)
		txn.NoticeError(err)
		apperr := fmt.Errorf("problem decoding the response from Pollen.com current forecast API: %s", err)
		return retval, apperr
	}

	//	Build the predominant pollen:
	predomPollens := []string{}
	for _, trigger := range serviceCurrentResponse.Location.Periods[0].Triggers {
		predomPollens = append(predomPollens, trigger.Name)
	}

	predomPollen := strings.Join(predomPollens, ", ")

	//	Set the properties in the return object:
	retval = PollenReport{
		ReportingService:  "Pollen.com",
		PredominantPollen: predomPollen,
		Zipcode:           zipcode,
		Location:          fmt.Sprintf("%s, %s", serviceResponse.Location.City, serviceResponse.Location.State),
		StartDate:         time.Now(),
		Data:              dataitems,
	}

	return retval, nil
}
