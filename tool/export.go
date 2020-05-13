package tool

import (
    "fmt"
    "github.com/chengshiwen/influx-tool/backend"
    "github.com/influxdata/influxdb1-client/models"
    "github.com/influxdata/influxql"
    "io/ioutil"
    "path/filepath"
    "strings"
    "time"
)

func Export(be *backend.Backend, db, measurement, dir string) (err error) {
    rsp, err := be.QueryIQL(db, fmt.Sprintf("select * from \"%s\"", measurement))
    if err != nil {
        return
    }
    series, err := backend.SeriesFromResponseBytes(rsp)
    if err != nil {
        return
    }
    if len(series) < 1 {
        fmt.Printf("select empty data from %s on %s\n", db, measurement)
        return
    }
    columns := series[0].Columns

    tagKeys := be.GetTagKeys(db, measurement)
    tagMap := make(map[string]bool, 0)
    for _, t := range tagKeys {
        tagMap[t] = true
    }
    fieldKeys := be.GetFieldKeys(db, measurement)

    defer func() {
        if err := recover(); err != nil {
            fmt.Printf("multi data type error from %s on %s: %s\n", err, db, measurement)
        }
    }()

    var lines []string
    lines = append(lines,
        "# DDL",
        fmt.Sprintf("CREATE DATABASE %s WITH NAME autogen", influxql.QuoteIdent(db)),
        "# DML",
        fmt.Sprintf("# CONTEXT-DATABASE:%s", db),
        "# CONTEXT-RETENTION-POLICY:autogen",
    )
    for _, value := range series[0].Values {
        mtagSet := []string{EscapeMeasurement(measurement)}
        fieldSet := make([]string, 0)
        for i := 1; i < len(value); i++ {
            k := columns[i]
            v := value[i]
            if _, ok := tagMap[k]; ok {
                if v != nil {
                    mtagSet = append(mtagSet, fmt.Sprintf("%s=%s", EscapeTag(k), EscapeTag(v.(string))))
                }
            } else if vtype, ok := fieldKeys[k]; ok {
                if v != nil {
                    if vtype == "float" || vtype == "boolean" {
                        fieldSet = append(fieldSet, fmt.Sprintf("%s=%v", EscapeTag(k), v))
                    } else if vtype == "integer" {
                        fieldSet = append(fieldSet, fmt.Sprintf("%s=%vi", EscapeTag(k), v))
                    } else if vtype == "string" {
                        fieldSet = append(fieldSet, fmt.Sprintf("%s=\"%s\"", EscapeTag(k), models.EscapeStringField(v.(string))))
                    }
                }
            }
        }
        mtagStr := strings.Join(mtagSet, ",")
        fieldStr := strings.Join(fieldSet, ",")
        ts, _ := time.Parse(time.RFC3339Nano, value[0].(string))
        line := fmt.Sprintf("%s %s %d", mtagStr, fieldStr, ts.UnixNano())
        lines = append(lines, line)
    }
    if len(lines) != 0 {
        data := []byte(strings.Join(lines, "\n") + "\n")
        ioutil.WriteFile(filepath.Join(dir, measurement+".txt"), data, 0644)
    }
    return
}
