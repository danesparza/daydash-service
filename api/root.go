package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/danesparza/daydash-service/internal/logger"
	"github.com/danesparza/daydash-service/version"
	"go.uber.org/zap"
)

var (
	rootlogger *zap.Logger
	zlog       *zap.SugaredLogger
)

// Service encapsulates API service operations
type Service struct {
	StartTime time.Time
}

// SystemResponse is a response for a system request
type SystemResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ErrorResponse represents an API response
type ErrorResponse struct {
	Message string `json:"message"`
}

// sendErrorResponse is used to send back an error:
func sendErrorResponse(rw http.ResponseWriter, err error, code int) {
	//	Our return value
	response := ErrorResponse{Message: "Error: " + err.Error()}

	//	Serialize to JSON & return the response:
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(code)
	json.NewEncoder(rw).Encode(response)
}

// ShowUI redirects to the '/ui/' virtual directory
func ShowUI(rw http.ResponseWriter, req *http.Request) {
	http.Redirect(rw, req, "/ui/", http.StatusMovedPermanently)
}

func init() {
	rootlogger, _ = logger.NewProd()
	defer rootlogger.Sync() // flushes buffer, if any
	zlog = rootlogger.Sugar()
}

// ApiVersionMiddleware adds the API version informaiton to the response header
func ApiVersionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		//	Include the version in the response headers:
		rw.Header().Set(version.Header, version.String())

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, r)
	})
}
