package main

import (
	"fmt"
	"github.com/blevesearch/bleve/search"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	index *Search
}

func NewServer(search *Search) *Server {
	return &Server{index: search}
}

func (s *Server) Start(host string, port int) {
	router := http.NewServeMux() // here you could also go with third party packages to create a router
	// Register your routes
	router.HandleFunc("/download/", s.download)
	router.HandleFunc("/", s.search)

	listenAddr := host + ":" + strconv.Itoa(port)
	server := &http.Server{
		Addr:         listenAddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	fmt.Printf("starting search server at %s\n", listenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Could not listen on %s: %v\n", listenAddr, err)
		os.Exit(10)
	}
	fmt.Println("Server stopped")
}

func (s *Server) download(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(r.URL.Path, "/")
	if len(segments) != 3 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	name := segments[2]
	fname := s.index.FilenameForID(name)
	file, err := os.Open(fname)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer file.Close()
	io.Copy(w, file)
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	viewModel := &SearchModel{}
	viewModel.Query = r.URL.Query().Get("q")
	if len(viewModel.Query) > 0 {
		res := s.index.Query(viewModel.Query)
		if res != nil {
			viewModel.Count = int(res.Total)
			viewModel.Max = len(res.Hits)
			pageCount := 0
			for _, doc := range res.Hits {
				viewModel.Entries = append(viewModel.Entries, asEntry(doc))
				pageCount++
			}
		}
	}

	err := tpl.Execute(w, viewModel)
	if err != nil {
		fmt.Printf("failed to apply tpl: %v\n", err)
	}

}

func asEntry(doc *search.DocumentMatch) *Entry {
	body := fmt.Sprintf("%v", doc.Fields["Body"])
	if len(body) > 200 {
		body = body[:200] + "..."
	}
	body = strings.ReplaceAll(body, "\n", "")
	sizeBytes := fmt.Sprintf("%v", doc.Fields["Size"])
	sizeNum, _ := strconv.ParseInt(sizeBytes, 10, 32)
	size := strconv.Itoa(int(sizeNum)) + " Byte"
	if sizeNum > 1024 {
		size = strconv.Itoa(int(sizeNum/1024)) + " KiB"
		sizeNum /= 1024
	}
	if sizeNum > 1024 {
		size = strconv.Itoa(int(sizeNum/1024)) + " MiB"
		sizeNum /= 1024
	}

	attachmentsStr := fmt.Sprintf("%v", doc.Fields["AttachmentCount"])
	AttachmentCountNum, _ := strconv.ParseInt(attachmentsStr, 10, 32)
	return &Entry{
		Title:        fmt.Sprintf("%v", doc.Fields["Subject"]),
		Body:         body,
		DownloadLink: "/download/" + doc.ID,
		Size:         size,
		Attachments:  int(AttachmentCountNum),
	}
}

type SearchModel struct {
	Query   string
	Max     int
	Count   int
	Entries []*Entry
}

type Entry struct {
	Title        string
	Body         string
	DownloadLink string
	Size         string
	Attachments  int
}

const page = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>imaparc search</title>
    <style>
        body {
            font-family: "Roboto Thin", sans-serif;
        }

        .content {
            margin-left: auto;
            margin-right: auto;
            max-width: 900px;

        }

        .header {
            text-align: center;
        }

        .results p {
            margin-top: 0px;
            margin-bottom: 4px;
        }

        .results span {
            font-size: small;
        }

        .results a {
            font-size: small;
        }

        h2 {
            font-size: large;
        }

        .searchfield {
            border-radius: 25px;
            padding: 4px;
            border: 1px solid #000000;
            width: 100%;
            display: block;
        }

        .searchfield:focus {
            border: 1px solid #757575;
            outline: none;
        }

        .searchbutton {
            background-color: #e4e4e4;
            border: 0;
            padding: 10px;
            margin-top: 10px;
        }

        .searchbutton:focus {
            background-color: #dcdcdc;
            outline: none;
        }
    </style>
</head>
<body>
<div class="content">
    <div class="header">
        <form action="/" method="get">
            <h1>imaparchive search</h1>
            <input name="q" class="searchfield" type="text" value="{{ .Query }}"/>
            <button class="searchbutton" type="submit">Search</button>
        </form>
    </div>
    <div class="results">
		{{ if gt .Count 0 }}
        <span class="rescount">showing {{ .Max }} out of {{ .Count }} results</span>
		{{ end }}
		{{ range .Entries }}
        <h2>{{ .Title }}</h2>
        <p>{{ .Body }}</p>
        <a href="{{ .DownloadLink }}" download>Download</a>
        <span>{{ .Attachments }} Attachments</span>
        <span>{{ .Size }}</span>
        <br>
        <br>
		{{ end }}

    </div>
</div>
</body>
</html>
`

var tpl = template.Must(template.New("page").Parse(page))
