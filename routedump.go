package main

import (
	"cp"
	"encoding/json"
	"fmt"
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

func dump(tables []string, config *Config, database string) error {
	//log.Printf("%v\r\n", config)
	//log.Printf("%v\r\n", tables)

	messageChan := make(chan string)
	var wg sync.WaitGroup
	wg.Add(len(tables))

	var filenames = make([]string, 100)
	for i, table := range tables {

		var filename = config.Path + "/" + table + ".sql"
		var tempfilename = config.Path + "/" + table + "_dump.sql"

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

			messageChan <- tablename + " success"
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

// DumpIndex ...
func DumpIndex(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	v := r.URL.Query()
	// url path
	path := r.URL.Path
	dbver := strings.Split(path, "/")[1]
	//log.Printf("%s/dump/index\r\n", path)
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

	if dbver == "trunk" {
		context.Databases[1].Database = "doz3_user_trunk1"
		context.Databases[2].Database = "doz3_user_trunk2"
	}

	render(w, "dumpindex", context)
}

// EmptyIndex ..
func EmptyIndex(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	render(w, "index", Context{})
}

// DumpExc ...
func DumpExc(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	decoder := json.NewDecoder(r.Body)
	//fmt.Println(r.Body)
	path := r.URL.Path
	dbver := strings.Split(path, "/")[1]
	log.Printf("/%s/dump/exec\r\n", dbver)

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

	fmt.Println(dumpData)
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

// DumpDelete ...
func DumpDelete(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	path := r.URL.Path
	dbver := strings.Split(path, "/")[1]
	log.Printf("/%s/dump/delete\r\n", dbver)

	files, _ := filepath.Glob("./" + dbver + "/dump/files/*")
	for _, p := range files {
		err := os.Remove(p)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
	}

	//log.Printf("Redirect\r\n")
	//http.Redirect(w, r, "/"+dbver, http.StatusMovedPermanently)
}
