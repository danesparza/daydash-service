package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image/color"
	"image/jpeg"
	"net/http"

	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type MapImageRequest struct {
	Lat  float64 `json:"lat"`
	Long float64 `json:"long"`
	Zoom int     `json:"zoom"`
}

type MapImageResponse struct {
	Lat   float64 `json:"lat"`
	Long  float64 `json:"long"`
	Zoom  int     `json:"zoom"`
	Image string  `json:"image"` // The map image (in base64 encoded data uri format)
}

// GetMapImageForCoordinates godoc
// @Summary Gets a map image for the given lat, long and zoom level
// @Description Gets a map image for the given lat, long and zoom level. Returns the map image as a base64 encoded jpeg
// @Tags dashboard
// @Accept  json
// @Produce  json
// @Param config body api.MapImageRequest true "The calendar data to fetch"
// @Success 200 {object} api.MapImageResponse
// @Failure 400 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /mapimage [post]
func (s Service) GetMapImageForCoordinates(rw http.ResponseWriter, req *http.Request) {

	txn := newrelic.FromContext(req.Context())
	defer txn.StartSegment("Mapimage GetMapImageForCoordinates").End()

	//	Parse the request
	request := MapImageRequest{}
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

	//	Set a default zoom
	zoom := request.Zoom
	if zoom == 0 {
		zoom = 3
	}

	lat := request.Lat
	long := request.Long

	retval := MapImageResponse{
		Lat:  lat,
		Long: long,
		Zoom: zoom,
	}

	//	Get the map image
	ctx := sm.NewContext()
	ctx.SetSize(600, 360)
	ctx.SetZoom(zoom)
	ctx.SetTileProvider(sm.NewTileProviderWikimedia())
	ctx.AddObject(
		sm.NewMarker(
			s2.LatLngFromDegrees(lat, long),
			color.RGBA{0xff, 0, 0, 0xff},
			16.0,
		),
	)

	img, err := ctx.Render()
	if err != nil {
		zlog.Errorw(
			"error rendering map image",
			"lat", lat,
			"long", long,
			"zoom", zoom,
		)
		txn.NoticeError(err)
		return
	}

	//	Encode to jpg
	buffer := new(bytes.Buffer)
	jpeg.Encode(buffer, img, nil)

	//	Encode the jpeg to base64
	retval.Image = base64.StdEncoding.EncodeToString(buffer.Bytes())

	//	Serialize to JSON & return the response:
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(rw).Encode(retval)
}
