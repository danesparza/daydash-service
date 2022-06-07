package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/apognu/gocal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"golang.org/x/net/context/ctxhttp"
)

type CalendarRequest struct {
	CalendarURL string `json:"url"`
	Timezone    string `json:"timezone"`
}

type CalendarResponse struct {
	TimeZone         string          `json:"timezone"`         // The timezone used
	CurrentLocalTime time.Time       `json:"currentlocaltime"` // Sanity check:  Current local time in the timezone given
	Events           []CalendarEvent `json:"events"`           // The calendar events found
	Version          string          `json:"version"`          // Service version
}

type CalendarEvent struct {
	UID         string    `json:"uid"`         // Unique event id
	Summary     string    `json:"summary"`     // Event summary
	Description string    `json:"description"` // Event long description
	StartTime   time.Time `json:"starttime"`   // Event start time
	EndTime     time.Time `json:"endtime"`     // Event end time
}

// GetTodaysEvents gets today's events from the given ical calendar url and the timezone.
func (s Service) GetCalendar(rw http.ResponseWriter, req *http.Request) {

	txn := newrelic.FromContext(req.Context())
	defer txn.StartSegment("Calendar GetCalendar").End()

	//	Our return value
	retval := CalendarResponse{}
	retval.Events = []CalendarEvent{} // Initialize the array

	//	Parse the request
	request := CalendarRequest{}
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

	//	Set the timezone in the response
	timezone := request.Timezone
	url := request.CalendarURL
	retval.TimeZone = timezone

	//	Set our start / end times
	location, err := time.LoadLocation(timezone)
	if err != nil {
		zlog.Errorw(
			"Error setting location from the timezone - most likely timezone data not loaded in host OS",
			"timezone", timezone,
			"error", err,
		)
		txn.NoticeError(err)
	}

	//	Current time in the location
	t := time.Now().In(location)

	//	Find the beginning and end of the day in the given timezone
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, location)
	end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, location)

	retval.CurrentLocalTime = t

	zlog.Infow(
		"Time debugging",
		"currentLocalTime", t,
		"start", start,
		"end", end,
		"timezone", timezone,
		"url", url,
	)

	//	First, get the ical calendar at the url given
	clientRequest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		zlog.Errorw(
			"problem creating request to the ical url given",
			"err", err,
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
	calendarDataResponse, err := ctxhttp.Do(req.Context(), client, clientRequest)
	if err != nil {
		zlog.Errorw(
			"error when sending request to get the calendar data from the url",
			"err", err,
		)
		txn.NoticeError(err)
		return
	}
	defer calendarDataResponse.Body.Close()

	//	Create a parser and use our start/end times
	calEvents, err := GetEventsForDay(calendarDataResponse.Body, start, end, location)
	if err != nil {
		zlog.Errorw(
			"problem getting events for day",
			"err", err,
			"start", start,
			"end", end,
			"location", location,
		)
		txn.NoticeError(err)
		return
	}

	//	Track our event IDs
	eventIDs := make(map[string]struct{})

	//	Google calendar is so fucking weird.
	//	Regular events (even if they recur) are returned as UTC start/end times.  They can be converted to local time easily.  This is good.
	//	All-day events are going to be returned without a timezone.  A parser will assume a time of midnight IN THE UTC TIMEZONE.  This will lead to bullshit weirdness.
	for _, e := range calEvents {

		//	If we have duplicate event ids (and we shouldn't ... but I've seen this in the wild) discard anything after the first instance
		//	See https://stackoverflow.com/a/10486196/19020 for more info on using an empty struct in a map to track this
		if _, containsEvent := eventIDs[e.Uid]; containsEvent {
			continue
		} else {
			eventIDs[e.Uid] = struct{}{}
		}

		diff := e.End.Sub(*e.Start)

		//	If it looks like it's an all-day event, and the url includes 'calendar.google.com', then don't use the timezone
		if diff.Hours() > 23 && strings.Contains(url, "calendar.google.com") {

			zlog.Infow(
				"Google all-day event detected.  Using the UTC start/end times and rewriting them as local",
				"url", url,
				"summary", e.Summary,
				"description", e.Description,
				"starttime", e.Start.UTC(),
				"endtime", e.End.UTC(),
				"rewritten-starttime", RewriteToLocal(e.Start.UTC(), location),
				"rewritten-endtime", RewriteToLocal(e.End.UTC(), location),
			)

			calEvent := CalendarEvent{
				UID:         e.Uid,
				Summary:     e.Summary,
				Description: e.Description,
				StartTime:   RewriteToLocal(e.Start.UTC(), location), // Rewrite the UTC time to appear as local time
				EndTime:     RewriteToLocal(e.End.UTC(), location),   // Rewrite the UTC time to appear as local time
			}

			retval.Events = append(retval.Events, calEvent)
		} else {
			calEvent := CalendarEvent{
				UID:         e.Uid,
				Summary:     e.Summary,
				Description: e.Description,
				StartTime:   e.Start.In(location),
				EndTime:     e.End.In(location),
			}

			retval.Events = append(retval.Events, calEvent)
		}
	}

	//	Serialize to JSON & return the response:
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(rw).Encode(retval)

}

// RewriteToLocal - rewrites a given time to use the passed location data
func RewriteToLocal(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}

// GetEventsForDay gets events for the day given the calendar body and start/end times
func GetEventsForDay(calendarBody io.Reader, start, end time.Time, location *time.Location) ([]gocal.Event, error) {

	//	Our return value:
	retval := []gocal.Event{}

	//	Create a parser and use our start/end times
	c := gocal.NewParser(calendarBody)
	c.Start, c.End = &start, &end
	c.AllDayEventsTZ = location // Also give it a location

	err := c.Parse()
	if err != nil {
		return retval, fmt.Errorf("problem parsing calendar file: %v", err)
	}

	return c.Events, nil
}
