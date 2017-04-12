package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"golang.org/x/net/html/charset"
	"log"
	"net/http"
	"os"
	"strconv"
  "regexp"
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
	curses := make(map[string]float64)

	now_date := formatDate()

	resp, err := http.Get(configuration.Url + now_date)
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
    // конвертируем строку в число с плавающей точкой
    dig, err := strconv.ParseFloat(num, 64)
    // если ок - сохряняем в карту
		if err == nil {
			curses[value.CharCode] = dig
		}

	}

//Теперь нужно сохранить значения валют в БД, которые прописаны в конфиге
	 fmt.Printf("%v", curses["USD"])
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
func formatDate() string {

	t := time.Now()

	var year = fmt.Sprintf("%v", t.Year())
	var month = fmt.Sprintf("%d", t.Month())
	var day = fmt.Sprintf("%v", t.Day())

	if len(month) < 2 {
		month = "0" + month
	}

	if len(day) < 2 {
		day = "0" + day
	}

	format := day + "/" + month + "/" + year

	return format
}

// func checkCode(valute_code string, config_code []string) bool {
//   fmt.Printf("%v",config_code)
//   return true
// }
