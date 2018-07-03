package main

import (
	"sort"
	"sync"
)

type mockStorage interface {
	init()
	add(key string, exp Expectation)
	remove(key string)
	getOrdered() OrderedExpectations
}

// Storage is a structure with mutex to control access to expectations
type gzStorage struct {
	expectations Expectations
	mu           sync.RWMutex
}

func (storage *gzStorage) init() {
	storage = &gzStorage{expectations: make(Expectations)}
}

// AddExpectation adds new expectation to list. If expectation with same key exists, updates it
func (storage *gzStorage) add(key string, exp Expectation) {
	if storage == nil {
		panic("storage is nil")
	}
	storage.mu.Lock()
	storage.expectations[key] = exp
	storage.mu.Unlock()
}

// RemoveExpectation removes expectation with particular key
func (storage *gzStorage) remove(key string) {
	if storage == nil {
		panic("storage is nil")
	}

	storage.mu.RLock()
	_, ok := storage.expectations[key]
	storage.mu.RUnlock()

	if ok {
		storage.mu.Lock()
		delete(storage.expectations, key)
		storage.mu.Unlock()
	}
}

// OrderedExpectations is for sorting expectations by priority. the lowest priority is 0
type OrderedExpectations map[int]Expectation

func (exps OrderedExpectations) Len() int           { return len(exps) }
func (exps OrderedExpectations) Swap(i, j int)      { exps[i], exps[j] = exps[j], exps[i] }
func (exps OrderedExpectations) Less(i, j int) bool { return exps[i].Priority > exps[j].Priority }

// getOrdered returns map with int keys sorted by priority DESC.
// 0-indexed element has the highest priority
func (storage *gzStorage) getOrdered() OrderedExpectations {
	if storage == nil {
		panic("storage is nil")
	}

	listForSorting := OrderedExpectations{}
	i := 0
	storage.mu.RLock()
	for _, exp := range storage.expectations {
		listForSorting[i] = exp
		i++
	}
	storage.mu.RUnlock()
	sort.Sort(listForSorting)
	return listForSorting
}
