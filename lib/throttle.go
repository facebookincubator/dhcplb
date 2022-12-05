/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package dhcplb

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/time/rate"
)

// An LRU cache implementation of Throttle.
//
// We keep track of request rates per client in an LRU cache to
// keep memory usage under control against malicious requests. Each
// value in the cache is a rate.Limiter struct which is an implementation
// of Taken Bucket algorithm.
//
// Adding new items to the cache is also limited to control cache
// invalidation rate.
type Throttle struct {
	mu             sync.Mutex
	lru            *lru.Cache[string, *rate.Limiter]
	maxRatePerItem int
	cacheLimiter   *rate.Limiter
	cacheRate      int
}

// Returns true if the rate is below maximum for the given key
func (c *Throttle) OK(key string) (bool, error) {
	if c.maxRatePerItem <= 0 {
		return true, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// If the limiter is not in the cache for the given key
	// check for the cache limiter. If it is below the maximum,
	// then create a limiter, add it to the cache and allocate a bucket.
	limiter, ok := c.lru.Get(key)
	if !ok {
		if c.cacheLimiter.Allow() {
			limiter := rate.NewLimiter(rate.Limit(c.maxRatePerItem), c.maxRatePerItem)
			c.lru.Add(key, limiter)

			return limiter.Allow(), nil
		}

		err := fmt.Errorf("Cache invalidation is too fast (max: %d item/sec) - throttling", c.cacheRate)
		return false, err
	}

	// So the limiter object is in the cache. Try to allocate a bucket.
	if !limiter.Allow() {
		err := fmt.Errorf("Request rate is too high for %v (max: %d req/sec) - throttling", key, c.maxRatePerItem)
		return false, err
	}

	return true, nil
}

func (c *Throttle) len() int {
	return c.lru.Len()
}

func (c *Throttle) setRate(MaxRatePerItem int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxRatePerItem = MaxRatePerItem
}

// NewThrottle returns a Throttle struct
//
//	Capacity:
//	    Maximum capacity of the LRU cache
//
//	CacheRate (per second):
//	    Maximum allowed rate for adding new items to the cache. By that way it
//	    prevents the cache invalidation to happen too soon for the existing rate
//	    items in the cache. Cache rate will be infinite for 0 or negative values.
//
//	MaxRatePerItem (per second):
//	    Maximum allowed requests rate for each key in the cache. Throttling will
//	    be disabled for 0 or negative values. No cache will be created in that case.
func NewThrottle(Capacity int, CacheRate int, MaxRatePerItem int) (*Throttle, error) {
	if MaxRatePerItem <= 0 {
		glog.Info("No throttling will be done")
	}

	cache, err := lru.New[string, *rate.Limiter](Capacity)
	if err != nil {
		return nil, err
	}

	// Keep track of the item creation rate.
	var cacheLimiter *rate.Limiter
	if CacheRate <= 0 {
		glog.Info("No cache rate limiting will be done")
		cacheLimiter = rate.NewLimiter(rate.Inf, 1) // bucket size is ignored
	} else {
		cacheLimiter = rate.NewLimiter(rate.Limit(CacheRate), CacheRate)
	}

	throttle := &Throttle{
		lru:            cache,
		maxRatePerItem: MaxRatePerItem,
		cacheLimiter:   cacheLimiter,
		cacheRate:      CacheRate,
	}

	return throttle, nil
}
