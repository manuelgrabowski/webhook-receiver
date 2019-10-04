package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/go-chi/chi"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	_ "github.com/lib/pq"
	"github.com/peterbourgon/ff"
)

type maxBytesHandler struct {
	h http.Handler
	n int64
}

func (h *maxBytesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.n)
	h.h.ServeHTTP(w, r)
}

func main() {
	fs := flag.NewFlagSet("wr", flag.ExitOnError)
	var (
		environment = fs.String("environment", "develop", "the environment we are running in")
		port        = fs.String("port", "8080", "the port archivepipe is running on")
	)

	ff.Parse(fs, os.Args[1:],
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
		ff.WithEnvVarPrefix("WR"),
	)

	// Heroku doesn't support EnvVarPrefixes so we have to overwrite this
	if os.Getenv("PORT") != "" {
		*port = os.Getenv("PORT")
	}

	// var t = &http.Transport{
	// 	Dial: (&net.Dialer{
	// 		Timeout: 5 * time.Second,
	// 	}).Dial,
	// 	TLSHandshakeTimeout: 5 * time.Second,
	// }
	// var c = &http.Client{
	// 	Timeout:   time.Second * 10,
	// 	Transport: t,
	// }

	l := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	switch strings.ToLower(*environment) {
	case "development":
		l = level.NewFilter(l, level.AllowInfo())
	case "prod":
		l = level.NewFilter(l, level.AllowError())
	}
	l = log.With(l, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	// Set up HTTP API
	r := chi.NewRouter()

	// Register Prometheus metrics
	r.Handle("/metrics", promhttp.Handler())

	level.Info(l).Log("msg", fmt.Sprintf("archivepipe api is running on :%s", *port), "environment", *environment)

	// Set up webserver and and set max file limit to 50MB
	err := http.ListenAndServe(fmt.Sprintf(":%s", *port), &maxBytesHandler{h: r, n: (50 * 1024 * 1024)})
	if err != nil {
		level.Error(l).Log("err", err)
		return
	}
}
