package main

import (
	"encoding/json"
	"strings"
	"os"
	"log"
	"flag"
	"bytes"
	"mime/multipart"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"github.com/fsnotify/fsnotify"
)

type ConfigItem struct {
	Receiver string `json:"receiver"`
	Root string `json:"root"`
	SrcPath string `json:"path"`
}

var reciever, root, srcPath string
var configPath, app string

func init() {
	flag.StringVar(&app, "app", "", "`app` to watch")
	flag.StringVar(&app, "a", "", "`path` to watch, short for -app")

	flag.StringVar(&configPath, "conf", "./conf.json", "path of json `config` file")
	flag.StringVar(&configPath, "c", "./conf.json", "path of json `config` file, short for -config")
}

var basePath string

func main() {
	flag.Parse()

	if app == "" {
		flag.PrintDefaults()
		return
	}

	conf := readConfig()
	reciever = conf.Receiver
	root = conf.Root
	
	srcPath = conf.SrcPath
	basePath, _ = filepath.Abs(srcPath)

	watcher, err := fsnotify.NewWatcher()
	defer watcher.Close()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("[info] Start watching path", basePath)

	err = firstSync(basePath)
	if err != nil {
		log.Fatal(err)
	}
	
	err = watcher.Add(srcPath)
	childErr := filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() { 
				return filepath.SkipDir 
			} else { 
				return nil
			}
		}

		err = watcher.Add(path)
		if err != nil {
			log.Fatal(err)
		}
		return nil
	})
	if err != nil || childErr != nil {
		log.Fatal(err, childErr)
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					absPath, _ := filepath.Abs(event.Name)
					fileSync(absPath)
				}
			case err := <-watcher.Errors:
				log.Println(err)
			}
		}
	}()

	// hang the program
	done := make(chan bool)
	<-done
}

func firstSync(path string) error {
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() { 
				return filepath.SkipDir 
			} else { 
				return nil
			}
		}

		fileSync(path)
		return err
	})
	return err
}

func fileSync(fname string) error {
	// Dir or prefix with .
	info, err := os.Stat(fname)
	if err != nil || (info.IsDir() || strings.HasPrefix(info.Name(), ".")) {
		return err
	}

	// skip the temp files from JetBrain IDE.
	if strings.HasSuffix(info.Name(), "___jb_tmp___") || strings.HasSuffix(info.Name(), "___jb_old___") {
		return nil
	}

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	relPath, err := filepath.Rel(basePath, fname)
	if err != nil {
        log.Println(err)
        return err
	}

	err = bodyWriter.WriteField("to", root + relPath)
	log.Printf("[info] %s >> %s\n", relPath, root + relPath)
	if err != nil {
        log.Println("error writing to buffer")
        return err
	}

    // 关键的一步操作
	fileWriter, err := bodyWriter.CreateFormFile("file", fname)
    if err != nil {
        log.Println("error writing to buffer")
        return err
	}

    // 打开文件句柄操作
    fh, err := os.Open(fname)
    if err != nil {
        log.Println("error opening file")
        return err
    }
    defer fh.Close()

    //iocopy
    _, err = io.Copy(fileWriter, fh)
    if err != nil {
        return err
    }

    contentType := bodyWriter.FormDataContentType()
    bodyWriter.Close()

    resp, err := http.Post(reciever, contentType, bodyBuf)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}

func readConfig() ConfigItem {
	cf, err := os.Open(configPath)
	if err != nil {
		log.Fatal(err)
	}
	defer cf.Close()

	buf, err := ioutil.ReadAll(cf)
	if err != nil {
		log.Fatal(err)
	}
	
	var configs map[string]ConfigItem
	_ = json.Unmarshal(buf, &configs)
	return configs[app]
}
