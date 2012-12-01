// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package reporter

import (
	"testing"
	"time"
)

func TestNsToNextInterval(t *testing.T) {
	tm := time.Date(2012, 12, 1, 12, 10, 19, 1e9-200, time.UTC)
	ns := nsToNextInterval(tm, time.Minute)
	exp := time.Second*40 + 200
	if ns != exp {
		t.Fatalf("nsToNextInterval expected to return %+v instead of %+v", exp, ns)
	}
}
