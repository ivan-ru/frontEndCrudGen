package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Table ...
type Table struct {
	Name  string
	Field []DBField
}

// DBField ...
type DBField struct {
	Name     string
	DataType string
	Length   int
}

var (
	tableName = flag.String("table", "", "Table Name")
	table     []Table
	dbDriver  string
	dbCred    string
)

func init() {
	err := godotenv.Load("conf/env")
	if err != nil {
		checkErr(err)
		panic(1)
	}
	dbDriver = os.Getenv("DB_DRIVER")
	dbCred = os.Getenv("DB_CRED")
	flag.Parse()
	check()
}

func check() {
	if *tableName == "" {
		fmt.Println("Define -table params")
		os.Exit(1)
		return
	}
	if dbDriver == "" {
		fmt.Println("Define db driver")
		os.Exit(1)
		return
	}
	if dbCred == "" {
		fmt.Println("Define db cred")
		os.Exit(1)
		return
	}
}

func main() {
	cmd := `bee generate appcode -tables="` + *tableName + `" -driver=` + dbDriver + ` -conn="` + dbCred + `" -level=1`
	exec.Command(
		"sh",
		"-c",
		cmd).Output()
	time.Sleep(2 * time.Second)
	tableSplit := strings.Split(*tableName, ",")
	for i := 0; i < len(tableSplit); i++ {
		var tableTemp Table
		tableTemp.Name = tableSplit[i]
		modelByte, err := ioutil.ReadFile("models/" + tableSplit[i] + ".go")
		if err != nil {
			checkErr(err)
		}
		modelString := stringBetween(string(modelByte), "struct {\n", "\n}")
		modelField := strings.Split(modelString, "\n")
		var dbField []DBField
		for j := 0; j < len(modelField); j++ {
			var dbFieldTemp DBField
			modelField[j] = standardizeSpaces(modelField[j])
			modelFieldProps := strings.Split(modelField[j], " ")
			dbFieldTemp.DataType = modelFieldProps[1]

			ormString := strings.Replace(strings.Replace(modelFieldProps[2], "`orm:\"", "", -1), "\"`", "", -1)
			ormSplit := strings.Split(ormString, ";")
			for k := 0; k < len(ormSplit); k++ {
				if strings.Contains(ormSplit[k], "column") {
					dbFieldTemp.Name = strings.Replace(strings.Replace(ormSplit[k], "column(", "", -1), ")", "", -1)
				} else if strings.Contains(ormSplit[k], "size") {
					dbFieldTemp.Length, _ = strconv.Atoi(strings.Replace(strings.Replace(ormSplit[k], "size(", "", -1), ")", "", -1))
				}
			}
			dbField = append(dbField, dbFieldTemp)
		}
		tableTemp.Field = dbField
		table = append(table, tableTemp)
	}

	for l := 0; l < len(table); l++ {
		// fmt.Println(table[l])
		var htmlString string
		for n := 0; n < len(table[l].Field); n++ {
			var err error
			var formArrByte []byte
			if table[l].Field[n].DataType == "string" {
				if table[l].Field[n].Length > 50 {
					formArrByte, err = ioutil.ReadFile("templates/form_textarea.html")
				} else {
					formArrByte, err = ioutil.ReadFile("templates/form_text.html")
				}
			} else if table[l].Field[n].DataType == "int" {
				formArrByte, err = ioutil.ReadFile("templates/form_number.html")
			} else if table[l].Field[n].DataType == "time.Time" {
				formArrByte, err = ioutil.ReadFile("templates/form_datetimetz.html")
			} else if table[l].Field[n].DataType == "bool" {
				formArrByte, err = ioutil.ReadFile("templates/form_bool.html")
			}
			if err != nil {
				checkErr(err)
			}
			formString := string(formArrByte)
			htmlString += strings.Replace(formString, "[field]", table[l].Field[n].Name, -1)
		}
		// write form
		f, err := os.Create("result/" + table[l].Name + ".html")
		if err != nil {
			log.Fatal("error create file", err)
			return
		}
		defer f.Close()
		w := bufio.NewWriter(f)
		_, err = w.WriteString(htmlString)
		if err != nil {
			log.Fatal("error write to "+table[l].Name+".html", err)
			return
		}
		w.Flush()
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func stringBetween(value string, a string, b string) string {
	// Get substring between two strings.
	posFirst := strings.Index(value, a)
	if posFirst == -1 {
		return ""
	}
	posLast := strings.Index(value, b)
	if posLast == -1 {
		return ""
	}
	posFirstAdjusted := posFirst + len(a)
	if posFirstAdjusted >= posLast {
		return ""
	}
	return value[posFirstAdjusted:posLast]
}

func standardizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
