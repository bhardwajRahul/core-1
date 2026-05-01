package search

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/model"
)

const (
	ChannelIndexEvent = "sys-fts"
	defaultSearchSize = 25
)

type Search struct {
	pubsub cache.Volatilizer
	index  bleve.Index
}

type IndexDocument struct {
	ID     string            `json:"id"`
	DBName string            `json:"dbname"`
	Key    string            `json:"key"`
	Text   string            `json:"text"`
	Fields map[string]string `json:"fields,omitempty"`
}

func (IndexDocument) Type() string {
	return "IndexDocument"
}

func New(filename string, pubsub cache.Volatilizer) (*Search, error) {
	s := &Search{pubsub: pubsub}

	if _, err := os.Stat(filename); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		idx, err := createMapping(filename)
		if err != nil {
			return nil, err
		}
		s.index = idx
	} else {
		idx, err := bleve.Open(filename)
		if err != nil {
			return nil, err
		}

		s.index = idx
	}

	if s.pubsub != nil {
		go s.setupIndexEvent()
	}
	return s, nil
}

func createMapping(filename string) (bleve.Index, error) {
	mapping := bleve.NewDocumentMapping()

	dbMap := bleve.NewKeywordFieldMapping()
	mapping.AddFieldMappingsAt("dbname", dbMap)

	keyMap := bleve.NewKeywordFieldMapping()
	mapping.AddFieldMappingsAt("key", keyMap)

	textMap := bleve.NewTextFieldMapping()
	textMap.Analyzer = "en"
	mapping.AddFieldMappingsAt("text", textMap)

	fieldsMap := bleve.NewDocumentMapping()
	fieldsMap.Dynamic = true
	fieldsMap.DefaultAnalyzer = "en"
	mapping.AddSubDocumentMapping("fields", fieldsMap)

	idxmap := bleve.NewIndexMapping()
	idxmap.AddDocumentMapping("IndexDocument", mapping)
	idxmap.DefaultMapping = mapping

	return bleve.New(filename, idxmap)
}

func (s *Search) Index(dbName, col, id, text string) error {
	return s.indexDocument(IndexDocument{
		ID:     id,
		DBName: dbName,
		Key:    col,
		Text:   text,
	})
}

func (s *Search) IndexFields(dbName, col, id string, fields map[string]string) error {
	doc := IndexDocument{
		ID:     id,
		DBName: dbName,
		Key:    col,
		Fields: fields,
		Text:   combineFields(fields),
	}

	return s.indexDocument(doc)
}

func (s *Search) indexDocument(doc IndexDocument) error {
	if s == nil || s.index == nil {
		return errors.New("search index is not initialized")
	}

	docID := documentID(doc.DBName, doc.Key, doc.ID)
	if legacyID := legacyDocumentID(doc.DBName, doc.Key, doc.ID); legacyID != docID {
		_ = s.index.Delete(legacyID)
	}
	return s.index.Index(docID, doc)
}

func combineFields(fields map[string]string) string {
	if len(fields) == 0 {
		return ""
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		if value := strings.TrimSpace(fields[key]); value != "" {
			if b.Len() > 0 {
				b.WriteString(" ")
			}
			b.WriteString(value)
		}
	}
	return b.String()
}

func (s *Search) PublishIndex(dbName, col, id, text string) error {
	if s == nil || s.pubsub == nil {
		return errors.New("search pubsub is not initialized")
	}

	doc := IndexDocument{
		ID:     id,
		DBName: dbName,
		Key:    col,
		Text:   text,
	}

	b, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	msg := model.Command{
		SID:           "system",
		Type:          "system",
		Data:          string(b),
		Channel:       ChannelIndexEvent,
		Token:         "system",
		IsSystemEvent: true,
	}

	return s.pubsub.Publish(msg)
}

func (s *Search) Delete(dbName, col, id string) error {
	if s == nil || s.index == nil {
		return errors.New("search index is not initialized")
	}

	if err := s.index.Delete(documentID(dbName, col, id)); err != nil {
		return err
	}
	if legacyID := legacyDocumentID(dbName, col, id); legacyID != documentID(dbName, col, id) {
		return s.index.Delete(legacyID)
	}

	return nil
}

type SearchResult struct {
	DBName string
	Col    string
	IDs    []string
}

func (s *Search) Search(dbName, col, keywords string) (SearchResult, error) {
	sr := SearchResult{DBName: dbName, Col: col}

	if s == nil || s.index == nil {
		return sr, errors.New("search index is not initialized")
	}

	tokens := strings.Fields(keywords)
	if len(tokens) == 0 {
		return sr, nil
	}

	var queries []query.Query

	dbQry := bleve.NewTermQuery(dbName)
	dbQry.SetField("dbname")

	colQry := bleve.NewTermQuery(col)
	colQry.SetField("key")

	queries = append(queries, dbQry)
	queries = append(queries, colQry)

	for _, keyword := range tokens {
		keyword = strings.ToLower(keyword)

		mq := bleve.NewMatchQuery(keyword)
		mq.SetField("text")
		mq.SetBoost(2)

		pq := bleve.NewPrefixQuery(keyword)
		pq.SetField("text")

		fq := bleve.NewFuzzyQuery(keyword)
		fq.SetField("text")
		fq.SetFuzziness(1)

		queries = append(queries, bleve.NewDisjunctionQuery(mq, pq, fq))
	}

	conj := bleve.NewConjunctionQuery(queries...)
	if conj == nil {
		return sr, errors.New("conj is nil")
	}

	req := bleve.NewSearchRequest(conj)

	if req == nil {
		return sr, errors.New("search request is nil")
	}
	req.Size = defaultSearchSize

	results, err := s.index.Search(req)
	if err != nil {
		return sr, err
	}

	for _, r := range results.Hits {
		id, ok := parseDocumentID(r.ID)
		if !ok {
			continue
		}

		sr.IDs = append(sr.IDs, id)
	}

	return sr, nil
}

func documentID(dbName, col, id string) string {
	return fmt.Sprintf("%s\n%s\n%s", dbName, col, id)
}

func legacyDocumentID(dbName, col, id string) string {
	return fmt.Sprintf("%s_%s_%s", dbName, col, id)
}

func parseDocumentID(value string) (string, bool) {
	parts := strings.SplitN(value, "\n", 3)
	if len(parts) == 3 {
		return parts[2], true
	}

	parts = strings.Split(value, "_")
	if len(parts) == 3 {
		return parts[2], true
	}

	return "", false
}

func (s *Search) setupIndexEvent() {
	receiver := make(chan model.Command)
	close := make(chan bool)

	go s.pubsub.Subscribe(receiver, "system", ChannelIndexEvent, close)

	for {
		select {
		case msg := <-receiver:
			go s.receivedIndexEvent(msg.Data)
		case <-close:
			return
		}
	}
}

func (s *Search) receivedIndexEvent(data string) {
	var doc IndexDocument
	if err := json.Unmarshal([]byte(data), &doc); err != nil {
		log.Println(err)
		return
	}

	if err := s.indexDocument(doc); err != nil {
		log.Println(err)
	}
}

func (s *Search) Close() {
	if s == nil || s.index == nil {
		return
	}

	if err := s.index.Close(); err != nil {
		log.Println(err)
	}
}
