package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/johnmccabe/go-bitbar"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
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
	myEnv map[string]string

	transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Dial: (&net.Dialer{
			Timeout:   0,
			KeepAlive: 0,
		}).Dial,
		TLSHandshakeTimeout: 3 * time.Second,
	}
	// http.Clients should be reused instead of created as needed.
	client = &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}
	userAgent = randUserAgent()
)

func init() {
	var err error
	myEnv, err = godotenv.Read("/Users/rrj/Projekty/Swiftbar/.env")
	if err != nil {
		log.Fatalln("Error loading .env file")
	}
}

func main() {
	log.Println(myEnv)
	ts := strings.Split(myEnv["TIME_START"], ":")
	te := strings.Split(myEnv["TIME_END"], ":")
	assets := strings.Split(myEnv["ASSETS"], ":")
	city := myEnv["CITY"]
	location, _ := time.LoadLocation(city)
	tn := time.Now().In(location).Format("1504")
	weekday := time.Now().Weekday()
	app := bitbar.New()
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
					color = "green"
				}
				line.Color(color)
				m := fmt.Sprintf("%s - %s: %.5g %.5g", quote.time, quote.symbol, quote.bid, quote.change)
				a := fmt.Sprintf("%s: %.5g %s [%.5g - %.5g]", quote.symbol, quote.bid, quote.percentChange, quote.low, quote.high)
				submenu.Line(m).Href(quote.webURL).Color(color)
				submenu.Line(a).Alternate(true).Href(quote.webURL).Color(color)
			}
			// stop if we've received all quotes
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
	city := myEnv["CITY"]
	webURL := myEnv["WEB_URL"]
	apiURL := fmt.Sprintf("%s%s.", myEnv["API_URL"], asset)
	request, _ := http.NewRequest("GET", apiURL, nil)
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Connection", "close")
	if response, err := client.Do(request); err == nil {
		var body Quotes
		json.NewDecoder(response.Body).Decode(&body)
		tm := time.Unix(0, body[0].QuoteTm*int64(time.Millisecond))
		location, _ := time.LoadLocation(city)
		q.time = tm.In(location).Format("15:04:05")
		q.symbol = asset
		q.bid = body[0].BidPrice
		q.change = body[0].BidDayChange
		q.percentChange = body[0].BidDayChangePcnt
		q.high = body[0].HighBidPrice
		q.low = body[0].LowBidPrice
		q.webURL = fmt.Sprintf("%s?a=%s", webURL, asset)
		// No such host error on MacOS - due to limit of open connections
		defer response.Body.Close()
	} else {
		q.err = err
	}
	ch <- &q
}
