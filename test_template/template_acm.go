package main

import (
	"fmt"
	"html/template"
	"os"
	"strconv"
	"strings"
)

type temp struct {
	MaxCPU, MinCPU string
	CPU            string
}

func main() {

	var tmplFile = "sysfs.tmpl"

	values := temp{
		MaxCPU: "280",
		MinCPU: "250",
		CPU:    "0-2,10-11",
	}

	funcMap := template.FuncMap{
		"cpuList": SpecialStringtoArray,
	}
	tmpl, err := template.New(tmplFile).Funcs(funcMap).ParseFiles(tmplFile)
	if err != nil {
		fmt.Println(err)
	}
	err = tmpl.Execute(os.Stdout, values)
	if err != nil {
		fmt.Println(err)
	}

}

func SpecialStringtoArray(val string) []string {

	var arr []string

	splittedString := strings.Split(val, ",")
	for _, val := range splittedString {
		val = strings.TrimSpace(val)
		arr = func(val string, arr []string) []string {
			var intArr []int
			var strArray []string

			split := strings.Split(val, "-")
			if len(split) < 2 {
				arr = append(arr, split[0])
				return arr
			}
			currentValue := split[0]
			nextValue := split[1]

			val0, _ := strconv.Atoi(currentValue)
			val1, _ := strconv.Atoi(nextValue)

			if (val0 + 1) == val1 {
				for _, v := range split {
					arr = append(arr, v)
				}
			} else {
				for i := val0 - 1; i < val1; i++ {
					intArr = append(intArr, i+1)
				}
				for _, y := range intArr {
					strArray = append(strArray, strconv.Itoa(y))
				}
				arr = append(arr, strArray...)
			}
			return arr
		}(val, arr)
	}
	return arr
}
