// Command flightpathd is a microservice that performs
// flight path calculatons.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dwlnetnl/flighttracker-assignment/flightpath"
)

func main() {
	var args struct {
		addr    string
		readto  time.Duration
		writeto time.Duration
	}
	flag.StringVar(&args.addr, "addr", ":8080", "Address to listen on.")
	// set some (sane) timeouts so it's safe to run in production
	// see https://simon-frey.com/blog/go-as-in-golang-standard-net-http-config-will-break-your-production/
	flag.DurationVar(&args.readto, "readto", 5*time.Second, "Request read timeout.")
	flag.DurationVar(&args.writeto, "writeto", 10*time.Second, "Request write timeout.")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/calculate", calculate)

	srv := &http.Server{
		Addr:         args.addr,
		Handler:      mux,
		ReadTimeout:  args.readto,
		WriteTimeout: args.writeto,
	}
	go func() {
		log.Printf("Listening at %s\n", args.addr)
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatalln(err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	<-sigs

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	log.Println("Signal received, shutting down now...")
	srv.Shutdown(ctx)
}

func calculate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		const code = http.StatusMethodNotAllowed
		http.Error(w, http.StatusText(code), code)
		reqlogPrintln(r, code, r.Method)
		return
	}

	input := http.MaxBytesReader(w, r.Body, 4096)
	data, err := io.ReadAll(input)
	if err != nil {
		const code = http.StatusBadRequest
		http.Error(w, http.StatusText(code), code)
		reqlogPrintln(r, code, "unmarshal:", err)
		return
	}
	flights, err := unmarshal(data)
	if err != nil {
		const code = http.StatusBadRequest
		http.Error(w, http.StatusText(code), code)
		reqlogPrintln(r, code, "unmarshal:", err, fmt.Sprintf("(%q)", data))
		return
	}

	route := flightpath.Calculate(flights)

	err = marshal(w, route)
	if err != nil {
		const code = http.StatusInternalServerError
		http.Error(w, http.StatusText(code), code)
		reqlogPrintln(r, code, "marshal:", err)
	} else {
		reqlogPrintln(r, http.StatusOK, flights, "->", route)
	}
}

func reqlogPrintln(r *http.Request, status int, args ...any) {
	args = append([]any{
		r.RemoteAddr,
		r.RequestURI,
		status,
	}, args...)
	log.Println(args...)
}

func unmarshal(data []byte) ([]*flightpath.Flight, error) {
	return unmarshalJSON(data)
}

func unmarshalJSON(data []byte) ([]*flightpath.Flight, error) {
	var in [][2]string
	err := json.Unmarshal(data, &in)
	if err != nil {
		return nil, err
	}

	out := make([]*flightpath.Flight, len(in))
	for i, f := range in {
		if f[0] == "" || f[1] == "" {
			return nil, fmt.Errorf("invalid flight: [%s, %s]", f[0], f[1])
		}
		out[i] = &flightpath.Flight{From: f[0], To: f[1]}
	}

	return out, nil
}

var flightRegexp = regexp.MustCompile(`\["(\S{3})", *"(\S{3})"\]`)

func unmarshalRegexp(data []byte) ([]*flightpath.Flight, error) {
	if len(data) == 0 {
		// return empty allocated slice to indicate no results
		// could return nil slice but may be confusing to callers
		return []*flightpath.Flight{}, nil
	}

	last := len(data) - 1
	if data[0] != '[' || data[last] != ']' {
		return nil, errors.New("JSON array expected")
	}

	payload := string(data[1:last])
	matches := flightRegexp.FindAllStringSubmatch(payload, -1)
	out := make([]*flightpath.Flight, len(matches))

	for i, match := range matches {
		out[i] = &flightpath.Flight{From: match[1], To: match[2]}
	}

	return out, nil
}

func unmarshalCustom(data []byte) (out []*flightpath.Flight, err error) {
	if len(data) == 0 {
		// return empty allocated slice to indicate no results
		// could return nil slice but may be confusing to callers
		return []*flightpath.Flight{}, nil
	}

	end := len(data) - 1
	if data[0] != '[' || data[end] != ']' {
		return nil, errors.New("JSON array expected")
	}

	s := string(data[1 : len(data)-1])
	nquote := strings.Count(s, `"`)
	const quotesPerElem = 4
	if nquote%2 != 0 {
		return nil, errors.New("invalid syntax, missing string quote")
	}

	elem := 1
	nelem := nquote / quotesPerElem
	out = make([]*flightpath.Flight, 0, nelem)
	for i := 0; i < len(s); {
		if s[i] != '[' {
			return nil, fmt.Errorf("JSON array expected at element %d", elem)
		}
		i++

		if s[i] != '"' {
			return nil, fmt.Errorf("JSON string expected at element %d", elem)
		}
		i++
		j := i
		for s[i] != '"' {
			i++
		}

		f := &flightpath.Flight{From: s[j:i]}
		i++

		if s[i] != ',' {
			return nil, fmt.Errorf("JSON array of 2 expected at element %d", elem)
		}
		i++
		for s[i] == ' ' {
			i++
		}

		if s[i] != '"' {
			return nil, fmt.Errorf("JSON string expected at element %d", elem)
		}
		i++
		j = i
		for s[i] != '"' {
			i++
		}
		f.To = s[j:i]
		i++

		if s[i] != ']' {
			return nil, fmt.Errorf("JSON array expected at element %d", elem)
		}
		i++

		if i == len(s) {
			out = append(out, f)
			break
		}

		if s[i] != ',' {
			return nil, fmt.Errorf("invalid syntax after element %d", elem)
		}
		i++
		for s[i] == ' ' {
			i++
		}

		out = append(out, f)
		elem++
	}

	return out, nil
}

func marshal(w io.Writer, flight *flightpath.Flight) error {
	return marshalJSON(w, flight)
}

func marshalJSON(w io.Writer, flight *flightpath.Flight) error {
	payload := [2]string{flight.From, flight.To}
	return json.NewEncoder(w).Encode(payload)
}

func marshalFmt(w io.Writer, flight *flightpath.Flight) error {
	_, err := fmt.Fprintf(w, "[%q,%q]\n", flight.From, flight.To)
	return err
}

func marshalAppend(w io.Writer, flight *flightpath.Flight) error {
	b := make([]byte, 0, 16)
	b = append(b, '[')
	b = strconv.AppendQuote(b, flight.From)
	b = append(b, ',')
	b = strconv.AppendQuote(b, flight.To)
	b = append(b, ']')
	b = append(b, '\n')
	_, err := w.Write(b)
	return err
}
