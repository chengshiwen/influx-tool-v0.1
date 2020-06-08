package util

import (
	"encoding/csv"
	"fmt"
	"github.com/chengshiwen/influx-tool/backend"
	"github.com/deckarep/golang-set"
	"github.com/influxdata/influxdb1-client/models"
	"github.com/influxdata/influxql"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func reformFieldKeys(fieldKeys map[string][]string, castFields map[string][]string) (fieldMap map[string]string, keyClause string) {
	// The SELECT statement returns all field values if all values have the same type.
	// If field value types differ across shards, InfluxDB first performs any applicable cast operations and
	// then returns all values with the type that occurs first in the following list: float, integer, string, boolean.
	fieldSet := make(map[string]mapset.Set, len(fieldKeys))
	for field, types := range fieldKeys {
		fieldSet[field] = NewSetFromStrSlice(types)
	}
	fieldMap = make(map[string]string, len(fieldKeys))
	selects := []string{"*"}
	for field, types := range fieldKeys {
		if len(types) == 1 {
			fieldMap[field] = types[0]
		} else {
			tmap := fieldSet[field]
			fok := tmap.Contains("float")
			iok := tmap.Contains("integer")
			sok := tmap.Contains("string")
			if fok || iok {
				if fok {
					// force cast to float whether there is an integer
					fieldMap[field] = "float"
				} else {
					fieldMap[field] = "integer"
				}
				if sok {
					selects = append(selects, fmt.Sprintf("\"%s\"::string", field))
				}
				// discard boolean
			} else {
				fieldMap[field] = "boolean"
				selects = append(selects, fmt.Sprintf("\"%s\"::boolean", field))
			}
		}
	}
	keyClause = strings.Join(selects, ", ")

	for ft, fields := range castFields {
		for _, field := range fields {
			if _, ok := fieldKeys[field]; ok && fieldMap[field] == "string" {
				fieldMap[field] = ft
			}
		}
	}
	return
}

func GetDMLHeader(db string) string {
	return strings.Join([]string{
		"# DDL",
		fmt.Sprintf("CREATE DATABASE %s WITH NAME autogen", influxql.QuoteIdent(db)),
		"# DML",
		fmt.Sprintf("# CONTEXT-DATABASE:%s", db),
		"# CONTEXT-RETENTION-POLICY:autogen",
	}, "\n")
}

func Export(be *backend.Backend, db, measurement string, start, end int64, dir string, castFields map[string][]string, merge bool) (err error) {
	timeClause := fmt.Sprintf("where time >= %ds and time <= %ds", start, end)

	tagKeys := be.GetTagKeys(db, measurement)
	tagMap := NewSetFromStrSlice(tagKeys)
	fieldKeys := be.GetFieldKeys(db, measurement)
	fieldMap, keyClause := reformFieldKeys(fieldKeys, castFields)

	rsp, err := be.QueryIQL(db, fmt.Sprintf("select %s from \"%s\" %s", keyClause, measurement, timeClause))
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

	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("multi data type error from %s on %s: %s\n", err, db, measurement)
		}
	}()

	var lines []string
	if !merge {
		lines = append(lines, GetDMLHeader(db))
	}
	headerTotal := 1 + len(tagKeys) + len(fieldKeys)
	for _, value := range series[0].Values {
		mtagSet := []string{EscapeMeasurement(measurement)}
		fieldSet := make([]string, 0)
		for i := 1; i < len(value); i++ {
			k := columns[i]
			v := value[i]
			if tagMap.Contains(k) {
				if v != nil {
					mtagSet = append(mtagSet, fmt.Sprintf("%s=%s", EscapeTag(k), EscapeTag(v.(string))))
				}
			} else {
				if i >= headerTotal {
					if idx := strings.LastIndex(k, "_"); idx > -1 {
						k = k[:idx]
					}
				}
				if vtype, ok := fieldMap[k]; ok && v != nil {
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

func ExportCsv(be *backend.Backend, db, measurement string, start, end int64, dir string, castFields map[string][]string) (err error) {
	timeClause := fmt.Sprintf("where time >= %ds and time <= %ds", start, end)

	tagKeys := be.GetTagKeys(db, measurement)
	tagMap := NewSetFromStrSlice(tagKeys)
	fieldKeys := be.GetFieldKeys(db, measurement)
	fieldMap, keyClause := reformFieldKeys(fieldKeys, castFields)

	rsp, err := be.QueryIQL(db, fmt.Sprintf("select %s from \"%s\" %s", keyClause, measurement, timeClause))
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

	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("export panic from %s on %s: %s\n", err, db, measurement)
		}
	}()

	w, err := os.Create(filepath.Join(dir, measurement+".csv"))
	if err != nil {
		panic(err)
	}
	defer w.Close()
	w.WriteString("\xEF\xBB\xBF")
	csvw := csv.NewWriter(w)
	headerTotal := 1 + len(tagKeys) + len(fieldKeys)
	csvHeaders := append([]string{"name"}, columns[:headerTotal]...)
	csvw.Write(csvHeaders)

	for _, value := range series[0].Values {
		records := make([]string, 0, headerTotal+1)
		ts, _ := time.Parse(time.RFC3339Nano, value[0].(string))
		records = append(records, measurement)
		records = append(records, fmt.Sprintf("%d", ts.UnixNano()))
		smap := make(map[string]string, len(tagKeys)+len(fieldKeys))
		for i := 1; i < len(value); i++ {
			k := columns[i]
			v := value[i]
			if tagMap.Contains(k) {
				if v != nil {
					smap[k] = v.(string)
				} else {
					smap[k] = ""
				}
			} else {
				if i >= headerTotal {
					if idx := strings.LastIndex(k, "_"); idx > -1 {
						k = k[:idx]
					}
				}
				if _, ok := fieldMap[k]; ok {
					if v != nil {
						smap[k] = fmt.Sprintf("%v", v)
					} else {
						smap[k] = ""
					}
				}
			}
		}
		for i := 1; i < headerTotal; i++ {
			records = append(records, smap[columns[i]])
		}
		csvw.Write(records)
	}
	csvw.Flush()
	return
}
