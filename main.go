package main

import (
	"flag"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

var (
	host      = flag.String("host", ":8080", "host name to listen on")
	webdir    = flag.String("webdir", "./web", "source of static files")
	cachefile = flag.String("cachefile", "./events.json", "cache of current dataset")

	dev = flag.Bool("dev", false, "output dev signals")

	ec  *EventCache
	err error
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {

	log.Println("eventing starting up")

	flag.Parse()

	ec, err = LoadCache(*cachefile)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			start := time.Now()
			err := ec.RefreshData()
			if err != nil {
				log.Println(err)
				// retry every minute until we're back online
				time.Sleep(time.Minute)
				continue
			}
			ec.WriteCache(*cachefile)
			log.Println("Cache updated in", time.Since(start).String())
			time.Sleep(time.Hour)
		}
	}()

	http.Handle("/", NewGzipFileHandler(*webdir, []string{}))
	http.Handle("/search", NewGzipHandler(EventSearchHandler))
	log.Fatal(http.ListenAndServe(*host, nil))

}

func EventSearchHandler(w http.ResponseWriter, req *http.Request) {

	w.Header().Set("Content-Type", "application/json;charset=utf-8")

	swlat, _ := strconv.ParseFloat(req.FormValue("swlat"), 64)
	swlng, _ := strconv.ParseFloat(req.FormValue("swlng"), 64)
	nelat, _ := strconv.ParseFloat(req.FormValue("nelat"), 64)
	nelng, _ := strconv.ParseFloat(req.FormValue("nelng"), 64)

	if swlat == 0 || swlng == 0 || nelat == 0 || nelng == 0 {
		http.Error(w, "swlat, swlng, nelat and nelng required", http.StatusBadRequest)
		return
	}

	ec.WriteResponse(w, swlat, swlng, nelat, nelng)

}
