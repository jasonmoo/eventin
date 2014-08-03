package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
)

var (
	host   = flag.String("host", ":8080", "host name to listen on")
	webdir = flag.String("webdir", "./web", "source of static files")

	dev = flag.Bool("dev", false, "output dev signals")
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {

	fmt.Println("eventing starting up")

	flag.Parse()

	http.Handle("/", NewGzipFileHandler(*webdir, []string{}))
	http.Handle("/search", NewGzipHandler(EventSearchHandler))
	log.Fatal(http.ListenAndServe(*host, nil))

}

func EventSearchHandler(w http.ResponseWriter, req *http.Request) {

	w.Header().Set("Content-Type", "application/json;charset=utf-8")

	var (
		lat = strings.TrimSpace(req.FormValue("lat"))
		lng = strings.TrimSpace(req.FormValue("lng"))
	)

	if len(lat) == 0 || len(lng) == 0 {
		http.Error(w, "lat and lng required", http.StatusBadRequest)
		return
	}

	r, err, total_time := Fetch(lat, lng)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("X-NYTAPI-Response-Time", total_time.String())

	json.NewEncoder(w).Encode(r)

}
