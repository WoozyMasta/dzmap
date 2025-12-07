package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"text/template"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/svg"
)

type PageData struct {
	CSS string
	JS  string
	SVG string
}

func main() {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/javascript", js.Minify)
	m.AddFunc("image/svg+xml", svg.Minify)

	cssRaw, err := os.ReadFile("assets/style.css")
	if err != nil {
		log.Fatal("error read CSS:", err)
	}
	cssMin, err := m.String("text/css", string(cssRaw))
	if err != nil {
		log.Fatal("error minify CSS:", err)
	}

	jsRaw, err := os.ReadFile("assets/script.js")
	if err != nil {
		log.Fatal("error read JS:", err)
	}
	jsMin, err := m.String("text/javascript", string(jsRaw))
	if err != nil {
		log.Fatal("error minify JS:", err)
	}

	svgRaw, err := os.ReadFile("assets/steam.svg")
	if err != nil {
		log.Fatal("error read CSV:", err)
	}
	svgMin, err := m.String("image/svg+xml", string(svgRaw))
	if err != nil {
		log.Fatal("error minify CSV:", err)
	}

	htmlRaw, err := os.ReadFile("assets/index.html.tpl")
	if err != nil {
		log.Fatal("error read HTML:", err)
	}

	tmpl, err := template.New("index").Parse(string(htmlRaw))
	if err != nil {
		log.Fatal("error read template:", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, PageData{
		CSS: cssMin,
		JS:  jsMin,
		SVG: svgMin,
	})
	if err != nil {
		log.Fatal("error parse template:", err)
	}

	finalHTML, err := m.String("text/html", buf.String())
	if err != nil {
		log.Fatal("error minify HTML:", err)
	}

	err = os.WriteFile("assets/index.html", []byte(finalHTML), 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("minify done")
}
