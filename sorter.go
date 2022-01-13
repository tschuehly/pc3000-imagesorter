package main

import (
	"errors"
	"fmt"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/syrinsecurity/gologger"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	// Create Global File Logger
	logger, errLog = gologger.New("./log.txt", 2000)
	mux            = &sync.RWMutex{}
	pCount         = 0
)

type FolderDetails struct {
	Path      string
	FileCount int
}

type WorkerImageInfo struct {
	FileInfo      fs.FileInfo
	FolderDetails FolderDetails
	Vendor        string
	SourceFolder  string
	SourceFileDir string
}

func main() {
	if errLog != nil {
		panic(errLog)
	}
	openWebview()
}

func moveImages(folderUrls map[string][]FolderDetails, sourceFolder string) string {
	var waitGroup sync.WaitGroup
	start := time.Now()
	imageInfoJobs := make(chan WorkerImageInfo, 1000)
	for w := 1; w <= 10000; w++ {
		waitGroup.Add(1)
		go imageWorker(w, &waitGroup, mux, &pCount, imageInfoJobs)
	}
	for vendor := range folderUrls {
		logger.WritePrint(folderUrls[vendor])
		for _, folderDetails := range folderUrls[vendor] {
			files, _ := ioutil.ReadDir(filepath.Join(sourceFolder, folderDetails.Path))
			for _, fileInfo := range files {
				sourceFolder := sourceFolder
				workerImageInfo := WorkerImageInfo{
					FileInfo:      fileInfo,
					FolderDetails: folderDetails,
					Vendor:        vendor,
					SourceFolder:  sourceFolder,
					SourceFileDir: filepath.Join(sourceFolder, folderDetails.Path),
				}
				imageInfoJobs <- workerImageInfo
			}
		}
	}
	close(imageInfoJobs)
	waitGroup.Wait()
	executionTime := time.Since(start).String()
	logger.WritePrint("EXECUTION TIME: " + executionTime)
	pCount = 0
	return executionTime
}

func imageWorker(id int, waitGroup *sync.WaitGroup, mux *sync.RWMutex, progressCount *int, imageInfoJob <-chan WorkerImageInfo) {
	defer waitGroup.Done()
	for info := range imageInfoJob {
		fmt.Println("worker", id, "started  job", info.FileInfo.Name())
		mux.Lock()
		*progressCount = *progressCount + 1
		mux.Unlock()
		saveFilePath := getFilePathToSave(info.SourceFileDir, info.Vendor, info.SourceFolder, info.FileInfo)
		createDirMoveFile(info.SourceFileDir, saveFilePath, info.FileInfo.Name())
		mux.Lock()
		*progressCount = *progressCount - 1
		mux.Unlock()
		fmt.Println("worker", id, "ended  job", info.FileInfo.Name())
	}
}

func extractSubDirectories(sourceFolder string) map[string][]FolderDetails {
	logger.WritePrint("extractSubdirectories")
	var folderUrls = make(map[string][]FolderDetails)
	directories, err := ioutil.ReadDir(sourceFolder)
	if err != nil {
		logger.WritePrint("ERROR: could not open sourcefolder")
		return nil
	}
	logger.WritePrint("extractSubdirectories")
	for _, dir := range directories {
		var dirName = dir.Name()
		var dirEntryArray, err = os.ReadDir(filepath.Join(sourceFolder, dirName))
		if err != nil {
			logger.WritePrint("ERROR: " + err.Error())
		}
		if strings.Contains(dirName, "Mpx") {
			subString := strings.Split(dirName, "Mpx")
			if subString[1] == "" {
				folderUrls["noname"] = append(folderUrls["noname"], FolderDetails{dirName, len(dirEntryArray)})
				logger.WritePrint("Added noname folder: " + dirName)
			} else if subString[1] != "" {
				vendor := strings.TrimSpace(subString[1])
				folderUrls[vendor] = append(folderUrls[vendor], FolderDetails{dirName, len(dirEntryArray)})
				logger.WritePrint("Added " + vendor + " folder: " + dirName)
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
		err := os.Rename(filepath.Join(sourceFileDir, fileName), filepath.Join(saveDirPath, fileName))
		if err != nil {
			logger.WriteString("ERROR: could not move File " + sourceFileDir + fileName + " to " + saveDirPath + fileName)
			logger.WriteString(err)
		}
	}
}
