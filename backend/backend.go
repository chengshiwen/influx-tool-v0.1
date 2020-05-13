package backend

import (
    "bytes"
    "compress/gzip"
    "crypto/tls"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "strings"
)

type Backend struct {
    Url             string                      `json:"url"`
    Username        string                      `json:"username"`
    Password        string                      `json:"password"`
    Transport       *http.Transport             `json:"transport"`
}

func NewBackend(host string, port int, username string, password string, ssl bool) *Backend {
    url := fmt.Sprintf("http://%s:%d", host, port)
    if ssl {
        url = fmt.Sprintf("https://%s:%d", host, port)
    }
    return &Backend{
        Url: url,
        Username: username,
        Password: password,
        Transport: NewTransport(ssl),
    }
}

func NewTransport(ssl bool) *http.Transport {
    return &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: ssl}}
}

func Compress(buf *bytes.Buffer, p []byte) (err error) {
    zip := gzip.NewWriter(buf)
    defer zip.Close()
    n, err := zip.Write(p)
    if err != nil {
        return
    }
    if n != len(p) {
        err = io.ErrShortWrite
        return
    }
    return
}

func NewRequest(db, query string) *http.Request {
    header := map[string][]string{"Accept-Encoding": {"gzip"}}
    if db == "" {
        return &http.Request{Form: url.Values{"q": []string{query}}, Header: header}
    }
    return &http.Request{Form: url.Values{"db": []string{db}, "q": []string{query}}, Header: header}
}

func (backend *Backend) Query(req *http.Request) ([]byte, error) {
    var err error
    if len(req.Form) == 0 {
        req.Form = url.Values{}
    }
    req.Form.Del("u")
    req.Form.Del("p")
    req.ContentLength = 0
    if backend.Username != "" || backend.Password != "" {
        req.SetBasicAuth(backend.Username, backend.Password)
    }

    req.URL, err = url.Parse(backend.Url + "/query?" + req.Form.Encode())
    if err != nil {
        log.Print("internal url parse error: ", err)
        return nil, err
    }

    q := strings.TrimSpace(req.FormValue("q"))
    resp, err := backend.Transport.RoundTrip(req)
    if err != nil {
        log.Printf("query error: %s, the query is %s", err, q)
        return nil, err
    }
    defer resp.Body.Close()

    respBody := resp.Body
    if resp.Header.Get("Content-Encoding") == "gzip" {
        respBody, err = gzip.NewReader(resp.Body)
        defer respBody.Close()
        if err != nil {
            log.Printf("unable to decode gzip body")
            return nil, err
        }
    }

    return ioutil.ReadAll(respBody)
}

func (backend *Backend) QueryIQL(db, query string) ([]byte, error) {
    return backend.Query(NewRequest(db, query))
}

func (backend *Backend) GetSeriesValues(db, query string) []string {
    p, _ := backend.Query(NewRequest(db, query))
    series, _ := SeriesFromResponseBytes(p)
    var values []string
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

func (backend *Backend) GetMeasurements(db string) []string {
    return backend.GetSeriesValues(db, "show measurements")
}

func (backend *Backend) GetTagKeys(db, measure string) []string {
    return backend.GetSeriesValues(db, fmt.Sprintf("show tag keys from \"%s\"", measure))
}

func (backend *Backend) GetFieldKeys(db, measure string) map[string][]string {
    query := fmt.Sprintf("show field keys from \"%s\"", measure)
    p, _ := backend.Query(NewRequest(db, query))
    series, _ := SeriesFromResponseBytes(p)
    fieldKeys := make(map[string][]string)
    for _, s := range series {
        for _, v := range s.Values {
            fk := v[0].(string)
            ft := v[1].(string)
            fieldKeys[fk] = append(fieldKeys[fk], ft)
        }
    }
    return fieldKeys
}
