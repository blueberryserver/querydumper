package main

import (
	"cp"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"merge"
	"net/http"
	"os"
	"path/filepath"
	"process"
	"strings"
	"sync"
	"time"

	"github.com/blueberryserver/bluecore"

	"github.com/julienschmidt/httprouter"
)

// Config ...
type Config struct {
	Host  string `json:"host"`
	User  string `json:"user"`
	Pw    string `json:"pw"`
	Line  string `json:"line"`
	Path  string `json:"path"`
	DbVer string `json:"dbver"`
}

// Database ...
type Database struct {
	Database string
	Tables   []string
}

// Context ...
type Context struct {
	Title     string
	Databases []Database
	SelectDB  string
	Selected  []string
	Files     []string
	DBVer     string
}

func render(w http.ResponseWriter, tmpl string, context Context) {
	tmplList := []string{"templates/base.html",
		fmt.Sprintf("templates/%s.html", tmpl)}
	t, err := template.ParseFiles(tmplList...)
	if err != nil {
		log.Print("template parsing error: ", err)
	}
	err = t.Execute(w, context)
	if err != nil {
		log.Print("template executing error: ", err)
	}
}

func dump(tables []string, config *Config, database string) error {
	log.Printf("%v\r\n", config)
	log.Printf("%v\r\n", tables)

	messageChan := make(chan string)
	var wg sync.WaitGroup
	wg.Add(len(tables))

	var filenames = make([]string, 100)
	for i, table := range tables {

		var filename = config.Path + "\\" + table + ".sql"
		var tempfilename = config.Path + "\\" + table + "_dump.sql"

		go func(tablename string, name string, tempname string) {
			defer wg.Done()
			//log.Println(tablename, name, tempname)
			// mysqmdump.exe 실행 테입블 덤프
			err := process.Execute("./bin/mysqldump.exe", tempname, "--skip-opt", "--net_buffer_length=409600", "--create-options", "--disable-keys",
				"--lock-tables", "--quick", "--set-charset", "--extended-insert", "--single-transaction", "--add-drop-table", "--no-create-db", "-h",
				config.Host, "-u", config.User, config.Pw, database, tablename)
			if err != nil {
				log.Println(err)
				messageChan <- name + " fail"
				return
			}

			if config.Line != "1" {
				// ),( -> )\n( 변경 작업 줄바꿈 처리
				err = process.Execute("./bin/sed.exe", name, "s/),(/),\\\\r\\\\n(/g", tempname)
				if err != nil {
					log.Println(err)
					messageChan <- name + " fail"
					return
				}
			} else {
				err = process.Execute("./bin/sed.exe", name, "", tempname)
				if err != nil {
					log.Println(err)
					messageChan <- name + " fail"
					return
				}
			}

			cp.RM(tempname)

			messageChan <- name + " success"
		}(table, filename, tempfilename)

		filenames[i] = filename
	}

	go func() {
		for msg := range messageChan {
			log.Printf("%s\r\n", msg)
		}
	}()

	wg.Wait()

	Now := time.Now()
	// layout (2006-01-02 15:04:05)
	temp := Now.Format("20060102-150405")
	//fmt.Println("Now: ", temp)
	var resultfilename = config.Path + "/" + database + "_" + temp + ".sql"

	//merge.MERGE(resultfilename, config.Tables...)
	merge.MERGE(resultfilename, filenames...)
	bluecore.ZipFiles(config.Path+"\\"+database+"_"+temp+".zip", filenames)
	for _, file := range filenames {
		cp.RM(file)
	}

	//downloadFile(database+".zip", "http://localhost:8080/"+database+".zip")
	return nil
}

