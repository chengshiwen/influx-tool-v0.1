package tool

import (
	"github.com/deckarep/golang-set"
	"os"
	"strings"
)

func NewSetFromStrSlice(s []string) mapset.Set {
	set := mapset.NewSet()
	for _, v := range s {
		set.Add(v)
	}
	return set
}

func PathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func MakeDir(dir string) (err error) {
	exist, err := PathExist(dir)
	if err != nil {
		return
	}
	if !exist {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return
		}
	}
	return
}

func String2Array(str string) []string {
	var values []string
	str = strings.Trim(str, ", ")
	if str != "" {
		values = strings.Split(str, ",")
	}
	return values
}
