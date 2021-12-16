package main

import (
	"bytes"
	"errors"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/syrinsecurity/gologger"
	"github.com/webview/webview"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"
)

var (
	// Create Global File Logger
	logger, errLog = gologger.New("./log.txt", 2000)
)

func main() {
	createWebView()

}

func moveImages(sourceFolder string) string {
	if errLog != nil {
		panic(errLog)
	}
	var wg sync.WaitGroup
	start := time.Now()
	//reader := bufio.NewReader(os.Stdin)
	//logger.WritePrint("Gebe den Pfad des Ordners an: ")
	//var sourceFolder, _ = reader.ReadString('\n')
	//logger.WritePrint("Input: " + sourceFolder)
	folderUrls := extractSubDirectories(sourceFolder)

	var goCount = 0

	for vendor := range folderUrls {
		logger.WritePrint(folderUrls[vendor])
		for _, folder := range folderUrls[vendor] {
			files, _ := ioutil.ReadDir(sourceFolder + "\\" + folder)
			for _, fileInfo := range files {
				fileInfo := fileInfo
				folder := folder
				vendor := vendor
				sourceFolder := sourceFolder
				sourceFileDir := sourceFolder + "\\" + folder + "\\"
				wg.Add(1)
				go func() {
					defer wg.Done()
					saveFilePath := getFilePathToSave(sourceFileDir, vendor, sourceFolder, fileInfo)
					createDirMoveFile(sourceFileDir, saveFilePath, fileInfo.Name())
				}()
				goCount = runtime.NumGoroutine()
			}
		}
	}
	wg.Wait()
	logger.WritePrint("GOROUTINE: ", goCount)
	logger.WritePrint("EXECUTION TIME: ", time.Since(start))
	return "Successfully moved Images"
}

var count = 0

func createWebView() {

	w := webview.New(true)
	defer w.Destroy()

	w.SetSize(600, 600, webview.HintNone)
	w.Init(loadAlpine())
	w.Bind("extractSubDirectories", func(sourceFolder string) string {

		folderUrls := extractSubDirectories(sourceFolder)
		tmpl := template.Must(template.New("html").Parse(`<div>
    {{range $vendor, $folderArray := .}}
        <div>
            <h2>Hersteller: {{$vendor}}</h2>
            {{range $folder := $folderArray}}
                <table>
                    <tr>{{$folder}}</tr>
                </table>
            {{end}}
        </div>
    {{end}}
</div>
			`))
		var html bytes.Buffer
		_ = tmpl.Execute(&html, folderUrls)
		return html.String()
	})

	w.Navigate(`data:text/html,` +
		//language=HTML
		`<!doctype html>
<html lang="de" x-data="{ pathInput: '', tbl : '' }">
<body style="padding: 2rem">
    <h1>JPEG Sorter</h1>
    <p>Hier den Pfad eingeben</p>
    <input style="display:block;width: 30rem; margin-bottom: 1rem" type="text" x-model="pathInput"/>


    <button @click="tbl = await extractSubDirectories(pathInput)">Ordner analysieren</button>
    
    <div x-html=tbl></div>
</body>
</html>`)
	w.Run()
}
func loadAlpine() string {
	file, err := os.Open("alpinejs@3.7.0_dist_cdn.min.js")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	b, err := ioutil.ReadAll(file)
	return string(b)
}

func extractSubDirectories(sourceFolder string) map[string][]string {
	var folderUrls = make(map[string][]string)
	files, err := ioutil.ReadDir(sourceFolder)
	if err != nil {
		log.Fatal(err)
	}
	for _, dir := range files {
		var dirName = dir.Name()
		if strings.Contains(dirName, "Mpx") {
			subString := strings.Split(dirName, "Mpx")
			if subString[1] == "" {
				folderUrls["noname"] = append(folderUrls["noname"], dirName)
			} else if subString[1] != "" {
				vendor := strings.TrimSpace(subString[1])
				folderUrls[vendor] = append(folderUrls[vendor], dirName)
			}
		}
	}
	return folderUrls
}

func getFilePathToSave(sourceFileDir string, vendor string, sourceFolder string, fileInfo os.FileInfo) string {
	var sourceFilePath = sourceFileDir + fileInfo.Name()
	var f, openErr = os.Open(sourceFilePath)
	var saveFilePath string
	if openErr != nil {
		logger.WritePrint("ERROR: could not open file: " + sourceFilePath)
	} else {
		var timeNameError error = nil
		var year, month, exifError = getTimeFromExifData(f)
		if exifError != nil {
			year, month, timeNameError = getTimeFromFilename(fileInfo.Name())
		}
		if timeNameError != nil {
			saveFilePath = sourceFolder + "\\" + vendor + "\\keinDatum\\"
		} else {
			saveFilePath = sourceFolder + "\\" + vendor + "\\" + year + "\\" + month + "\\"
		}
	}
	err := f.Close()
	if err != nil {
		logger.WritePrint("ERROR: Could not close file: ", sourceFilePath)
	}
	return saveFilePath
}

func getTimeFromExifData(file *os.File) (year string, month string, err error) {
	year = ""
	month = ""
	err = nil
	var exifInfo, infoErr = exif.Decode(file)
	if infoErr == nil && exifInfo != nil {
		var dateTime, dateErr = exifInfo.DateTime()
		if dateErr == nil {
			year, month = dateTime.Format("2006"), dateTime.Format("January")
			return
		}
	}
	return year, month, errors.New("could not get time from filename")
}

func getTimeFromFilename(fileName string) (year string, month string, err error) {
	year = ""
	month = ""
	err = nil
	var re = regexp.MustCompile(`(\([0-9]{1,2}\.[0-9]{1,2}.[0-9]{4} [0-9]{1,2}_[0-9]{1,2}_[0-9]{1,2}\))`)
	var date = re.FindString(fileName)
	if date != "" {
		var dateTime, timeError = time.Parse(`(02.01.2006 15_04_05)`, date)
		if timeError != nil {
			return year, month, timeError
		} else {
			year, month = dateTime.Format("2006"), dateTime.Format("January")
		}
	}
	return year, month, errors.New("could not get time from filename")

}

func createDirMoveFile(sourceFileDir string, saveDirPath string, fileName string) {
	err := os.MkdirAll(saveDirPath, os.ModePerm)
	if err != nil {
		logger.WriteString("ERROR: could not create dir: " + saveDirPath)
		logger.WriteString(err)
	} else {
		logger.WritePrint("Move file: " + sourceFileDir + fileName)
		logger.WritePrint("To:        " + saveDirPath + fileName)
		err := os.Rename(sourceFileDir+fileName, saveDirPath+fileName)
		if err != nil {
			logger.WriteString("ERROR: could not move File " + sourceFileDir + fileName + " to " + saveDirPath + fileName)
			logger.WriteString(err)
		}
	}
}
