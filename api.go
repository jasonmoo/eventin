package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type (
	Response struct {
		Status     string   `json:"status"`
		Errors     []string `json:"errors"`
		NumResults int      `json:"num_results"`
		Results    []Event  `json:"results"`
		Copyright  string   `json:"copyright"`
	}

	Event struct {
		Borough             string    `json:"borough"`
		Category            string    `json:"category"`
		City                string    `json:"city"`
		CriticName          string    `json:"critic_name"`
		CrossStreet         string    `json:"cross_street"`
		DateTimeDescription string    `json:"date_time_description"`
		EventDetailUrl      string    `json:"event_detail_url"`
		EventId             int       `json:"event_id"`
		EventName           string    `json:"event_name"`
		EventScheduleId     int       `json:"event_schedule_id"`
		Festival            bool      `json:"festival"`
		Free                bool      `json:"free"`
		GeocodeLatitude     string    `json:"geocode_latitude"`
		GeocodeLongitude    string    `json:"geocode_longitude"`
		KidFriendly         bool      `json:"kid_friendly"`
		LastChance          bool      `json:"last_chance"`
		LastModified        time.Time `json:"last_modified"` // 2014-07-08T06:04:45.188Z
		LongRunningShow     bool      `json:"long_running_show"`
		Neighborhood        string    `json:"neighborhood"`
		PreviewsAndOpenings bool      `json:"previews_and_openings"`
		RecurDays           []string  `json:"recur_days"`
		RecurringStartDate  time.Time `json:"recurring_start_date"` // 2014-02-26T05:00:00.56Z
		State               string    `json:"state"`
		StreetAddress       string    `json:"street_address"`
		Subcategory         string    `json:"subcategory"`
		Telephone           string    `json:"telephone"`
		TimesPick           bool      `json:"times_pick"`
		VenueDetailUrl      string    `json:"venue_detail_url"`
		VenueName           string    `json:"venue_name"`
		VenueWebsite        string    `json:"venue_website"`
		WebDescription      string    `json:"web_description"`

		HTML   string `json:"html"`
		MapURL string `json:"map_url"`
	}
)

const Endpoint = "http://api.nytimes.com/svc/events/v2/listings.json"

var (
	DayTranslations = map[string]string{
		"mon": "Monday",
		"tue": "Tuesday",
		"wed": "Wednesday",
		"thu": "Thursday",
		"fri": "Friday",
		"sat": "Saturday",
		"sun": "Sunday",
	}
	ApiKey = os.Getenv("NYT_API_KEY")
)

func Fetch(lat, lng string) (*Response, error, time.Duration) {

	search := url.Values{
		"api-key": {ApiKey},
		"ll":      {lat + "," + lng},
		"radius":  {"1000"}, // meters
		"sort":    {"dist asc"},
		"limit":   {"100"},
	}

	apiurl := Endpoint + "?" + search.Encode()

	if *dev {
		log.Println("GET", apiurl)
	}

	start := time.Now()
	resp, err := http.Get(apiurl)
	total_time := time.Since(start)

	if *dev {
		log.Println(resp.Status, total_time, resp)
	}

	if err != nil {
		return nil, err, total_time
	}
	defer resp.Body.Close()

	r := new(Response)

	err = json.NewDecoder(resp.Body).Decode(r)
	if err != nil {
		return nil, err, total_time
	}

	for i, event := range r.Results {

		// cleanup day names
		for j, day := range r.Results[i].RecurDays {
			r.Results[i].RecurDays[j] = DayTranslations[day]
		}

		address := url.QueryEscape(event.StreetAddress + ", " + event.Borough + " " + event.State)
		r.Results[i].MapURL = fmt.Sprintf("https://maps.google.com/maps?saddr=Current+Location&daddr=%s", address)

		// populate html
		buf := new(bytes.Buffer)
		err := EventTemplate.Execute(buf, r.Results[i])
		if err != nil {
			r.Results[i].HTML = err.Error()
		} else {
			r.Results[i].HTML = buf.String()
		}

	}

	return r, nil, total_time
}
