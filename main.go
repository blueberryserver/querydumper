package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
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

	// start routing
	router := httprouter.New()
	router.GET("/", EmptyIndex)

	for _, conf := range gconfs.Configs {
		log.Printf("/%s\r\n", conf.DbVer)
		router.GET("/"+conf.DbVer+"/dump", DumpIndex)
		router.POST("/"+conf.DbVer+"/dump/exec", DumpExc)
		router.POST("/"+conf.DbVer+"/dump/delete", DumpDelete)
		router.ServeFiles("/"+conf.DbVer+"/dump/files/*filepath", http.Dir(conf.DbVer+"/dump/files"))
	}

	// for _, conf := range gconfs.Configs {
	// 	log.Printf("/%s\r\n", conf.DbVer)
	// 	router.GET("/"+conf.DbVer+"/copy", CopyIndex)

	// }

	log.Fatal(http.ListenAndServe(":"+*port, router))
}
