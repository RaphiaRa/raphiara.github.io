package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	goldhtml "github.com/yuin/goldmark/renderer/html"
)

type ArticleInfo struct {
	CreatedOn time.Time
	Title     string
	Link      string
}

type Week struct {
	Count int
}

type Article struct {
	HtmlHead      string
	ContentHeader string
	Content       string
	Footer        string
}

type Index struct {
	HtmlHead      string
	ContentHeader string
	Footer        string
	Articles      []ArticleInfo
	Weeks         []Week
	RecentEntries int
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func loadHtmlTemplate(name string, m map[string]string) {
	path := "../template/" + name + ".html"
	dat, err := os.ReadFile(path)
	check(err)
	m[name] = string(dat)
}

func htmlFromMarkdown(name string, m goldmark.Markdown) string {
	path := "../markdown/" + name + ".md"
	md, err := os.ReadFile(path)
	check(err)
	var buf bytes.Buffer
	if err := m.Convert(md, &buf); err != nil {
		check(err)
	}
	return string(buf.Bytes())
}

func htmlFromArticleMarkdown(name string, m goldmark.Markdown, ctx parser.Context) string {
	path := "../markdown/articles/" + name
	md, err := os.ReadFile(path)
	check(err)
	var buf bytes.Buffer
	if err := m.Convert(md, &buf, parser.WithContext(ctx)); err != nil {
		check(err)
	}
	return string(buf.Bytes())
}

func loadTemplates() map[string]string {
	m := make(map[string]string)
	loadHtmlTemplate("site", m)
	loadHtmlTemplate("footer", m)
	loadHtmlTemplate("head", m)
	loadHtmlTemplate("article_head", m)
	loadHtmlTemplate("header", m)
	loadHtmlTemplate("index", m)
	loadHtmlTemplate("article", m)
	return m
}

func createWeeks(articleInfo []ArticleInfo) (int, []Week) {
	// Always show the last 52 weeks
	var weeks []Week
	for i := 0; i < 53; i++ {
		weeks = append(weeks, Week{Count: 0})
	}
	// Count the number of articles in each week
	// of the last 52 weeks and store the number
	// in the corresponding week struct
	var entries int
	start := time.Now().AddDate(-1, 0, 0)
	start_year, start_week := start.ISOWeek()
	for _, article := range articleInfo {
		if article.CreatedOn.After(start) {
			year, week := article.CreatedOn.ISOWeek()
			index := (year-start_year)*52 + week - start_week
			// log index week and start week
			fmt.Printf("Index: %d, Week: %d, Start Week: %d\n", index, week, start_week)
			weeks[index-1].Count++
			entries++
		}
	}
	return entries, weeks
}

func buildIndex(t map[string]string, articleInfo []ArticleInfo) {
	var index Index
	index.ContentHeader = t["header"]
	index.Footer = t["footer"]
	index.HtmlHead = t["head"]
	index.Articles = articleInfo
	entries, weeks := createWeeks(articleInfo)
	index.RecentEntries = entries
	index.Weeks = weeks
	tmpl, err := template.New("index").Parse(t["index"])
	check(err)
	f, err := os.Create("../out/index.html")
	check(err)
	err = tmpl.Execute(f, index)
	check(err)
}

func buildSite(t map[string]string, m goldmark.Markdown, name string) {
	var site Article
	site.ContentHeader = t["header"]
	site.Footer = t["footer"]
	site.HtmlHead = t["head"]
	site.Content = htmlFromMarkdown(name, m)
	tmpl, err := template.New(name).Parse(t["site"])
	check(err)
	f, err := os.Create("../out/" + name + ".html")
	check(err)
	err = tmpl.Execute(f, site)
	check(err)
}

func buildSingleArticle(file string, t map[string]string, m goldmark.Markdown, isDebug bool) (ArticleInfo, bool) {
	var articleInfo ArticleInfo
	var article Article
	article.ContentHeader = t["header"]
	article.Footer = t["footer"]
	article.HtmlHead = t["article_head"]
	context := parser.NewContext()
	article.Content = htmlFromArticleMarkdown(file, m, context)
	tmpl, err := template.New(file).Parse(t["article"])
	check(err)
	metaData := meta.Get(context)
	draft := metaData["draft"]
	if isDebug || draft == false {
		date := fmt.Sprint(metaData["date"])
		title := fmt.Sprint(metaData["title"])
		path := "/articles/" + file[:len(file)-len(filepath.Ext(file))] + ".html"
		layout := "2006-01-02T15:04:05Z07:00"
		t, err := time.Parse(layout, date)
		check(err)
		articleInfo.CreatedOn = t
		articleInfo.Link = path
		articleInfo.Title = title
		f, err := os.Create("../out" + path)
		check(err)
		err = tmpl.Execute(f, article)
		check(err)
		return articleInfo, true
	}
	return articleInfo, false
}

func buildArticles(t map[string]string, m goldmark.Markdown, isDebug bool) []ArticleInfo {
	var articleInfos []ArticleInfo
	path := "../markdown/articles/"
	os.Mkdir("../out/articles", 0766)
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		check(err)
		if info.Mode().IsRegular() {
			articleInfo, ok := buildSingleArticle(info.Name(), t, m, isDebug)
			if ok {
				articleInfos = append(articleInfos, articleInfo)
			}
		}
		return nil
	})
	return articleInfos
}

func main() {
	isDebug := false
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "debug" {
			isDebug = true
		}
	}

	os.Mkdir("../out", 0766)
	markdown := goldmark.New(
		goldmark.WithRendererOptions(
			goldhtml.WithUnsafe(),
		),
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithFormatOptions(
					html.WithLineNumbers(true),
					html.WrapLongLines(true),
				),
			),
			meta.Meta,
		),
	)
	templates := loadTemplates()
	articleInfos := buildArticles(templates, markdown, isDebug)
	fmt.Printf("Number of articles created: %d\n", len(articleInfos))
	buildIndex(templates, articleInfos)
	//buildSite(templates, markdown, "cv")
	buildSite(templates, markdown, "about")
}
