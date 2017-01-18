package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/context"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Wrapper func(http.HandlerFunc) http.HandlerFunc

type ArgumentList []string

func (l ArgumentList) String() string {
	return strings.Join(l, ",")
}

func (l *ArgumentList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

var (
	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	hosts      = ArgumentList{"localhost:8080", "localhost:8080"}
	port       = 8080
	workLoad   = 1 // Time in ms we'll be spending when hops reaches 0
)

var (
	httpClient = &http.Client{
		Timeout: 0 * time.Second,
	}
)

func handleIt(w http.ResponseWriter, req *http.Request) {
	hops, _ := strconv.Atoi(req.URL.Query().Get("hops"))

	if hops > 1000 {
		w.Write([]byte("Hops setting too high, supporting a max of 1000."))
		return
	}

	if hops > 0 {

		var host string
		if len(hosts) == 1 {
			host = hosts[0]
		} else {
			host = hosts[seededRand.Intn(len(hosts))]
		}

		hops = hops - 1

		fmt.Printf("Making request to %s, remaining hops after this request: %d\n", host, hops)
		resp, err := httpClient.Get(fmt.Sprintf("http://%s?hops=%d", host, hops))

		if err != nil {
			panic(err)
		}

		defer resp.Body.Close()

		reply, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		if workLoad > 0 {
			time.Sleep(time.Millisecond * time.Duration(workLoad))
		}

		w.Write(reply)

		return
	}

	w.Write([]byte("Done"))
}

func init() {
	flag.IntVar(&port, "listen-on", 8080, "The port to listen on for this instance")
	flag.IntVar(&workLoad, "work-load", 0, "The amount of time (in ms) we'll spend when we're the last service")
	flag.Var(&hosts, "add-host", "Add a host to the pool of hosts (hostname or ip:port, e.g.: localhost:8080) to hop to")
}

func main() {

	flag.Parse()

	mux := http.DefaultServeMux

	handler := decorateHandler(handleIt,
		requestLogger(log.New(os.Stdout, "", 0)),
		addRequestID(),
	)

	mux.HandleFunc("/", handler)

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		Handler:        mux,
	}

	fmt.Println(s.ListenAndServe())
}

// requestLogger is a wrapper that adds a before/after log statement to any request
func requestLogger(l *log.Logger) Wrapper {
	return func(fn http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			rid, ok := req.Context().Value("rid").(string)
			if !ok {
				rid = "none"
			}

			l.Printf("[%s] Before", rid)

			start := time.Now()
			fn(w, req)
			l.Printf("[%s] Finished in %s", rid, time.Since(start))
		}
	}
}

// addRequestID Is a wrapper that defines the Request ID in both the header and the ctx
func addRequestID() Wrapper {
	return func(fn http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			var rid string = req.Header.Get("Request-ID")
			if rid == "" || len(rid) > 8 {
				rid = randStringBytesMaskSrc(8)
			}

			req = req.WithContext(context.WithValue(req.Context(), "rid", rid))
			req.Header.Add("Request-ID", rid)

			fn(w, req)
		}
	}
}

// decorateHandler decorates the handlerFunc with all the specified Wrappers
func decorateHandler(h http.HandlerFunc, ds ...Wrapper) http.HandlerFunc {

	for _, decorate := range ds {
		h = decorate(h)
	}

	return h
}

// randStringBytesMaskSrc generates a random string of N characters
func randStringBytesMaskSrc(n int) string {
	const (
		letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)

	var src = rand.NewSource(time.Now().UnixNano())
	var b = make([]byte, n)

	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
