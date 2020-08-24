package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/chengshiwen/influx-tool/backend"
	"github.com/chengshiwen/influx-tool/util"
	"github.com/panjf2000/ants/v2"
)

var (
	Host          string
	Port          int
	Database      string
	Measurements  string
	Range         string
	Start         string
	End           string
	Format        string
	Username      string
	Password      string
	Ssl           bool
	Dir           string
	Worker        int
	Merge         bool
	BooleanFields string
	FloatFields   string
	IntegerFields string
	Version       bool
	GitCommit     string
	BuildTime     string
	Pool          *ants.Pool
	Wg            sync.WaitGroup
)

func castFields() map[string][]string {
	booleanFields := make([]string, 0)
	floatFields := make([]string, 0)
	integerFields := make([]string, 0)
	if BooleanFields != "" {
		booleanFields = util.String2Array(BooleanFields)
	}
	if FloatFields != "" {
		floatFields = util.String2Array(FloatFields)
	}
	if IntegerFields != "" {
		integerFields = util.String2Array(IntegerFields)
	}
	return map[string][]string{"boolean": booleanFields, "float": floatFields, "integer": integerFields}
}

func main() {
	flag.StringVar(&Host, "host", "127.0.0.1", "host to connect to")
	flag.IntVar(&Port, "port", 8086, "port to connect to")
	flag.StringVar(&Database, "database", "", "database to connect to the server")
	flag.StringVar(&Measurements, "measurements", "", "measurements split by ',' while return all measurements if empty\nwildcard '*' and '?' supported")
	flag.StringVar(&Range, "range", "", "measurements range to export, as 'start,end', started from 1, included end\nignored when -measurements not empty")
	flag.StringVar(&Start, "start", "", "the start unix time to export (second precision), optional")
	flag.StringVar(&End, "end", "", "the end unix time to export (second precision), optional")
	flag.StringVar(&Format, "format", "line", "the output format to export, valid values are line or csv")
	flag.StringVar(&Username, "username", "", "username to connect to the server")
	flag.StringVar(&Password, "password", "", "password to connect to the server")
	flag.BoolVar(&Ssl, "ssl", false, "use https for requests")
	flag.StringVar(&Dir, "dir", "export", "directory to export")
	flag.IntVar(&Worker, "worker", 1, "number of concurrent workers to export")
	flag.BoolVar(&Merge, "merge", false, "merge and export into one file, ignored when -format is not line")
	flag.StringVar(&BooleanFields, "boolean-fields", "", "fields required to cast to boolean from string, split by ','")
	flag.StringVar(&FloatFields, "float-fields", "", "fields required to cast to float from string, split by ','")
	flag.StringVar(&IntegerFields, "integer-fields", "", "fields required to cast to integer from string, split by ','")
	flag.BoolVar(&Version, "version", false, "display the version and exit")
	flag.Parse()
	if Version {
		fmt.Printf("Version:    %s\n", "0.1.8")
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("Build time: %s\n", BuildTime)
		fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return
	}

	if Database == "" {
		fmt.Println("database required")
		return
	}
	if err := util.MakeDir(Dir); err != nil {
		fmt.Println("invalid dir")
		return
	}
	if Worker <= 0 || Worker > 4*runtime.NumCPU() {
		fmt.Println("invalid worker, not more than 4*cpus")
		return
	}
	if Format != "line" && Format != "csv" {
		fmt.Println("invalid format")
		return
	}

	rangeStart := 1
	rangeEnd := math.MaxUint32
	if Measurements == "" && Range != "" {
		pattern, _ := regexp.Compile(`^(\d*),(\d*)$`)
		matches := pattern.FindStringSubmatch(Range)
		if len(matches) != 3 {
			fmt.Println("invalid range")
			return
		}
		if matches[1] != "" {
			rangeStart, _ = strconv.Atoi(matches[1])
		}
		if matches[2] != "" {
			rangeEnd, _ = strconv.Atoi(matches[2])
		}
		if rangeStart == 0 || rangeStart > rangeEnd {
			fmt.Println("invalid range")
			return
		}
	}

	var StartTime, EndTime int64
	if i, err := strconv.ParseInt(Start, 10, 64); err == nil {
		StartTime = i
	} else {
		StartTime = 0
	}
	if i, err := strconv.ParseInt(End, 10, 64); err == nil {
		EndTime = i
	} else {
		EndTime = 9223372036
	}

	backend := backend.NewBackend(Host, Port, Username, Password, Ssl)
	measurements := make([]string, 0)
	if Measurements == "" {
		measurements = backend.GetMeasurements(Database)
	} else {
		if strings.Contains(Measurements, "*") || strings.Contains(Measurements, "?") {
			patterns := util.String2Array(Measurements)
			allMeases := backend.GetMeasurements(Database)
			for _, meas := range allMeases {
				for _, pat := range patterns {
					if strings.Contains(pat, "*") || strings.Contains(pat, "?") {
						if util.WildcardMatch(pat, meas) {
							measurements = append(measurements, meas)
						}
					} else if meas == pat {
						measurements = append(measurements, meas)
					}
				}
			}
		} else {
			measurements = util.String2Array(Measurements)
		}
	}

	castFields := castFields()

	cnt := 0
	Pool, _ = ants.NewPool(Worker)
	defer Pool.Release()
	for i, measurement := range measurements {
		_i, _measurement := i, measurement
		if Range != "" && (_i < rangeStart-1 || _i >= rangeEnd) {
			continue
		}
		cnt++
		Wg.Add(1)
		Pool.Submit(func() {
			defer Wg.Done()
			if Format == "csv" {
				util.ExportCsv(backend, Database, _measurement, StartTime, EndTime, Dir, castFields)
			} else {
				util.Export(backend, Database, _measurement, StartTime, EndTime, Dir, castFields, Merge)
			}
			fmt.Printf("%d/%d: %s processed\n", _i+1, len(measurements), _measurement)
		})
	}
	Wg.Wait()
	fmt.Printf("%d/%d measurements export done\n", cnt, len(measurements))
	if Format == "line" && Merge {
		ioutil.WriteFile(filepath.Join(Dir, "merge.tmp"), []byte(util.GetDMLHeader(Database)+"\n"), 0644)
		err := exec.Command("sh", "-c", fmt.Sprintf("cat %s >> %s", filepath.Join(Dir, "*.txt"), filepath.Join(Dir, "merge.tmp"))).Run()
		if err != nil {
			fmt.Printf("merge error: %s\n", err)
			return
		}
		err = exec.Command("sh", "-c", fmt.Sprintf("cd %s && rm -f *.txt && mv merge.tmp merge.txt", Dir)).Run()
		if err != nil {
			fmt.Printf("rename error: %s\n", err)
			return
		}
	}
}
