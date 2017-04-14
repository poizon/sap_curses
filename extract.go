package main

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	_ "github.com/denisenkom/go-mssqldb"
	"golang.org/x/net/html/charset"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

// импорт курсов в SAP
// файл с конфигурацией

type Configuration struct {
	Server  string   `json:"server"`
	Db      []string `json:"db"`
	User    string   `json:"user"`
	Pass    string   `json:"pass"`
	Url     string   `json: "url"`
	ValCode []string `json:"valcode"`
}

type Valute struct {
	NumCode  string `xml:"NumCode"`
	CharCode string `xml:"CharCode"`
	Nominal  string `xml:"Nominal"`
	Name     string `xml:"Name"`
	Value    string `xml:"Value"`
}

type Result struct {
	XMLName xml.Name `xml:"ValCurs"`
	Valute  []Valute
}

func main() {
	// попытка прочитать конфиг
	configuration, _ := readConfig()
	// готовим карту для хранения значений вида КОД_ВАЛЮТЫ => Курс на дату
	curses := make(map[string]float64)
	// получаем дату
	cur_time := time.Now()
	// читаем с сервера данные и парсим в структуру parsed
	resp, err := http.Get(configuration.Url + formatDate("web", cur_time))
	logError(err)
	var parsed Result
	defer resp.Body.Close()

	decoder := xml.NewDecoder(resp.Body)
	decoder.CharsetReader = charset.NewReaderLabel
	err = decoder.Decode(&parsed)
	logError(err)

	// заполняем хэш значениями USD => 59.22
	for _, value := range parsed.Valute {
		// заменяем все , на .
		re := regexp.MustCompile("(,{1})")
		num := re.ReplaceAllString(value.Value, ".")
		// конвертируем строку в число с плавающей точкой float64
		dig, err := strconv.ParseFloat(num, 64)
		// если ок - сохряняем в карту
		if err == nil {
			curses[value.CharCode] = dig
		}

	}

	// Теперь нужно сохранить значения валют во все БД, которые прописаны в конфиге
	// TODO: переделать чтобы можно было инсертить на разные сервера, а не в разные БД
	for _, dbname := range configuration.Db {
		dsn := "server=" + configuration.Server + ";user id=" + configuration.User + ";password=" + configuration.Pass + ";database=" + dbname
		db, err := sql.Open("mssql", dsn)
		logFatal(err)
		// ping db test
		err = db.Ping()
		logFatal(err)

		defer db.Close()

		for _, value := range configuration.ValCode {
			insertToDB(formatDate("sql", cur_time), value, curses[value], db)
		}

	}

}

// логгер ошибок
func logError(err error) {
	if err != nil {
		log.Println(err.Error())
	}
}

// обработчик фатальных ошибок
func logFatal(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

// читаем файл с конфигом, и возвращаем структуру
func readConfig() (Configuration, error) {
	// пробуем открыть файл конфига
	file, err := os.Open("config.json")
	logFatal(err)

	var config Configuration

	jsonParser := json.NewDecoder(file)

	if err = jsonParser.Decode(&config); err != nil {
		fmt.Println("parsing config file", err.Error())
	}

	return config, err

}

// форматируем дату
func formatDate(format string, t time.Time) string {

	var year = fmt.Sprintf("%v", t.Year())
	var month = fmt.Sprintf("%d", t.Month())
	var day = fmt.Sprintf("%v", t.Day())

	if len(month) < 2 {
		month = "0" + month
	}

	if len(day) < 2 {
		day = "0" + day
	}
	// если формат для sql
	if format == "sql" {
		return fmt.Sprintf("%s-%s-%s", year, day, month)
		// или для web
	} else if format == "web" {
		return fmt.Sprintf("%s/%s/%s", day, month, year)
	}

	return format
}

func insertToDB(cur_date string, charcode string, curs float64, db *sql.DB) {
	_, err := db.Query("INSERT INTO ORTT VALUES(?1,?2,?3,?4,?5)", cur_date, charcode, curs, "I", 13)
	logError(err)
}