// Index ...
func Index(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	v := r.URL.Query()
	// url path
	path := r.URL.Path
	dbver := strings.Split(path, "/")[1]
	log.Printf("%s\r\n", path)
	//fmt.Println(v["selecttables"])
	//fmt.Println(v["database"])

	//fmt.Println(strings.Split(path, "/"))
	//fmt.Println(path + "/files")
	var files []string
	filepath.Walk("."+path+"/files", func(p string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println(err)
		}
		if false == info.IsDir() {
			_, name := filepath.Split(p)
			files = append(files, name)
		}

		return nil
	})

	if len(v["database"]) == 0 {
		v["database"] = append(v["database"], "doz3_global_"+dbver)
	}

	context := Context{
		Title: "Table Dumper!",
		Databases: []Database{
			{
				Database: "doz3_global_" + dbver,
				Tables:   gtables.Global},
			{
				Database: "doz3_user_" + dbver + "_1",
				Tables:   gtables.User},
			{
				Database: "doz3_user_" + dbver + "_2",
				Tables:   gtables.User},
			{
				Database: "doz3_log_" + dbver,
				Tables:   gtables.Log},
		},
		SelectDB: v["database"][0],
		DBVer:    dbver,
		Selected: v["selecttables"],
		Files:    files,
	}
	render(w, "index", context)
}

// Dump ...
func Dump(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	decoder := json.NewDecoder(r.Body)
	//fmt.Println(r.Body)
	path := r.URL.Path
	dbver := strings.Split(path, "/")[1]
	log.Printf("dbver: %s\r\n", dbver)

	var conf = &Config{}

	for _, c := range gconfs.Configs {
		if c.DbVer == dbver {
			conf = &c
			break
		}
	}
	var dumpData struct {
		Database string   `json:"database"`
		Tables   []string `json:"tables"`
	}

	err := decoder.Decode(&dumpData)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 400)
		return
	}

	//fmt.Println(dumpData)
	err = dump(dumpData.Tables, conf, dumpData.Database)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		// Result info
		type Result struct {
			ResultCode string `json:"resultcode"`
		}
		result := Result{"0"}
		json.NewEncoder(w).Encode(result)

	} else {
		log.Println(err)
		http.Error(w, err.Error(), 400)
		return
	}
}

// DeleteFiles ...
func DeleteFiles(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	path := r.URL.Path
	dbver := strings.Split(path, "/")[1]
	log.Printf("%s\r\n", strings.Split(path, "/"))

	files, _ := filepath.Glob("./" + dbver + "/files/*")
	for _, p := range files {
		err := os.Remove(p)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
	}

	log.Printf("Redirect\r\n")
	http.Redirect(w, r, "/"+dbver, http.StatusMovedPermanently)
}

var gconfig = &Config{}

// DBConfig ...
type DBConfig struct {
	Configs []Config `yaml:"conf"`
}

// Table ...
type Table struct {
	Global []string `yaml:"global"`
	User   []string `yaml:"user"`
	Log    []string `yaml:"log"`
}

// yaml
var gconfs = &DBConfig{}
var gtables = &Table{}

func main() {

	logfile := "log/log_" + time.Now().Format("2006_01_02_15") + ".txt"
	fileLog, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	defer fileLog.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	mutiWriter := io.MultiWriter(fileLog, os.Stdout)
	log.SetOutput(mutiWriter)

	port := flag.String("p", "8080", "p=8080")
	flag.Parse() // 명령줄 옵션의 내용을 각 자료형별

	log.Printf("Start Query Dump !!!(%s) \r\n", *port)

	//config := readConfig()
	// err := bluecore.ReadJSON(gconfig, "conf.json")
	// if err != nil {
	// 	log.Println(err)
	// 	return
	// }
	// log.Println(gconfig)

	err = bluecore.ReadYAML(gconfs, "conf/conf.yaml")
	if err != nil {
		log.Println(err)
		return
	}
	//log.Println(gconfs)

	err = bluecore.ReadYAML(gtables, "conf/tables.yaml")
	if err != nil {
		log.Println(err)
		return
	}
	//log.Println(gtables)

	// start routing
	router := httprouter.New()

	for _, conf := range gconfs.Configs {
		log.Printf("/%s\r\n", conf.DbVer)
		router.GET("/"+conf.DbVer, Index)
		router.POST("/"+conf.DbVer+"/dump", Dump)
		router.POST("/"+conf.DbVer+"/deletefiles", DeleteFiles)
		router.ServeFiles("/"+conf.DbVer+"/files/*filepath", http.Dir(conf.DbVer+"/files"))
	}

	log.Fatal(http.ListenAndServe(":"+*port, router))
}
