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
    Dir             string
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
    flag.StringVar(&Dir, "dir", "export", "directory to export")
    flag.IntVar(&Cpu, "cpu", 1, "cpu number to export")
    flag.BoolVar(&Version, "version", false, "display the version and exit")
    flag.Parse()
    if Version {
        fmt.Printf("Version:    %s\n", "0.1.2")
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
    if err := tool.MakeDir(Dir); err != nil {
        fmt.Println("invalid dir")
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
        go func(i int, measurement string) {
            tool.Export(backend, Database, measurement, Dir)
            fmt.Printf("%d/%d: %s.txt done\n", i+1, len(measurements), measurement)
            defer Wg.Done()
        }(i, measurement)
        if i + 1 == len(measurements) || (i + 1) % Cpu == 0 {
            Wg.Wait()
        }
    }
    fmt.Printf("%d measurements export done\n", len(measurements))
}

