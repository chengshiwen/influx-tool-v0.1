package main

import (
    "flag"
    "fmt"
    "github.com/chengshiwen/influx-tool/backend"
    "github.com/chengshiwen/influx-tool/tool"
    "runtime"
    "sync"
)

var (
    Host            string
    Port            int
    Database        string
    Measurements    string
    Username        string
    Password        string
    Ssl             bool
    Cpu             int
    Version         bool
    GitCommit       string
    BuildTime       string
    Wg              *sync.WaitGroup
)

func main() {
    flag.StringVar(&Host, "host", "127.0.0.1", "host to connect to")
    flag.IntVar(&Port, "port", 8086, "port to connect to")
    flag.StringVar(&Database, "database", "", "database to connect to the server")
    flag.StringVar(&Measurements, "measurements", "", "measurements split by ',' while return all measurements if empty")
    flag.StringVar(&Username, "username", "", "username to connect to the server")
    flag.StringVar(&Password, "password", "", "password to connect to the server")
    flag.BoolVar(&Ssl, "ssl", false, "use https for requests")
    flag.IntVar(&Cpu, "cpu", 1, "cpu number to export")
    flag.BoolVar(&Version, "version", false, "display the version and exit")
    flag.Parse()
    if Version {
        fmt.Printf("Version:    %s\n", "0.1.0")
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
    if Cpu <= 0 || Cpu > runtime.NumCPU() {
        fmt.Println("invalid cpu")
        return
    }

    backend := backend.NewBackend(Host, Port, Username, Password, Ssl)
    measurements := make([]string, 0)
    if Measurements == "" {
        measurements = backend.GetMeasurements(Database)
    } else {
        measurements = tool.String2Array(Measurements)
    }

    Wg := &sync.WaitGroup{}
    for i, measurement := range measurements {
        Wg.Add(1)
        go func(measurement string) {
            tool.Export(backend, Database, measurement)
            defer Wg.Done()
        }(measurement)
        if i + 1 == len(measurements) || (i + 1) % Cpu == 0 {
            Wg.Wait()
        }
    }
    fmt.Printf("%d measurements export done\n", len(measurements))
}

