package main

import (
	"cp"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"merge"
	"net/http"
	"os"
	"path/filepath"
	"process"
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
	log.Println(config)

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
			log.Println(msg)
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
func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	v := r.URL.Query()
	//fmt.Println(v["selecttables"])
	//fmt.Println(v["database"])

	var files []string
	filepath.Walk("./files", func(path string, info os.FileInfo, err error) error {
		if false == info.IsDir() {
			_, name := filepath.Split(path)
			files = append(files, name)
		}
		return nil
	})

	if len(v["database"]) == 0 {
		v["database"] = append(v["database"], "doz3_global_"+gconfig.DbVer)
	}

	context := Context{
		Title: "Table Dumper!",
		Databases: []Database{
			{
				Database: "doz3_global_" + gconfig.DbVer,
				Tables: []string{
					"iu_achiv_table",
					"iu_boss_default",
					"iu_boss_info",
					"iu_common_systems",
					"iu_daily_reward_data",
					"iu_default_skill",
					"iu_drop_box",
					"iu_dropgroup",
					"iu_event_daily_reward_data",
					"iu_event_dungeon_table",
					"iu_gacha_data",
					"iu_gather_event_data",
					"iu_global_settings",
					"iu_item_data",
					"iu_item_option_type",
					"iu_product",
					"iu_product_bonus",
					"iu_product_iap",
					"iu_product_ingame",
					"iu_product_mystery_shop",
					"iu_web_event_mission",
					"iu_web_event_page",
					"iu_web_event_select",
					"iu_web_event_select_reward",
				}},
			{
				Database: "doz3_user_" + gconfig.DbVer + "_1",
				Tables: []string{
					"iu_achiv",
					"iu_artifact",
					"iu_char_info",
					"iu_daily_reward",
					"iu_guild",
					"iu_guild_member",
					"iu_iap_log",
					"iu_inven_class",
					"iu_inven_option",
					"iu_inven_stack",
					"iu_mystery_shop",
					"iu_post",
					"iu_product_count",
					"iu_rune",
					"iu_seq_quest",
					"iu_skill_class",
					"iu_smithy",
					"iu_special_box_count",
					"iu_special_products",
					"iu_stage_single",
					"iu_user_info",
					"iu_web_event_bingo_progress",
					"iu_web_event_roulette_progress",
					"iu_web_event_select_progress",
				}},
			{
				Database: "doz3_user_" + gconfig.DbVer + "_2",
				Tables: []string{
					"iu_achiv",
					"iu_artifact",
					"iu_char_info",
					"iu_daily_reward",
					"iu_guild",
					"iu_guild_member",
					"iu_iap_log",
					"iu_inven_class",
					"iu_inven_option",
					"iu_inven_stack",
					"iu_mystery_shop",
					"iu_post",
					"iu_product_count",
					"iu_rune",
					"iu_seq_quest",
					"iu_skill_class",
					"iu_smithy",
					"iu_special_box_count",
					"iu_special_products",
					"iu_stage_single",
					"iu_user_info",
					"iu_web_event_bingo_progress",
					"iu_web_event_roulette_progress",
					"iu_web_event_select_progress",
				}},
			{
				Database: "doz3_log_" + gconfig.DbVer,
				Tables: []string{
					"achiv_reward_logs",
					"action_logs",
					"bossraid_logs",
					"char_equip_snapshot_logs",
					"char_levelup_logs",
					"char_simple_snapshot_logs",
					"event_bingo_progress_logs",
					"event_quest_reward_logs",
					"event_roulette_progress_logs",
					"event_select_progress_logs",
					"hack_logs",
					"iap_buy_logs",
					"iap_error_logs",
					"immortalraid_logs",
					"inventory_expand_logs",
					"item_create_non_stackable_logs",
					"item_delete_non_stackable_logs",
					"item_dyeing_logs",
					"item_option_change_logs",
					"item_option_create_logs",
					"item_option_delete_logs",
					"item_stackable_logs",
					"log_character_connect",
					"log_connect",
					"post_receive_logs",
					"post_send_logs",
					"product_buy_logs",
					"product_mystery_shop_logs",
					"pvp_ai_logs",
					"pvp_logs",
					"skill_logs",
					"stage_logs",
					"vip_levelup_logs",
				}},
		},
		SelectDB: v["database"][0],
		Selected: v["selecttables"],
		Files:    files,
	}
	render(w, "index", context)
}

// Dump ...
func Dump(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	decoder := json.NewDecoder(r.Body)

	var dumpConfig struct {
		Database string   `json:"database"`
		Tables   []string `json:"tables"`
	}

	err := decoder.Decode(&dumpConfig)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 400)
		return
	}

	err = dump(dumpConfig.Tables, gconfig, dumpConfig.Database)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Result info
	type Result struct {
		ResultCode string `json:"resultcode"`
	}
	result := Result{"0"}
	json.NewEncoder(w).Encode(result)
}

// Select ...
func Select(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	decoder := json.NewDecoder(r.Body)
	//fmt.Println(r.Body)

	var dumpData struct {
		Database string   `json:"database"`
		Tables   []string `json:"tables"`
	}

	err := decoder.Decode(&dumpData)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	//fmt.Println(dumpData)
	err = dump(dumpData.Tables, gconfig, dumpData.Database)
	if err == nil {
		log.Println("Redirect")
		http.Redirect(w, r, "/"+gconfig.DbVer, http.StatusMovedPermanently)
	} else {
		log.Println(err)
		http.Error(w, err.Error(), 400)
		return
	}
}

// DeleteFiles
func DeleteFiles(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	files, _ := filepath.Glob("./files/*")
	for _, path := range files {
		err := os.Remove(path)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
	}

	log.Println("Redirect")
	http.Redirect(w, r, "/"+gconfig.DbVer, http.StatusMovedPermanently)
}

var gconfig = &Config{}

func main() {

	port := flag.String("p", "8080", "p=8080")
	flag.Parse() // 명령줄 옵션의 내용을 각 자료형별

	log.Println("Start Query Dump !!!" + *port)

	//config := readConfig()
	err := bluecore.ReadJSON(gconfig, "conf.json")
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(gconfig)

	// start routing
	router := httprouter.New()
	router.GET("/"+gconfig.DbVer, Index)
	router.POST("/"+gconfig.DbVer+"/dump", Dump)
	router.POST("/"+gconfig.DbVer+"/select", Select)
	router.POST("/"+gconfig.DbVer+"/deletefiles", DeleteFiles)
	router.ServeFiles("/"+gconfig.DbVer+"/files/*filepath", http.Dir("files"))

	log.Fatal(http.ListenAndServe(":"+*port, router))
}
