// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package reporter

type counterDeltaCache struct {
	previousCounters map[string]int64
}

func (c *counterDeltaCache) delta(name string, current int64) int64 {
	if c.previousCounters == nil {
		c.previousCounters = make(map[string]int64)
	}

	prev := c.previousCounters[name]
	c.previousCounters[name] = current
	return current - prev
}
