package main

import (
	"bytes"
	"fmt"
	"github.com/blevesearch/bleve"
	"github.com/jhillyerd/enmime"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type SearchConfig struct {
	Dir  string
	Host string
	Port int
}

// Search contains search and indexing logic. The index can be used while the index is updated, however because
// the mime parser is very slow and indexing one document after another is also slow, we do that in parallel.
type Search struct {
	cfg               *SearchConfig
	queue             chan string
	index             bleve.Index
	indexStatus       float64
	pendingBatch      *bleve.Batch
	pendingBatchMutex sync.Mutex
	idToFilenames     map[string]string
}

func NewSearch(cfg *SearchConfig) (*Search, error) {
	s := &Search{
		cfg:           cfg,
		idToFilenames: make(map[string]string),
		queue:         make(chan string, runtime.NumCPU()),
	}
	err := s.initIndex()
	if err != nil {
		return nil, err
	}

	s.spawnFindNewCandidates()
	go s.spawnIndexUpdate()

	return s, nil
}

func (s *Search) initIndex() error {
	mapping := bleve.NewIndexMapping()
	idxPath := filepath.Join(s.cfg.Dir, "index.bleve")
	index, err := bleve.New(idxPath, mapping)
	if err != nil {
		index, err = bleve.Open(idxPath)
		if err != nil {
			return fmt.Errorf("cannot create new or open existing index at %s: %w", idxPath, err)
		}
	}
	s.index = index
	return nil
}

func (s *Search) spawnIndexUpdate() {
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			fmt.Println("spawned indexer")
			for file := range s.queue {
				err := s.insert(file)
				if err != nil {
					fmt.Printf("failed to index %s: %v\n", file, err)
				}
			}
			wg.Done()
		}()
	}

	wg.Wait()
	fmt.Println("index update completed, applying batch")

	err := s.index.Batch(s.pendingBatch) // sadly not possible concurrently, but fast enough

	if err != nil {
		fmt.Printf("failed to apply batch: %v\n", err)
	} else {
		fmt.Printf("batch applyied")
	}

}

func (s *Search) applyBatch() {
	//fmt.Println("try to apply batch")
	s.pendingBatchMutex.Lock()
	defer s.pendingBatchMutex.Unlock()
	err := s.index.Batch(s.pendingBatch)
	if err != nil {
		fmt.Printf("failed to apply batch: %v\n", err)
	}
	s.pendingBatch = s.index.NewBatch()

	//fmt.Println("batch applied")
}

func (s *Search) insert(file string) error {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}
	email, err := enmime.ReadEnvelope(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}

	id := filepath.Base(file)
	idxModel := &IndexModel{
		Id:      id,
		File:    file,
		Subject: email.GetHeader("Subject"),
		From:    email.GetHeader("From"),
		To:      email.GetHeader("To"),
		CC:      email.GetHeader("CC"),
		Size:    int(len(b)),
	}
	for _, p := range email.Attachments {
		idxModel.Attachments += " " + p.FileName
		idxModel.AttachmentCount++
	}

	idxModel.Body = email.Text
	s.pendingBatchMutex.Lock()
	err = s.pendingBatch.Index(id, idxModel)
	s.pendingBatchMutex.Unlock()
	if err != nil {
		return fmt.Errorf("failed to add into index: %w", err)
	}
	return nil
}

func (s *Search) spawnFindNewCandidates() {
	go func() {
		var candidates []string
		err := filepath.Walk(s.cfg.Dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(info.Name(), ".eml") && info.Mode().IsRegular() {
				candidates = append(candidates, path)
			}
			return nil
		})
		if err != nil {
			fmt.Printf("failed to find emails in %s: %v\n", s.cfg.Dir, err)
		}
		fmt.Printf("found %d emails\n", len(candidates))
		var missing []string
		for _, file := range candidates {
			id := filepath.Base(file)
			s.idToFilenames[id] = file
			doc, err := s.index.Document(id)
			if err != nil {
				panic(err)
			}
			if doc == nil {
				missing = append(missing, file)
			}
		}
		fmt.Printf("found %d email to index\n", len(missing))
		s.pendingBatch = s.index.NewBatch()
		lastMsg := 0
		for i, file := range missing {
			s.queue <- file
			s.indexStatus = float64(i) / float64(len(missing))
			currentPercent := int(s.indexStatus * 100)
			if lastMsg != currentPercent {
				lastMsg = currentPercent
				fmt.Printf("index update at %d %%\n", lastMsg)
				s.applyBatch()
			}
		}
		fmt.Printf("all files added to index")
	}()
}

func (s *Search) FilenameForID(id string) string {
	return s.idToFilenames[id]
}

func (s *Search) Query(str string) *bleve.SearchResult {
	query := bleve.NewQueryStringQuery(str)
	req := bleve.NewSearchRequest(query)
	req.Fields = []string{"Id", "File", "Subject", "From", "To", "CC", "Body", "Attachments", "AttachmentCount", "Size"}
	req.Size = 1000
	res, err := s.index.Search(req)
	if err != nil {
		fmt.Printf("failed to search %s: %v\n", str, err)
		return nil
	}
	return res
}

func (s *Search) Close() error {
	close(s.queue)
	return s.index.Close()
}

type IndexModel struct {
	Id              string
	File            string
	Subject         string
	From            string
	To              string
	CC              string
	Body            string
	Attachments     string
	Size            int
	AttachmentCount int
}
