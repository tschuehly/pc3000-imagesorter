package main

import (
	"bytes"
	"fmt"
	"github.com/webview/webview"
	"html/template"
	"io/ioutil"
	"log"
	"os"
)

var (
	webView webview.WebView
)

func listenForProcessCount() {
	for true {
		var previousCount = 0
		mux.RLock()
		if previousCount != pCount {
			previousCount = pCount
			js := fmt.Sprintf(`document.getElementById("counter").innerHTML = '.%d.';`, pCount)
			webView.Dispatch(func() { webView.Eval(js) })
		}
		mux.RUnlock()
	}
}

func openWebview() {
	webView = webview.New(true)
	defer webView.Destroy()

	webView.SetSize(1000, 1080, webview.HintNone)
	webView.Init(loadAlpine())

	bindGoFunctions()
	go listenForProcessCount()

	webView.Navigate(`data:text/html,` +
		//language=HTML
		`<!doctype html>
<html lang="de" x-data="{ pathInput: '', tbl : '', counter : 0, processing : false}"
	style="background-color: rgb(20,20,20); color: rgb(200,200,200)">
	<head>
		<style>
		</style>
	</head>
    <body style="padding: 2rem">
		<h1>JPEG Sorter</h1>

		<p>Hier den Pfad eingeben</p>
		<input style="display:block;width: 30rem; margin-bottom: 1rem" type="text" x-model="pathInput"/>
		
		<button @click="tbl = await extractSubDirectories(pathInput)">Ordner analysieren</button>
		<div x-html=tbl></div>

		<div x-show="tbl != ''" style="margin-top: 2rem;">
			<button 
				@click="processing = await moveImages(pathInput); tbl = ''"
				style="padding: 0.7rem 2.5rem;font-size: 1.2rem;font-weight: 700;"
			>Bilder sortieren</button>
		</div>
		<div x-show="processing == true">
            <h1 id="counter" style="margin-bottom: 1rem" x-text="counter"></div>
        </div>
        <div>
            <h1 id="executionTime"></h1>
        </div>
    </body>
</html>`)
	webView.Run()

}

func bindGoFunctions() {
	var folderUrls map[string][]FolderDetails

	webView.Bind("extractSubDirectories", func(sourceFolder string) string {

		folderUrls = extractSubDirectories(sourceFolder)
		tmpl := template.Must(template.New("html").Parse(
			// language=GoTemplate
			`<div>
    {{range $vendor, $folderDetailsArray := .}}
        <div>
            <h3>Hersteller: {{$vendor}}</h2>
            {{range $folderDetails := $folderDetailsArray}}
                <ul>
                    <li>Ordnername: {{ .Path }} Dateianzahl: {{ .FileCount }}</li>
                </ul>
            {{end}}
        </div>
    {{end}}
</div>`))
		var html bytes.Buffer
		err := tmpl.Execute(&html, folderUrls)
		if err != nil {
			logger.WritePrint("ERROR: " + err.Error())
		}
		return html.String()
	})

	webView.Bind("moveImages", func(sourceFolder string) bool {
		go func() {
			executionTime := moveImages(folderUrls, sourceFolder)
			js := fmt.Sprintf(`document.getElementById("executionTime").innerHTML = 'Ausführungszeit: %s';`, executionTime)
			webView.Dispatch(func() { webView.Eval(js) })
		}()
		return true
	})
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
