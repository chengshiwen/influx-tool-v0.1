package tool

import (
    "strings"
)

func String2Array(str string) []string {
    var values []string
    str = strings.Trim(str, ", ")
    if str != "" {
        values = strings.Split(str, ",")
    }
    return values
}
