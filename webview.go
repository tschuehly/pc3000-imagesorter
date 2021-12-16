package main

import (
	"bytes"
	"fmt"
	"github.com/webview/webview"
	"html/template"
)

func createWebView() {

	webView := webview.New(true)
	defer webView.Destroy()

	webView.SetSize(1000, 1080, webview.HintNone)
	webView.Init(loadAlpine())

	var folderUrls map[string][]FolderDetails

	go func() {
		for true {
			//prog := <-progressChan
			mux.RLock()
			js := fmt.Sprintf(`document.getElementById("counter").innerHTML = '.%d.';`, pCount)
			//js := fmt.Sprintf(`console.log(%d)`, prog)
			fmt.Println(js)
			webView.Dispatch(func() { webView.Eval(js) })
			mux.RUnlock()
		}
	}()

	webView.Bind("extractSubDirectories", func(sourceFolder string) string {

		folderUrls = extractSubDirectories(sourceFolder)
		tmpl := template.Must(template.New("html").Parse(
			// language=GoTemplate
			`<div>
    {{range $vendor, $folderArray := .}}
        <div>
            <h3>Hersteller: {{$vendor}}</h2>
            {{range $folder := $folderArray}}
                <ul>
                    <li>{{$folder}}</li>
                </ul>
            {{end}}
        </div>
    {{end}}
</div>`))
		var html bytes.Buffer
		_ = tmpl.Execute(&html, folderUrls)
		return html.String()
	})

	webView.Bind("moveImages", func(sourceFolder string) string {
		go func() {
			moveImages(folderUrls, sourceFolder)
		}()
		return "done"
	})

	webView.Bind("count", func() {
		go func() {
			counter = counter + 1
			progressChan <- counter
		}()
	})
	webView.Navigate(`data:text/html,` +
		//language=HTML
		`<!doctype html>
<html lang="de" x-data="{ pathInput: '', tbl : '', success : '', counter : 0}">
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
				@click="success = await moveImages(pathInput); tbl = ''"
				style="padding: 0.7rem 2.5rem;font-size: 1.2rem;font-weight: 700;"
			>Bilder sortieren</button>
		</div>
		<button @click="await count()">Count +</button>
		<h2 id="counter" style="margin-bottom: 1rem;word-break: break-all;" x-text="counter"></div>
		<div x-show="success != ''">
			<h2 x-text="success"></h2>
		</div>
    </body>
</html>`)
	webView.Run()

}
