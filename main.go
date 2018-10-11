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
	tableName        = flag.String("table", "", "Table Name")
	table            []Table
	dbDriver         string
	dbCred           string
	template         string
	err              error
	pageStringConfig = make(map[string][]string)
	inputfieldString = make(map[string]string)
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

	// convertFieldStringConfToStruct
	pageStringConfig = convertFieldStringConfToStruct()

	// getAllFieldString
	inputfieldString = getAllFieldString()

	// delete temporary models dir after generation
	exec.Command(
		"sh",
		"-c",
		"rm -rf models/").Output()

	// loop through inputted table
	for l := 0; l < len(table); l++ {
		finalJSString := make(map[string]string)

		// loop through fields to generate needed things in js file
		for n := 0; n < len(table[l].Field); n++ {
			JSString := generateJSString(table[l].Field[n])
			fmt.Println(JSString)
			for index, el := range JSString {
				finalJSString[index] += el
			}
		}
		finalJSString["defaultStateString"] += "alert_message:''"
		for index, el := range finalJSString {
			finalJSString[index] = strings.TrimSpace(el)
		}

		// write JS file
		generateJSFile(
			table[l].Name,
			finalJSString,
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
	// delete old result and generate temp models file
	cmd := `bee generate appcode -tables="` + *tableName + `" -driver=` + dbDriver + ` -conn="` + dbCred + `" -level=1 && rm -rf result/`
	exec.Command(
		"sh",
		"-c",
		cmd).Output()

	// sleep to wait bee generate process
	time.Sleep(2 * time.Second)

	// create clean result folder
	os.MkdirAll("result", os.ModePerm)
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

func convertFieldStringConfToStruct() (pageStringConf map[string][]string) {
	pageStringConfTemp := make(map[string][]string)
	// set fieldStringConfig in a map
	fieldStringConfMap, err := ioutil.ReadFile("templates/" + template + "/fieldString/conf/map")
	if err != nil {
		log.Fatal("error read file fieldstring mapconfig", err)
		return
	}
	fieldStringConfMapString := string(fieldStringConfMap)
	fieldStringConfMapArr := strings.Split(fieldStringConfMapString, "\n")

	// loop through pages config
	for i := 0; i < len(fieldStringConfMapArr); i++ {
		pageConfSplit := strings.Split(fieldStringConfMapArr[i], "=")
		pageConfArr := strings.Split(pageConfSplit[1], ",")
		pageStringConfTemp[pageConfSplit[0]] = pageConfArr
	}
	pageStringConf = pageStringConfTemp
	return
}

func getAllFieldString() (inputfieldStr map[string]string) {
	inputfieldStrTemp := make(map[string]string)
	files, err := ioutil.ReadDir("templates/default/fieldString")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		fieldStringFromFile, err := ioutil.ReadFile("templates/" + template + "/fieldString/" + f.Name())
		if err != nil {
			checkErr(err)
		}
		inputfieldStrTemp[f.Name()] = string(fieldStringFromFile)
	}
	inputfieldStr = inputfieldStrTemp
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
			formArrByte, err = ioutil.ReadFile("templates/" + template + "/form/form_textarea")
		} else {
			// usual textfield
			formArrByte, err = ioutil.ReadFile("templates/" + template + "/form/form_text")
		}
	}
	// else if dbField.DataType == "int" {
	// 	// number input
	// 	formArrByte, err = ioutil.ReadFile("templates/" + template + "/form/form_number")
	// } else if dbField.DataType == "time.Time" {
	// 	// date time with timezone
	// 	formArrByte, err = ioutil.ReadFile("templates/" + template + "/form/form_datetimetz")
	// } else if dbField.DataType == "bool" {
	// 	// checkbox
	// 	formArrByte, err = ioutil.ReadFile("templates/" + template + "/form/form_bool")
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
	tableName string,
	JSString map[string]string,
) {
	// // Read from Pages Template
	// addFileContent, err := ioutil.ReadFile("templates/" + template + "/Add")
	// if err != nil {
	// 	checkErr(err)
	// }
	// addFileContentString := string(addFileContent)

	// editFileContent, err := ioutil.ReadFile("templates/" + template + "/Edit")
	// if err != nil {
	// 	checkErr(err)
	// }
	// editFileContentString := string(editFileContent)

	// indexFileContent, err := ioutil.ReadFile("templates/" + template + "/Index")
	// if err != nil {
	// 	checkErr(err)
	// }
	// indexFileContentString := string(indexFileContent)

	// listingFileContent, err := ioutil.ReadFile("templates/" + template + "/Listing")
	// if err != nil {
	// 	checkErr(err)
	// }
	// listingFileContentString := string(listingFileContent)

	pageFiles, err := ioutil.ReadDir("templates/" + template + "/page")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range pageFiles {
		fileContent, err := ioutil.ReadFile("templates/" + template + "/page/" + f.Name())
		if err != nil {
			checkErr(err)
		}
		fileContentString := string(fileContent)
		for _, el := range pageStringConfig[f.Name()] {
			fileContentString = strings.Replace(fileContentString, "["+el+"]", JSString[el], -1)
		}
		fileContentString = strings.Replace(fileContentString, "[table]", tableName, -1)

		os.MkdirAll("result/"+tableName, os.ModePerm)
		f, err := os.Create("result/" + tableName + "/" + f.Name() + ".js")
		if err != nil {
			log.Fatal("error create file", err)
			return
		}
		defer f.Close()
		w := bufio.NewWriter(f)
		_, err = w.WriteString(fileContentString)
		if err != nil {
			log.Fatal("error write to "+f.Name()+".js", err)
			return
		}
		w.Flush()
	}

	// // add
	// addFileContentString = strings.Replace(addFileContentString, "[bindOnChangeString]", bindOnChangeString, -1)
	// addFileContentString = strings.Replace(addFileContentString, "[defaultStateString]", defaultStateString, -1)
	// addFileContentString = strings.Replace(addFileContentString, "[funcOnChangeString]", funcOnChangeString, -1)
	// addFileContentString = strings.Replace(addFileContentString, "[fieldString]", fieldString, -1)
	// addFileContentString = strings.Replace(addFileContentString, "[formString]", formString, -1)

	// // edit
	// editFileContentString = strings.Replace(editFileContentString, "[bindOnChangeString]", bindOnChangeString, -1)
	// editFileContentString = strings.Replace(editFileContentString, "[defaultStateString]", defaultStateString, -1)
	// editFileContentString = strings.Replace(editFileContentString, "[setStateString]", setStateString, -1)
	// editFileContentString = strings.Replace(editFileContentString, "[funcOnChangeString]", funcOnChangeString, -1)
	// editFileContentString = strings.Replace(editFileContentString, "[fieldString]", fieldString, -1)
	// editFileContentString = strings.Replace(editFileContentString, "[formString]", formString, -1)

	// // listing
	// listingFileContentString = strings.Replace(listingFileContentString, "[tableColumnString]", tableColumnString, -1)
	// listingFileContentString = strings.Replace(listingFileContentString, "[tableRowString]", tableRowString, -1)

	// addFileContentString = strings.Replace(addFileContentString, "[table]", tableName, -1)
	// editFileContentString = strings.Replace(editFileContentString, "[table]", tableName, -1)
	// indexFileContentString = strings.Replace(indexFileContentString, "[table]", tableName, -1)
	// listingFileContentString = strings.Replace(listingFileContentString, "[table]", tableName, -1)

	// write Add.js
	// os.MkdirAll("result/"+tableName, os.ModePerm)
	// fAdd, err := os.Create("result/" + tableName + "/Add.js")
	// if err != nil {
	// 	log.Fatal("error create file", err)
	// 	return
	// }
	// defer fAdd.Close()
	// wAdd := bufio.NewWriter(fAdd)
	// _, err = wAdd.WriteString(addFileContentString)
	// if err != nil {
	// 	log.Fatal("error write to Add.js", err)
	// 	return
	// }
	// wAdd.Flush()

	// // write Edit.js
	// os.MkdirAll("result/"+tableName, os.ModePerm)
	// fEdit, err := os.Create("result/" + tableName + "/Edit.js")
	// if err != nil {
	// 	log.Fatal("error create file", err)
	// 	return
	// }
	// defer fEdit.Close()
	// wEdit := bufio.NewWriter(fEdit)
	// _, err = wEdit.WriteString(editFileContentString)
	// if err != nil {
	// 	log.Fatal("error write to Edit.js", err)
	// 	return
	// }
	// wEdit.Flush()

	// // write Index.js
	// os.MkdirAll("result/"+tableName, os.ModePerm)
	// fIndex, err := os.Create("result/" + tableName + "/Index.js")
	// if err != nil {
	// 	log.Fatal("error create file", err)
	// 	return
	// }
	// defer fIndex.Close()
	// wIndex := bufio.NewWriter(fIndex)
	// _, err = wIndex.WriteString(indexFileContentString)
	// if err != nil {
	// 	log.Fatal("error write to Index.js", err)
	// 	return
	// }
	// wIndex.Flush()

	// // write Listing.js
	// os.MkdirAll("result/"+tableName, os.ModePerm)
	// fListing, err := os.Create("result/" + tableName + "/Listing.js")
	// if err != nil {
	// 	log.Fatal("error create file", err)
	// 	return
	// }
	// defer fListing.Close()
	// wListing := bufio.NewWriter(fListing)
	// _, err = wListing.WriteString(listingFileContentString)
	// if err != nil {
	// 	log.Fatal("error write to Listing.js", err)
	// 	return
	// }
	// wListing.Flush()
}

