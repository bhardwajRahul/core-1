package memory

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/database"
	sbquery "github.com/staticbackendhq/core/internal/query"
)

/*const (
	FieldID        = "id"
	FieldAccountID = "accountId"
	FieldOwnerID   = "ownerId"
)*/

var errCollectionNotFound = errors.New("collection not found")

func init() {
	gob.Register(map[string]any{})
	gob.Register([]any{})
	gob.Register(time.Time{})
}

var mx *sync.RWMutex = &sync.RWMutex{}

type Memory struct {
	DB              map[string]map[string][]byte
	PublishDocument cache.PublishDocumentEvent
}

func New(pubdoc cache.PublishDocumentEvent) database.Persister {
	db := make(map[string]map[string][]byte)

	if err := initDB(db); err != nil {
		log.Fatal(err)
	}

	return &Memory{DB: db, PublishDocument: pubdoc}
}

func initDB(db map[string]map[string][]byte) error {
	db["sb_customers"] = make(map[string][]byte)
	db["sb_apps"] = make(map[string][]byte)
	return nil
}

func (m *Memory) NewID() string {
	return uuid.NewString()
}

func (m *Memory) Ping() error {
	return nil
}

func (m *Memory) CreateIndex(dbName, col, field string) error {
	return nil
}

func (m *Memory) CreateTypedIndex(dbName, col, field string, typ database.IndexType) error {
	if !database.IsSupportedIndexType(typ) {
		return fmt.Errorf("index type %q is not supported", typ)
	}
	return nil
}

func mustEnc(v any) []byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

func mustDec(b []byte, v any) error {
	return gob.NewDecoder(bytes.NewReader(b)).Decode(v)
}

func create[T any](m *Memory, dbName, col, id string, v T) error {
	key := fmt.Sprintf("%s_%s", dbName, col)

	repo, ok := m.DB[key]
	if !ok {
		repo = make(map[string][]byte)
	}

	repo[id] = mustEnc(v)

	mx.Lock()
	m.DB[key] = repo
	mx.Unlock()
	return nil
}

func getByID[T any](m *Memory, dbName, col, id string, v T) error {
	key := fmt.Sprintf("%s_%s", dbName, col)

	mx.Lock()
	repo, ok := m.DB[key]
	mx.Unlock()

	if !ok {
		if strings.HasPrefix(col, "sb_") {
			return nil
		}
		return errCollectionNotFound
	}

	b, ok := repo[id]
	if !ok {
		return errors.New("document not found")
	} else if err := mustDec(b, v); err != nil {
		return err
	}
	return nil
}

func all[T any](m *Memory, dbName, col string) (list []T, err error) {
	key := fmt.Sprintf("%s_%s", dbName, col)

	mx.Lock()
	repo, ok := m.DB[key]
	mx.Unlock()

	if !ok {
		if strings.HasPrefix(col, "sb_") {
			return []T{}, nil
		}
		return nil, errCollectionNotFound
	}

	for _, v := range repo {
		var li T
		if err = mustDec(v, &li); err != nil {
			return
		}

		list = append(list, li)
	}

	return
}

func filter[T any](list []T, fn func(x T) bool) []T {
	var results []T
	for _, item := range list {
		if fn(item) {
			results = append(results, item)
		}
	}

	return results
}

func filterByClauses(list []map[string]any, filter map[string]any) (filtered []map[string]any) {
	if q, ok := sbquery.FromFilter(filter); ok {
		for _, doc := range list {
			if matchQuery(doc, q) {
				filtered = append(filtered, doc)
			}
		}
		return
	}

	for _, doc := range list {
		matches := 0
		for k, v := range filter {
			op, field := extractOperatorAndValue(k)
			switch op {
			case "=":
				if equal(doc[field], v) {
					matches++
				}
			case "!=":
				if notEqual(doc[field], v) {
					matches++
				}
			case ">":
				if greater(doc[field], v) {
					matches++
				}
			case "<":
				if lower(doc[field], v) {
					matches++
				}
			case ">=":
				if greaterThanEqual(doc[field], v) {
					matches++
				}
			case "<=":
				if lowerThanEqual(doc[field], v) {
					matches++
				}
			case "contains":
				if contains(doc[field], v) {
					matches++
				}
			case "!contains":
				if notContains(doc[field], v) {
					matches++
				}
			}
		}

		if matches == len(filter) {
			filtered = append(filtered, doc)
		}
	}
	return
}

