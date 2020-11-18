package backend

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chengshiwen/influx-tool/util"
	gzip "github.com/klauspost/pgzip"
)

type Backend struct {
	Url       string // nolint:golint
	Username  string
	Password  string
	transport *http.Transport
}

func NewBackend(host string, port int, username string, password string, tlsSkip bool) *Backend {
	url := fmt.Sprintf("http://%s:%d", host, port)
	if tlsSkip {
		url = fmt.Sprintf("https://%s:%d", host, port)
	}
	return &Backend{
		Url:       url,
		Username:  username,
		Password:  password,
		transport: NewTransport(tlsSkip),
	}
}

func NewTransport(tlsSkip bool) *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Second * 30,
			KeepAlive: time.Second * 30,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       time.Second * 90,
		TLSHandshakeTimeout:   time.Second * 10,
		ExpectContinueTimeout: time.Second * 1,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: tlsSkip},
	}
}

func NewQueryRequest(method, db, q, epoch string) *http.Request {
	header := http.Header{}
	header.Set("Accept-Encoding", "gzip")
	form := url.Values{}
	form.Set("q", q)
	if db != "" {
		form.Set("db", db)
	}
	if epoch != "" {
		form.Set("epoch", epoch)
	}
	return &http.Request{Method: method, Form: form, Header: header}
}

func (be *Backend) Query(req *http.Request) (body []byte, err error) {
	if len(req.Form) == 0 {
		req.Form = url.Values{}
	}
	req.Form.Del("u")
	req.Form.Del("p")
	req.ContentLength = 0
	if be.Username != "" || be.Password != "" {
		req.SetBasicAuth(be.Username, be.Password)
	}

	req.URL, err = url.Parse(be.Url + "/query?" + req.Form.Encode())
	if err != nil {
		log.Print("internal url parse error: ", err)
		return
	}

	q := strings.TrimSpace(req.FormValue("q"))
	resp, err := be.transport.RoundTrip(req)
	if err != nil {
		log.Printf("query error: %s, the query is %s", err, q)
		return
	}
	defer resp.Body.Close()

	respBody := resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		b, err := gzip.NewReader(resp.Body)
		if err != nil {
			log.Printf("unable to decode gzip body")
			return nil, err
		}
		defer b.Close()
		respBody = b
	}

	body, err = ioutil.ReadAll(respBody)
	if err != nil {
		log.Printf("read body error: %s, the query is %s", err, q)
		return
	}
	if resp.StatusCode >= 400 {
		rsp, _ := ResponseFromResponseBytes(body)
		err = errors.New(rsp.Err)
	}
	return
}

func (be *Backend) QueryIQL(method, db, q, epoch string) ([]byte, error) {
	return be.Query(NewQueryRequest(method, db, q, epoch))
}

func (be *Backend) GetSeriesValues(db, q string) []string {
	var values []string
	p, err := be.Query(NewQueryRequest("GET", db, q, ""))
	if err != nil {
		return values
	}
	series, _ := SeriesFromResponseBytes(p)
	for _, s := range series {
		for _, v := range s.Values {
			if s.Name == "databases" && v[0].(string) == "_internal" {
				continue
			}
			values = append(values, v[0].(string))
		}
	}
	return values
}

func (be *Backend) GetMeasurements(db string) []string {
	return be.GetSeriesValues(db, "show measurements")
}

func (be *Backend) GetTagKeys(db, meas string) []string {
	return be.GetSeriesValues(db, fmt.Sprintf("show tag keys from \"%s\"", util.EscapeIdentifier(meas)))
}

func (be *Backend) GetFieldKeys(db, meas string) map[string][]string {
	fieldKeys := make(map[string][]string)
	q := fmt.Sprintf("show field keys from \"%s\"", util.EscapeIdentifier(meas))
	p, err := be.Query(NewQueryRequest("GET", db, q, ""))
	if err != nil {
		return fieldKeys
	}
	series, _ := SeriesFromResponseBytes(p)
	for _, s := range series {
		for _, v := range s.Values {
			fk := v[0].(string)
			fieldKeys[fk] = append(fieldKeys[fk], v[1].(string))
		}
	}
	return fieldKeys
}