func generateJSString(dbField DBField) (
	JSString map[string]string,
) {
	inputfieldStringDuplicate := make(map[string]string)
	for k, v := range inputfieldString {
		inputfieldStringDuplicate[k] = v
	}
	JSString = inputfieldStringDuplicate
	for index := range JSString {
		if index == "formString" {
			JSString[index] = getFormString(dbField)
		}
		JSString[index] = strings.Replace(JSString[index], "[field]", dbField.Name, -1)
		JSString[index] = strings.Replace(JSString[index], "[Field]", strcase.ToCamel(dbField.Name), -1)
	}

	// bindOnChangeString = "this.onChange" +
	// 	strcase.ToCamel(dbField.Name) + " = this.onChange" +
	// 	strcase.ToCamel(dbField.Name) + ".bind(this);\n\t\t"

	// defaultStateString = "" + dbField.Name + ":'',\n\t\t\t"

	// setStateString = "this.setState({" + dbField.Name + ":response.data." + dbField.Name + "});\n\t\t\t"

	// funcOnChangeString = "\tonChange" + strcase.ToCamel(dbField.Name) + "(e){\n\t\t" +
	// 	"this.setState({\n\t\t\t" +
	// 	dbField.Name + ":e.target.value\n\t\t" +
	// 	"});\n\t" +
	// 	"}\n"

	// fieldString = dbField.Name + ": this.state." + dbField.Name + ",\n\t\t\t"

	// formString = getFormString(dbField)
	// formString = strings.Replace(formString, "[field]", dbField.Name, -1)
	// formString = strings.Replace(formString, "[Field]", strcase.ToCamel(dbField.Name), -1)

	// tableColumnString = "<th scope=\"col\">" + dbField.Name + "</th>\n\t\t\t\t\t\t"
	// tableRowString = "<td>{[table]." + dbField.Name + "}</td>\n\t\t\t\t\t\t\t\t\t"
	return
}

// CORE FUNC
