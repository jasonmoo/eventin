package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
)

type (
	Response struct {
		Status     string   `json:"status"`
		Errors     []string `json:"errors"`
		NumResults int      `json:"num_results"`
		Results    []*Event `json:"results"`
		Copyright  string   `json:"copyright"`
	}

	EventCache struct {
		mu        sync.RWMutex
		Copyright string
		Events    []*Event
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

		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`

		JSON []byte `json:"-"`
	}
)

const (
	Endpoint = "http://api.nytimes.com/svc/events/v2/listings.json"
	MaxRange = (1 << 31) - 1
)

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

func (ec *EventCache) WriteResponse(w http.ResponseWriter, swlat, swlng, nelat, nelng float64) {

	ec.mu.RLock()
	defer ec.mu.RUnlock()

	buf := bufio.NewWriter(w)
	fmt.Fprintf(buf, `{"copyright":%q,"results":[`, ec.Copyright)

	var written bool
	for _, event := range ec.Events {
		if event.Lat > swlat && event.Lat < nelat && event.Lng > swlng && event.Lng < nelng {
			if written {
				buf.WriteByte(',')
			}
			buf.Write(event.JSON)
			written = true
		}
	}
	buf.WriteString("]}")
	buf.Flush()

}

func LoadCache(cachefile string) (*EventCache, error) {

	file, err := os.Open(cachefile)
	if err != nil {
		return &EventCache{Events: make([]*Event, 0)}, nil
	}
	defer file.Close()

	ec := &EventCache{}
	err = json.NewDecoder(file).Decode(ec)
	if err != nil {
		return nil, err
	}

	for i, event := range ec.Events {
		ec.Events[i].JSON, _ = json.Marshal(event)
	}

	return ec, nil

}

func (ec *EventCache) RefreshData() error {

	events := make([]*Event, 0)

	// initial search with values and num count
	r, err, _ := fetch(0)
	if err != nil {
		return err
	}

	for _, event := range r.Results {
		if event.GeocodeLatitude == "" || event.GeocodeLongitude == "" {
			continue
		}
		events = append(events, event)
	}

	for offset := 0; offset < r.NumResults; offset += 100 {
		result, err, _ := fetch(offset)
		if err != nil {
			return err
		}
		for _, event := range result.Results {
			if event.GeocodeLatitude == "" || event.GeocodeLongitude == "" {
				continue
			}
			events = append(events, event)
		}
		time.Sleep(time.Second) // rate limit ouseves
	}

	if *dev {
		log.Println("finished caching")
	}

	for i, _ := range events {

		// convert lat,lng for easy comparison
		events[i].Lat, _ = strconv.ParseFloat(events[i].GeocodeLatitude, 64)
		events[i].Lng, _ = strconv.ParseFloat(events[i].GeocodeLongitude, 64)

		// cleanup day names
		for j, day := range events[i].RecurDays {
			events[i].RecurDays[j] = DayTranslations[day]
		}

		address := url.QueryEscape(events[i].StreetAddress + ", " + events[i].Borough + " " + events[i].State)
		events[i].MapURL = fmt.Sprintf("https://maps.google.com/maps?saddr=Current+Location&daddr=%s", address)

		// populate html
		buf := new(bytes.Buffer)
		err = EventTemplate.Execute(buf, events[i])
		if err != nil {
			events[i].HTML = err.Error()
		} else {
			events[i].HTML = buf.String()
		}

		events[i].JSON, _ = json.Marshal(events[i])

	}

	ec.mu.Lock()
	ec.Copyright = r.Copyright
	ec.Events = events
	ec.mu.Unlock()

	return nil

}

func (ec *EventCache) WriteCache(cachefile string) error {

	ec.mu.RLock()
	defer ec.mu.RUnlock()

	data, err := json.Marshal(ec)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cachefile, data, 0666)

}

func fetch(offset int) (*Response, error, time.Duration) {
	// central park:
	// 40.767927, -73.980047
	search := url.Values{
		"api-key": {ApiKey},
		"ll":      {"40.767927,-73.980047"},
		"radius":  {strconv.Itoa(MaxRange)}, // meters
		"sort":    {"geocode_latitude asc"},
		"limit":   {"100"},
		"offset":  {strconv.Itoa(offset)},
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

	return r, nil, total_time

}
