package main

import (
	"errors"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/syrinsecurity/gologger"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	// Create Global File Logger
	logger, errLog = gologger.New("./log.txt", 2000)
	progressChan   = make(chan int, 1)
	mux            = &sync.RWMutex{}
)

type FolderDetails struct {
	path      string
	fileCount int
}

func main() {
	if errLog != nil {
		panic(errLog)
	}
	createWebView()
}

var counter = 0
var pCount = 0

func moveImages(folderUrls map[string][]FolderDetails, sourceFolder string) string {
	var wg sync.WaitGroup
	start := time.Now()
	var goCount = 0
	for vendor := range folderUrls {
		logger.WritePrint(folderUrls[vendor])
		for _, folderDetails := range folderUrls[vendor] {
			files, _ := ioutil.ReadDir(filepath.Join(sourceFolder, folderDetails.path))
			for _, fileInfo := range files {
				fileInfo := fileInfo
				folderDetails := folderDetails
				vendor := vendor
				sourceFolder := sourceFolder
				sourceFileDir := filepath.Join(sourceFolder, folderDetails.path)
				wg.Add(1)
				go func(progressCount *int, mux *sync.RWMutex) {
					mux.Lock()
					*progressCount = *progressCount + 1
					mux.Unlock()
					//progressChan <- *progressCount
					defer wg.Done()
					time.Sleep(200 * time.Millisecond)
					saveFilePath := getFilePathToSave(sourceFileDir, vendor, sourceFolder, fileInfo)
					createDirMoveFile(sourceFileDir, saveFilePath, fileInfo.Name())
					mux.Lock()
					*progressCount = *progressCount - 1
					mux.Unlock()
					//progressChan <- *progressCount
				}(&pCount, mux)
				goCount = runtime.NumGoroutine()
			}
		}
	}
	wg.Wait()
	logger.WritePrint("GOROUTINE: ", goCount)
	logger.WritePrint("EXECUTION TIME: ", time.Since(start))
	pCount = 0
	return "Successfully moved Images"
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

func extractSubDirectories(sourceFolder string) map[string][]FolderDetails {
	var folderUrls = make(map[string][]FolderDetails)
	directories, err := ioutil.ReadDir(sourceFolder)
	if err != nil {
		log.Fatal(err)
	}
	for _, dir := range directories {
		var dirName = dir.Name()
		var dirEntryArray, err = os.ReadDir(filepath.Join(sourceFolder, dirName))
		if err != nil {
			logger.WritePrint(err)
			os.Exit(1)
		}
		if strings.Contains(dirName, "Mpx") {
			subString := strings.Split(dirName, "Mpx")
			if subString[1] == "" {
				folderUrls["noname"] = append(folderUrls["noname"], FolderDetails{dirName, len(dirEntryArray)})
			} else if subString[1] != "" {
				vendor := strings.TrimSpace(subString[1])
				folderUrls[vendor] = append(folderUrls[vendor], FolderDetails{dirName, len(dirEntryArray)})
			}
		}
	}
	return folderUrls
}

func getFilePathToSave(sourceFileDir string, vendor string, sourceFolder string, fileInfo os.FileInfo) string {
	var sourceFilePath = filepath.Join(sourceFileDir, fileInfo.Name())
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
			saveFilePath = filepath.Join(sourceFolder, vendor, "keinDatum")
		} else {
			saveFilePath = filepath.Join(sourceFolder, vendor, year, month)
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
		logger.WritePrint("Move file: " + filepath.Join(sourceFileDir+fileName))
		logger.WritePrint("To:        " + filepath.Join(saveDirPath+fileName))
		err := os.Rename(filepath.Join(sourceFileDir+fileName), filepath.Join(saveDirPath+fileName))
		if err != nil {
			logger.WriteString("ERROR: could not move File " + sourceFileDir + fileName + " to " + saveDirPath + fileName)
			logger.WriteString(err)
		}
	}
}
