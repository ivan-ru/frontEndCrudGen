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

	"github.com/iancoleman/strcase"
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
	template  string
	err       error
)

func init() {
	err := godotenv.Load("conf/env")
	if err != nil {
		checkErr(err)
		panic(1)
	}
	dbDriver = os.Getenv("DB_DRIVER")
	dbCred = os.Getenv("DB_CRED")
	template = os.Getenv("TEMPLATE")
	flag.Parse()
	check()
}

func main() {
	// prepare folders and model
	prepare()

	// convert process to table and dbfield struct
	table = convertToDBStruct()

	// delete temporary models dir after generation and delete old result
	exec.Command(
		"sh",
		"-c",
		"rm -rf models/ && rm -rf result/").Output()

	// loop through inputted table
	for l := 0; l < len(table); l++ {
		var bindOnChangeString, defaultStateString, funcOnChangeString, fieldString, formString string

		// loop through fields to generate needed things in js file
		for n := 0; n < len(table[l].Field); n++ {
			bindOnChangeStringTemp, defaultStateStringTemp, funcOnChangeStringTemp, fieldStringTemp, formStringTemp := generateJSString(table[l].Field[n])
			bindOnChangeString += bindOnChangeStringTemp
			defaultStateString += defaultStateStringTemp
			funcOnChangeString += funcOnChangeStringTemp
			fieldString += fieldStringTemp
			formString += formStringTemp
		}

		// write JS file
		generateJSFile(
			table[l].Name,
			bindOnChangeString,
			defaultStateString,
			funcOnChangeString,
			fieldString,
			formString,
		)
	}
}

// HELPERS
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

// HELPERS

// CORE FUNC
func check() {
	if *tableName == "" {
		fmt.Println("Define -table params")
		os.Exit(1)
		return
	}
	if dbDriver == "" {
		fmt.Println("Define db driver on env")
		os.Exit(1)
		return
	}
	if dbCred == "" {
		fmt.Println("Define db cred on env")
		os.Exit(1)
		return
	}
	if template == "" {
		fmt.Println("Define template on env")
		os.Exit(1)
		return
	}
}

func prepare() {
	prepare()
	os.MkdirAll("result", os.ModePerm)
	cmd := `bee generate appcode -tables="` + *tableName + `" -driver=` + dbDriver + ` -conn="` + dbCred + `" -level=1`
	exec.Command(
		"sh",
		"-c",
		cmd).Output()
	time.Sleep(2 * time.Second)
}

func convertToDBStruct() (table []Table) {
	tableSplit := strings.Split(*tableName, ",")
	for i := 0; i < len(tableSplit); i++ {
		var tableTemp Table
		tableTemp.Name = tableSplit[i]
		modelByte, err := ioutil.ReadFile("models/" + tableSplit[i] + ".go")
		if err != nil {
			checkErr(err)
			log.Fatal("error read file")
			return
		}
		modelString := stringBetween(string(modelByte), "struct {\n", "\n}")
		modelField := strings.Split(modelString, "\n")
		var dbField []DBField
		for j := 0; j < len(modelField); j++ {
			dbField = append(dbField, convertStringToDBFieldStruct(modelField[j]))
		}
		tableTemp.Field = dbField
		table = append(table, tableTemp)
	}
	return
}

func convertStringToDBFieldStruct(modelField string) (dbFieldTemp DBField) {
	modelField = standardizeSpaces(modelField)
	modelFieldProps := strings.Split(modelField, " ")
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
	return dbFieldTemp
}

func getFormString(dbField DBField) (formString string) {
	var formArrByte []byte
	if dbField.DataType == "string" {
		// length only supported in mysql
		if dbField.Length > 50 {
			// text area
			formArrByte, err = ioutil.ReadFile("templates/" + template + "/form_textarea.html")
		} else {
			// usual textfield
			formArrByte, err = ioutil.ReadFile("templates/" + template + "/form_text.html")
		}
	}
	// else if dbField.DataType == "int" {
	// 	// number input
	// 	formArrByte, err = ioutil.ReadFile("templates/" + template + "/form_number.html")
	// } else if dbField.DataType == "time.Time" {
	// 	// date time with timezone
	// 	formArrByte, err = ioutil.ReadFile("templates/" + template + "/form_datetimetz.html")
	// } else if dbField.DataType == "bool" {
	// 	// checkbox
	// 	formArrByte, err = ioutil.ReadFile("templates/" + template + "/form_bool.html")
	// }
	if err != nil {
		checkErr(err)
		log.Fatal("Form Template not found")
		return
	}
	formString = string(formArrByte)
	return
}

func generateJSFile(
	tableName,
	bindOnChangeString string,
	defaultStateString string,
	funcOnChangeString string,
	fieldString string,
	formString string,
) {
	// Create JS
	addFileContent, err := ioutil.ReadFile("templates/" + template + "/js/add")
	if err != nil {
		checkErr(err)
	}
	addFileContentString := string(addFileContent)
	addFileContentString = strings.Replace(addFileContentString, "[bindOnChangeString]", bindOnChangeString, -1)
	addFileContentString = strings.Replace(addFileContentString, "[defaultStateString]", defaultStateString, -1)
	addFileContentString = strings.Replace(addFileContentString, "[funcOnChangeString]", funcOnChangeString, -1)
	addFileContentString = strings.Replace(addFileContentString, "[fieldString]", fieldString, -1)
	addFileContentString = strings.Replace(addFileContentString, "[formString]", formString, -1)

	// write form
	f, err := os.Create("result/" + tableName + "/add.js")
	if err != nil {
		log.Fatal("error create file", err)
		return
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	_, err = w.WriteString(addFileContentString)
	if err != nil {
		log.Fatal("error write to add.js", err)
		return
	}
	w.Flush()
}

func generateJSString(dbField DBField) (
	bindOnChangeString string,
	defaultStateString string,
	funcOnChangeString string,
	fieldString string,
	formString string,
) {
	// Create JS
	bindOnChangeString += "this.onChange" +
		strcase.ToCamel(dbField.Name) + " = this.onChange" +
		strcase.ToCamel(dbField.Name) + ".bind(this);\n"

	defaultStateString += "" + dbField.Name + ":'',\n"

	funcOnChangeString += "onChange" + strcase.ToCamel(dbField.Name) + "(e){\n" +
		"this.setState({\n" +
		dbField.Name + ":e.target.value\n" +
		"});\n" +
		"}\n\n"

	fieldString = dbField.Name + ": this.state." + dbField.Name + ",\n"

	formString = getFormString(dbField)
	formString = strings.Replace(formString, "[field]", dbField.Name, -1)
	formString = strings.Replace(formString, "[fieldCamel]", strcase.ToCamel(dbField.Name), -1)
	return
}

// CORE FUNC
