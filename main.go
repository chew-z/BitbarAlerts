package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/chew-z/go-swiftbar"

	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
	"github.com/valyala/fasthttp"
)

/*Quotes - ..
 */
type Quotes []Quote

/*Quote - ..
 */
type Quote struct {
	Symbol           string  `json:"_symbol"`
	AskPrice         float64 `json:"_ask_price"`
	BidPrice         float64 `json:"_bid_price"`
	RefBidPrice      float64 `json:"_ref_bid_price"`
	HighBidPrice     float64 `json:"_high_bid_price"`
	LowBidPrice      float64 `json:"_low_bid_price"`
	BidDayChange     float64 `json:"_bid_day_change"`
	BidDayChangePcnt string  `json:"_bid_day_change_pcnt"`
	QuoteTm          int64   `json:"_quote_tm"`
	Pips             float64 `json:"_pips"`
	PipsLot          float64 `json:"_pips_lot"`
	Digits           float64 `json:"_digits"`
	MonthMin         float64 `json:"_30d_min_bid_price"`
	MonthMax         float64 `json:"_30d_max_bid_price"`
}

type displayQuote struct {
	time          string
	symbol        string
	bid           float64
	percentChange string
	change        float64
	high          float64
	low           float64
	webURL        string
	err           error
}

var (
	myConfig        map[string]string
	readTimeout, _  = time.ParseDuration("4800ms")
	writeTimeout, _ = time.ParseDuration("5000ms")
	client          = &fasthttp.Client{
		ReadTimeout:              readTimeout,
		WriteTimeout:             writeTimeout,
		NoDefaultUserAgentHeader: true, // Don't send: User-Agent: fasthttp
		Dial: (&fasthttp.TCPDialer{
			Concurrency: 128,
		}).Dial,
	}
	headerContentTypeJson = []byte("application/json")
	userAgent             = randUserAgent()
)

func init() {
	var err error
	myConfig, err = godotenv.Read("/Users/rrj/Projekty/SwiftBar/.env")
	if err != nil {
		log.Fatalln("Error loading .env file")
	}
	client.TLSConfig = &tls.Config{
		InsecureSkipVerify: true,
		ClientSessionCache: tls.NewLRUClientSessionCache(0),
	}
}

func main() {
	// log.Println(myConfig)
	ts := strings.Split(myConfig["TIME_START"], ":")
	te := strings.Split(myConfig["TIME_END"], ":")
	assets := strings.Split(myConfig["ASSETS"], ":")
	city := myConfig["CITY"]
	location, _ := time.LoadLocation(city)
	tn := time.Now().In(location).Format("1504")
	weekday := time.Now().Weekday()
	app := swiftbar.New()
	if int(weekday) > 0 && int(weekday) < 6 {
		submenu := app.NewSubMenu()
		// get all quotes in paralel
		resultsChan := make(chan *displayQuote)
		activeAssets := 0
		for i, asset := range assets {
			if tn > ts[i] && tn < te[i] {
				activeAssets++
				go getQuote(asset, resultsChan)
			}
		}
		// process results
		results := 0
		if activeAssets == 0 {
			app.StatusLine("Markets closed").DropDown(false)
			goto AppRender
		}

		defer func() {
			close(resultsChan)
		}()
		for {
			quote := <-resultsChan
			results++
			if quote.err != nil {
				// just quietly ignore errors - there is too many things that can go wrong
				// (wifi off, no internet, timeout etc.)
				// log.Println(quote.err.Error())
				submenu.Line(quote.err.Error()).Color("red").Length(25)
			} else {
				var color string
				l := fmt.Sprintf("%s: %.5g %s", quote.symbol, quote.bid, quote.percentChange)
				line := app.StatusLine(l).DropDown(false)
				if quote.change < 0.0 {
					color = "red"
				} else {
					color = "green,white"
				}
				line.Color(color)
				m := fmt.Sprintf("%s - %s: %.5g %.5g", quote.time, quote.symbol, quote.bid, quote.change)
				a := fmt.Sprintf("%s: %.5g %s [%.5g - %.5g]", quote.symbol, quote.bid, quote.percentChange, quote.low, quote.high)
				// submenu.Line(m).Href(quote.webURL).Color(color)
				submenu.Line(m).WebView(quote.webURL, 640, 480).Color(color)
				submenu.Line(a).Alternate(true).Href(quote.webURL).Color(color)
			}
			// STOP if we've received all quotes
			if results == activeAssets {
				break
			}
		}
	} else {
		app.StatusLine("Weekend - Markets closed").DropDown(false)
	}
AppRender:
	app.Render()
}

/*getQuote
 */
func getQuote(asset string, ch chan<- *displayQuote) {
	var q displayQuote
	var statusCode int
	var respBody []byte
	var err error
	var errName string
	apiURL := fmt.Sprintf("%s%s.", myConfig["API_URL"], asset)
	if statusCode, respBody, err = fastGet(apiURL); err != nil {
		errName, _ = httpConnError(err)
		log.Printf("%s %s", errName, err.Error())
		if errName == "timeout" {
			// retry once after in case of timeout
			log.Println("retrying after timeout...")
			statusCode, respBody, err = fastGet(apiURL)
		}
	}
	// log.Printf("%s %v %v", statusCode, respBody, err)
	if err == nil && statusCode == fasthttp.StatusOK {
		var body Quotes
		r := bytes.NewReader(respBody) //fasthttp is not providing io.reader
		json.NewDecoder(r).Decode(&body)
		tm := time.Unix(0, body[0].QuoteTm*int64(time.Millisecond))
		city := myConfig["CITY"]
		location, _ := time.LoadLocation(city)
		q.time = tm.In(location).Format("15:04:05")
		q.symbol = asset
		q.bid = body[0].BidPrice
		q.change = body[0].BidDayChange
		q.percentChange = body[0].BidDayChangePcnt
		q.high = body[0].HighBidPrice
		q.low = body[0].LowBidPrice
		webURL := myConfig["WEB_URL"]
		q.webURL = fmt.Sprintf("%s?a=%s", webURL, asset)
	} else {
		q.err = err
	}
	ch <- &q
}

/*
	fastGet - make Get request with fasthttp

`
*/
func fastGet(url string) (int, []byte, error) {
	request := fasthttp.AcquireRequest()
	request.SetRequestURI(url)
	request.Header.SetMethod(fasthttp.MethodGet)
	request.Header.SetContentTypeBytes(headerContentTypeJson)
	request.Header.SetUserAgent(userAgent)
	response := fasthttp.AcquireResponse()
	err := client.Do(request, response)
	fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)
	if err == nil {
		respBody := response.Body()
		return response.StatusCode(), respBody, nil
	}
	return 0, nil, err
}

func httpConnError(err error) (string, bool) {
	errName := ""
	known := false
	if err == fasthttp.ErrTimeout {
		errName = "timeout"
		known = true
	} else if err == fasthttp.ErrNoFreeConns {
		errName = "conn_limit"
		known = true
	} else if err == fasthttp.ErrConnectionClosed {
		errName = "conn_close"
		known = true
	} else {
		errName = reflect.TypeOf(err).String()
		if errName == "*net.OpError" {
			// Write and Read errors are not so often and in fact they just mean timeout problems
			errName = "timeout"
			known = true
		}
	}
	return errName, known
}