func matchQuery(doc map[string]any, q sbquery.Query) bool {
	for _, clause := range q {
		left := doc[clause.Field]
		right := operandValue(doc, clause.Value)

		switch clause.Operator {
		case sbquery.OpEqual:
			if !equal(left, right) {
				return false
			}
		case sbquery.OpNotEqual:
			if !notEqual(left, right) {
				return false
			}
		case sbquery.OpGreater:
			if !compare(left, right, clause.Value.Type, func(c int) bool { return c > 0 }) {
				return false
			}
		case sbquery.OpLower:
			if !compare(left, right, clause.Value.Type, func(c int) bool { return c < 0 }) {
				return false
			}
		case sbquery.OpGreaterEq:
			if !compare(left, right, clause.Value.Type, func(c int) bool { return c >= 0 }) {
				return false
			}
		case sbquery.OpLowerEq:
			if !compare(left, right, clause.Value.Type, func(c int) bool { return c <= 0 }) {
				return false
			}
		case sbquery.OpIn:
			if !in(left, right) {
				return false
			}
		case sbquery.OpNotIn:
			if in(left, right) {
				return false
			}
		case sbquery.OpContains:
			if !contains(left, right) {
				return false
			}
		case sbquery.OpNotContains:
			if !notContains(left, right) {
				return false
			}
		}
	}
	return true
}

func operandValue(doc map[string]any, operand sbquery.Operand) any {
	if operand.Kind == sbquery.OperandField {
		return doc[operand.Field]
	}
	return operand.Value
}

func compare(v any, val any, typ sbquery.ValueType, fn func(int) bool) bool {
	switch typ {
	case sbquery.TypeNumber:
		left, ok := number(v)
		if !ok {
			return false
		}
		right, ok := number(val)
		if !ok {
			return false
		}
		switch {
		case left < right:
			return fn(-1)
		case left > right:
			return fn(1)
		default:
			return fn(0)
		}
	case sbquery.TypeBoolean:
		left, ok := boolean(v)
		if !ok {
			return false
		}
		right, ok := boolean(val)
		if !ok {
			return false
		}
		switch {
		case !left && right:
			return fn(-1)
		case left && !right:
			return fn(1)
		default:
			return fn(0)
		}
	}

	return fn(strings.Compare(fmt.Sprintf("%v", v), fmt.Sprintf("%v", val)))
}

func number(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		f, err := strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
		return f, err == nil
	}
}

func boolean(v any) (bool, bool) {
	switch b := v.(type) {
	case bool:
		return b, true
	case string:
		switch strings.ToLower(b) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	}
	return false, false
}

func in(v any, val any) bool {
	switch list := val.(type) {
	case []any:
		for _, item := range list {
			if equal(v, item) {
				return true
			}
		}
	case []string:
		for _, item := range list {
			if equal(v, item) {
				return true
			}
		}
	default:
		return equal(v, val)
	}
	return false
}

func sortSlice[T any](list []T, fn func(a, b T) bool) []T {
	sort.Slice(list, func(i, j int) bool {
		return fn(list[i], list[j])
	})
	return list
}

/*
func create_bolt[T any](m *Memory, dbName, col, id string, v T) error {
	bucketName := fmt.Sprintf("%s_%s", dbName, col)

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return m.DB.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}

		return b.Put([]byte(id), data)
	})
}

func getByID_bolt[T any](m *Memory, dbName, col, id string, v T) error {
	bucketName := fmt.Sprintf("%s_%s", dbName, col)

	var data []byte
	err := m.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			if bytes.Equal(k, []byte(id)) {
				data = v
				return nil
			}
		}
		return fmt.Errorf("cannot find id: %s in bucket: %s", id, bucketName)
	})
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

func all_bolt[T any](m *Memory, dbName, col string) (list []T, err error) {
	bucketName := fmt.Sprintf("%s_%s", dbName, col)
	err = m.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var m T
			if err := json.Unmarshal(v, m); err != nil {
				return err
			}

			list = append(list, m)
		}
		return nil
	})

	return
}
*/
