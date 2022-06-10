package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// PollenReport represents the report of pollen data
type PollenReport struct {
	Location          string    `json:"location"`           // The location for the report
	Zipcode           string    `json:"zip"`                // The zipcode for the report
	PredominantPollen string    `json:"predominant_pollen"` // The predominant pollen in the report period
	StartDate         time.Time `json:"startdate"`          // The start time for this report
	Data              []float64 `json:"data"`               //	Pollen data indices -- one for today and each future day
	ReportingService  string    `json:"service"`            // The reporting service
}

type PollenRequest struct {
	Zipcode string `json:"zipcode"`
}

// PollenService is the interface for all services that can fetch pollen data
type PollenService interface {
	// GetPollenReport gets the pollen report
	GetPollenReport(ctx context.Context, zipcode string) (PollenReport, error)
}

// GetPollenReport godoc
// @Summary Gets pollen data and forecast for a given location
// @Description Gets pollen data and forecast for a given location
// @Tags dashboard
// @Accept  json
// @Produce  json
// @Param config body api.PollenRequest true "The location to get data for"
// @Success 200 {object} api.PollenReport
// @Failure 400 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /pollen [post]
func (s Service) GetPollenReport(rw http.ResponseWriter, req *http.Request) {

	ch := make(chan PollenReport, 1)

	txn := newrelic.FromContext(req.Context())
	defer txn.StartSegment("Pollen GetPollenReport").End()

	//	Parse the request
	request := PollenRequest{}
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
	if request.Zipcode == "" {
		sendErrorResponse(rw, fmt.Errorf("you must include a valid zipcode param"), http.StatusBadRequest)
		return
	}

	//	Set the services to call with
	services := []PollenService{
		NasacortService{},
		PollencomService{},
	}

	//	For each passed service ...
	for _, service := range services {

		//	Launch a goroutine for each service...
		go func(c context.Context, s PollenService, zip string) {

			//	Get its pollen report ...
			result, err := s.GetPollenReport(c, zip)

			//	As long as we don't have an error, return what we found on the result channel
			if err == nil {

				//	Make sure we also have more than one datapoint!
				if len(result.Data) > 1 {
					select {
					case ch <- result:
					default:
					}
				}
			}
		}(req.Context(), service, request.Zipcode)

	}

	//	Capture the first result passed on the channel
	retval := <-ch

	//	Serialize to JSON & return the response:
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(rw).Encode(retval)
}
