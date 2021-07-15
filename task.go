package main

//curl  -H 'Content-Type: application/json' --data '{"url":"test1"}' http://127.0.0.1:8008/long/
//curl  -H 'Content-Type: application/json' --data '{"url":"http://127.0.0.1:8008/bPlNFG"}' http://127.0.0.1:8008/short/

/*
CREATE DATABASE testbd
    WITH
    OWNER = postgres
    ENCODING = 'UTF8'
    CONNECTION LIMIT = -1;
    
CREATE SEQUENCE url_seq;
CREATE TABLE url (
   id integer DEFAULT nextval('url_seq') NOT NULL,
   longurl character varying(512) NOT NULL,
  shorturl character varying(28) NOT NULL,
 );
 */

import(
  "net/http"
  "fmt"
  "log"
 "io/ioutil"
 "math/rand"
  "encoding/json"
  "database/sql"
   _ "github.com/lib/pq"
  sq "github.com/Masterminds/squirrel"
)

type URL struct{
  Url string
}

const path = "http://127.0.0.1:8008/"

func main(){
  http.HandleFunc("/", CheckUrl)
  http.ListenAndServe(":8008", nil)
}

func CheckUrl(w http.ResponseWriter, r *http.Request){// Проверка метода
  if r.URL.Path == "/short/"{
     ShortToLong(w, r)
  } else if r.URL.Path == "/long/"{
     LongToShort(w, r)
  } else{
    log.Println("Неправильный url = %v", r.URL.Path)
  }
}


func DeserializeJsonUrl(body string) string{
  var url URL
  err := json.Unmarshal([]byte(body), &url)
  if err != nil {
    log.Println(err)
  }
  return url.Url
}

func SerializeJsonUrl(url string) string {
  jsonurl := URL{
    Url: url,
  }

  var jsondata []byte
  jsondata, err := json.Marshal(jsonurl)
  if err != nil {
    log.Println(err)
  }
  return string(jsondata)
}

func SearchInBD(url string) (found bool, longurl string){// Поиск long url в БД

  connStr := "user=postgres password=1234 dbname=testbd sslmode=disable"
  db, err := sql.Open("postgres", connStr)

  if err != nil {
    panic("Не удалось подключиться к базе данных")
  }
  defer db.Close()

  sql := sq.Select("longurl").From("url").Where(sq.Eq{"shorturl": url}).PlaceholderFormat(sq.Dollar)

  rows, err := sql.RunWith(db).Query()
  if err != nil {
    panic(fmt.Sprintf("Ошибка строки запроса: %v", err))
  }

  for rows.Next(){
    var data string
    found = true
    rows.Scan(&data)
    longurl = data

  }
  return found, longurl
}

func ShortToLong(w http.ResponseWriter, r *http.Request){// Преорбразование long url в short url
  body, err := ioutil.ReadAll(r.Body)
  if err != nil {
    panic(err)
  }
  log.Println(string(body))
  go StL(string(body), w)
}

func StL(body string, w http.ResponseWriter){
  url := DeserializeJsonUrl(string(body))

  found, longurl := SearchInBD(url)
  if found == true{
    response := SerializeJsonUrl(longurl)
    fmt.Fprint(w, response)
  } else {
    log.Printf("url не найден")
  }
}

func InsertInDB(shorturl string, longurl string) {// Вставка url в БД
  connStr := "user=postgres password=1234 dbname=testbd sslmode=disable"
  db, err := sql.Open("postgres", connStr)
   if err != nil {
       panic("Не удалось подключиться к базе данных")
   }
   defer db.Close()

  sql  := sq.Insert("url").Columns("longurl", "shorturl").Values(longurl, path + shorturl).PlaceholderFormat(sq.Dollar).RunWith(db)
  _, sqlerr := sql.Exec()

   if sqlerr != nil {
       panic(fmt.Sprintf("Ошибка строки запроса: %v", sqlerr))
   }
}

const array = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func GenerateShortUrl(longurl string) (string){//Генерация токена
  shorturl := make([]byte, 6)// длина токена 6
  for i := 0; i < 6; i++{
    shorturl[i] = array[rand.Intn(len(array))]
  }
  return string(shorturl)
}

func LongToShort(w http.ResponseWriter, r *http.Request){// Преорбразование short url в long url
  body, err := ioutil.ReadAll(r.Body)
  if err != nil {
    panic(err)
  }
  log.Println(string(body))
  go LtS(string(body), w)
}

func LtS(body string, w http.ResponseWriter){
  longurl := DeserializeJsonUrl(string(body))

  shorturl := GenerateShortUrl(longurl)
  InsertInDB(shorturl, longurl)

  response := SerializeJsonUrl(shorturl)

  fmt.Fprint(w, response)
}
