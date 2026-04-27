package stats

import (
	"sync/atomic"
)

var (
	RequestCount int64 = 0
	CacheHits    int64 = 0
	CacheMisses  int64 = 0
	VotesTotal   int64 = 0
)

func IncrementRequestCount() {
	atomic.AddInt64(&RequestCount, 1)
}

func IncrementCacheHit() {
	atomic.AddInt64(&CacheHits, 1)
}

func IncrementCacheMiss() {
	atomic.AddInt64(&CacheMisses, 1)
}

func IncrementVotes() {
	atomic.AddInt64(&VotesTotal, 1)
}

func GetRequestCount() int64 {
	return atomic.LoadInt64(&RequestCount)
}

func GetCacheHits() int64 {
	return atomic.LoadInt64(&CacheHits)
}

func GetCacheMisses() int64 {
	return atomic.LoadInt64(&CacheMisses)
}

func GetVotesTotal() int64 {
	return atomic.LoadInt64(&VotesTotal)
}