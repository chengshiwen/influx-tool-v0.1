package main

import (
	"flag"
	"fmt"
	"github.com/chengshiwen/influx-tool/backend"
	"github.com/chengshiwen/influx-tool/util"
	"io/ioutil"
	"math"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var (
	Host          string
	Port          int
	Database      string
	Measurements  string
	Range         string
	Username      string
	Password      string
	Ssl           bool
	Dir           string
	Cpu           int
	Merge         bool
	BooleanFields string
	FloatFields   string
	IntegerFields string
	Version       bool
	GitCommit     string
	BuildTime     string
	Wg            *sync.WaitGroup
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
	flag.StringVar(&Username, "username", "", "username to connect to the server")
	flag.StringVar(&Password, "password", "", "password to connect to the server")
	flag.BoolVar(&Ssl, "ssl", false, "use https for requests")
	flag.StringVar(&Dir, "dir", "export", "directory to export")
	flag.IntVar(&Cpu, "cpu", 1, "cpu number to export")
	flag.BoolVar(&Merge, "merge", false, "merge and export into one file")
	flag.StringVar(&BooleanFields, "boolean-fields", "", "fields required to cast to boolean from string, split by ','")
	flag.StringVar(&FloatFields, "float-fields", "", "fields required to cast to float from string, split by ','")
	flag.StringVar(&IntegerFields, "integer-fields", "", "fields required to cast to integer from string, split by ','")
	flag.BoolVar(&Version, "version", false, "display the version and exit")
	flag.Parse()
	if Version {
		fmt.Printf("Version:    %s\n", "0.1.6")
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
	if Cpu <= 0 || Cpu > runtime.NumCPU() {
		fmt.Println("invalid cpu")
		return
	}
	rangeStart := 1
	rangeEnd := math.MaxUint32
	if Measurements == "" && Range != "" {
		pattern, _ := regexp.Compile("^(\\d*),(\\d*)$")
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
	Wg := &sync.WaitGroup{}
	for i, measurement := range measurements {
		if Range != "" && (i < rangeStart-1 || i >= rangeEnd) {
			continue
		}
		cnt++
		Wg.Add(1)
		go func(i int, measurement string) {
			util.Export(backend, Database, measurement, Dir, castFields, Merge)
			fmt.Printf("%d/%d: %s processed\n", i+1, len(measurements), measurement)
			defer Wg.Done()
		}(i, measurement)
		if cnt%Cpu == 0 || i == len(measurements)-1 || i == rangeEnd-1 {
			Wg.Wait()
		}
	}
	fmt.Printf("%d/%d measurements export done\n", cnt, len(measurements))
	if Merge {
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
