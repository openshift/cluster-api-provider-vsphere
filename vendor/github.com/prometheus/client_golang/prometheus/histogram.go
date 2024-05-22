// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheus

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	//nolint:staticcheck // Ignore SA1019. Need to keep deprecated package for compatibility.
	"github.com/golang/protobuf/proto"

	dto "github.com/prometheus/client_model/go"
)

// A Histogram counts individual observations from an event or sample stream in
// configurable buckets. Similar to a summary, it also provides a sum of
// observations and an observation count.
//
// On the Prometheus server, quantiles can be calculated from a Histogram using
// the histogram_quantile function in the query language.
//
// Note that Histograms, in contrast to Summaries, can be aggregated with the
// Prometheus query language (see the documentation for detailed
// procedures). However, Histograms require the user to pre-define suitable
// buckets, and they are in general less accurate. The Observe method of a
// Histogram has a very low performance overhead in comparison with the Observe
// method of a Summary.
//
// To create Histogram instances, use NewHistogram.
type Histogram interface {
	Metric
	Collector

	// Observe adds a single observation to the histogram. Observations are
	// usually positive or zero. Negative observations are accepted but
	// prevent current versions of Prometheus from properly detecting
	// counter resets in the sum of observations. See
	// https://prometheus.io/docs/practices/histograms/#count-and-sum-of-observations
	// for details.
	Observe(float64)
}

// bucketLabel is used for the label that defines the upper bound of a
// bucket of a histogram ("le" -> "less or equal").
const bucketLabel = "le"

// DefBuckets are the default Histogram buckets. The default buckets are
// tailored to broadly measure the response time (in seconds) of a network
// service. Most likely, however, you will be required to define buckets
// customized to your use case.
var (
	DefBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

	errBucketLabelNotAllowed = fmt.Errorf(
		"%q is not allowed as label name in histograms", bucketLabel,
	)
)

// LinearBuckets creates 'count' buckets, each 'width' wide, where the lowest
// bucket has an upper bound of 'start'. The final +Inf bucket is not counted
// and not included in the returned slice. The returned slice is meant to be
// used for the Buckets field of HistogramOpts.
//
// The function panics if 'count' is zero or negative.
func LinearBuckets(start, width float64, count int) []float64 {
	if count < 1 {
		panic("LinearBuckets needs a positive count")
	}
	buckets := make([]float64, count)
	for i := range buckets {
		buckets[i] = start
		start += width
	}
	return buckets
}

// ExponentialBuckets creates 'count' buckets, where the lowest bucket has an
// upper bound of 'start' and each following bucket's upper bound is 'factor'
// times the previous bucket's upper bound. The final +Inf bucket is not counted
// and not included in the returned slice. The returned slice is meant to be
// used for the Buckets field of HistogramOpts.
//
// The function panics if 'count' is 0 or negative, if 'start' is 0 or negative,
// or if 'factor' is less than or equal 1.
func ExponentialBuckets(start, factor float64, count int) []float64 {
	if count < 1 {
		panic("ExponentialBuckets needs a positive count")
	}
	if start <= 0 {
		panic("ExponentialBuckets needs a positive start value")
	}
	if factor <= 1 {
		panic("ExponentialBuckets needs a factor greater than 1")
	}
	buckets := make([]float64, count)
	for i := range buckets {
		buckets[i] = start
		start *= factor
	}
	return buckets
}

// HistogramOpts bundles the options for creating a Histogram metric. It is
// mandatory to set Name to a non-empty string. All other fields are optional
// and can safely be left at their zero value, although it is strongly
// encouraged to set a Help string.
type HistogramOpts struct {
	// Namespace, Subsystem, and Name are components of the fully-qualified
	// name of the Histogram (created by joining these components with
	// "_"). Only Name is mandatory, the others merely help structuring the
	// name. Note that the fully-qualified name of the Histogram must be a
	// valid Prometheus metric name.
	Namespace string
	Subsystem string
	Name      string

	// Help provides information about this Histogram.
	//
	// Metrics with the same fully-qualified name must have the same Help
	// string.
	Help string

	// ConstLabels are used to attach fixed labels to this metric. Metrics
	// with the same fully-qualified name must have the same label names in
	// their ConstLabels.
	//
	// ConstLabels are only used rarely. In particular, do not use them to
	// attach the same labels to all your metrics. Those use cases are
	// better covered by target labels set by the scraping Prometheus
	// server, or by one specific metric (e.g. a build_info or a
	// machine_role metric). See also
	// https://prometheus.io/docs/instrumenting/writing_exporters/#target-labels-not-static-scraped-labels
	ConstLabels Labels

	// Buckets defines the buckets into which observations are counted. Each
	// element in the slice is the upper inclusive bound of a bucket. The
	// values must be sorted in strictly increasing order. There is no need
	// to add a highest bucket with +Inf bound, it will be added
	// implicitly. The default value is DefBuckets.
	Buckets []float64

	// If NativeHistogramBucketFactor is greater than one, so-called sparse
	// buckets are used (in addition to the regular buckets, if defined
	// above). A Histogram with sparse buckets will be ingested as a Native
	// Histogram by a Prometheus server with that feature enabled (requires
	// Prometheus v2.40+). Sparse buckets are exponential buckets covering
	// the whole float64 range (with the exception of the “zero” bucket, see
	// NativeHistogramZeroThreshold below). From any one bucket to the next,
	// the width of the bucket grows by a constant
	// factor. NativeHistogramBucketFactor provides an upper bound for this
	// factor (exception see below). The smaller
	// NativeHistogramBucketFactor, the more buckets will be used and thus
	// the more costly the histogram will become. A generally good trade-off
	// between cost and accuracy is a value of 1.1 (each bucket is at most
	// 10% wider than the previous one), which will result in each power of
	// two divided into 8 buckets (e.g. there will be 8 buckets between 1
	// and 2, same as between 2 and 4, and 4 and 8, etc.).
	//
	// Details about the actually used factor: The factor is calculated as
	// 2^(2^-n), where n is an integer number between (and including) -4 and
	// 8. n is chosen so that the resulting factor is the largest that is
	// still smaller or equal to NativeHistogramBucketFactor. Note that the
	// smallest possible factor is therefore approx. 1.00271 (i.e. 2^(2^-8)
	// ). If NativeHistogramBucketFactor is greater than 1 but smaller than
	// 2^(2^-8), then the actually used factor is still 2^(2^-8) even though
	// it is larger than the provided NativeHistogramBucketFactor.
	//
	// NOTE: Native Histograms are still an experimental feature. Their
	// behavior might still change without a major version
	// bump. Subsequently, all NativeHistogram... options here might still
	// change their behavior or name (or might completely disappear) without
	// a major version bump.
	NativeHistogramBucketFactor float64
	// All observations with an absolute value of less or equal
	// NativeHistogramZeroThreshold are accumulated into a “zero” bucket.
	// For best results, this should be close to a bucket boundary. This is
	// usually the case if picking a power of two. If
	// NativeHistogramZeroThreshold is left at zero,
	// DefNativeHistogramZeroThreshold is used as the threshold. To
	// configure a zero bucket with an actual threshold of zero (i.e. only
	// observations of precisely zero will go into the zero bucket), set
	// NativeHistogramZeroThreshold to the NativeHistogramZeroThresholdZero
	// constant (or any negative float value).
	NativeHistogramZeroThreshold float64

	// The remaining fields define a strategy to limit the number of
	// populated sparse buckets. If NativeHistogramMaxBucketNumber is left
	// at zero, the number of buckets is not limited. (Note that this might
	// lead to unbounded memory consumption if the values observed by the
	// Histogram are sufficiently wide-spread. In particular, this could be
	// used as a DoS attack vector. Where the observed values depend on
	// external inputs, it is highly recommended to set a
	// NativeHistogramMaxBucketNumber.) Once the set
	// NativeHistogramMaxBucketNumber is exceeded, the following strategy is
	// enacted:
	//  - First, if the last reset (or the creation) of the histogram is at
	//    least NativeHistogramMinResetDuration ago, then the whole
	//    histogram is reset to its initial state (including regular
	//    buckets).
	//  - If less time has passed, or if NativeHistogramMinResetDuration is
	//    zero, no reset is performed. Instead, the zero threshold is
	//    increased sufficiently to reduce the number of buckets to or below
	//    NativeHistogramMaxBucketNumber, but not to more than
	//    NativeHistogramMaxZeroThreshold. Thus, if
	//    NativeHistogramMaxZeroThreshold is already at or below the current
	//    zero threshold, nothing happens at this step.
	//  - After that, if the number of buckets still exceeds
	//    NativeHistogramMaxBucketNumber, the resolution of the histogram is
	//    reduced by doubling the width of the sparse buckets (up to a
	//    growth factor between one bucket to the next of 2^(2^4) = 65536,
	//    see above).
	//  - Any increased zero threshold or reduced resolution is reset back
	//    to their original values once NativeHistogramMinResetDuration has
	//    passed (since the last reset or the creation of the histogram).
	NativeHistogramMaxBucketNumber  uint32
	NativeHistogramMinResetDuration time.Duration
	NativeHistogramMaxZeroThreshold float64

	// now is for testing purposes, by default it's time.Now.
	now func() time.Time
}

// HistogramVecOpts bundles the options to create a HistogramVec metric.
// It is mandatory to set HistogramOpts, see there for mandatory fields. VariableLabels
// is optional and can safely be left to its default value.
type HistogramVecOpts struct {
	HistogramOpts

	// VariableLabels are used to partition the metric vector by the given set
	// of labels. Each label value will be constrained with the optional Constraint
	// function, if provided.
	VariableLabels ConstrainableLabels
}

// NewHistogram creates a new Histogram based on the provided HistogramOpts. It
// panics if the buckets in HistogramOpts are not in strictly increasing order.
//
// The returned implementation also implements ExemplarObserver. It is safe to
// perform the corresponding type assertion. Exemplars are tracked separately
// for each bucket.
func NewHistogram(opts HistogramOpts) Histogram {
	return newHistogram(
		NewDesc(
			BuildFQName(opts.Namespace, opts.Subsystem, opts.Name),
			opts.Help,
			nil,
			opts.ConstLabels,
		),
		opts,
	)
}

func newHistogram(desc *Desc, opts HistogramOpts, labelValues ...string) Histogram {
	if len(desc.variableLabels) != len(labelValues) {
		panic(makeInconsistentCardinalityError(desc.fqName, desc.variableLabels, labelValues))
	}

	for _, n := range desc.variableLabels {
		if n == bucketLabel {
			panic(errBucketLabelNotAllowed)
		}
	}
	for _, lp := range desc.constLabelPairs {
		if lp.GetName() == bucketLabel {
			panic(errBucketLabelNotAllowed)
		}
	}

	if opts.now == nil {
		opts.now = time.Now
	}

	h := &histogram{
		desc:                            desc,
		upperBounds:                     opts.Buckets,
		labelPairs:                      MakeLabelPairs(desc, labelValues),
		nativeHistogramMaxBuckets:       opts.NativeHistogramMaxBucketNumber,
		nativeHistogramMaxZeroThreshold: opts.NativeHistogramMaxZeroThreshold,
		nativeHistogramMinResetDuration: opts.NativeHistogramMinResetDuration,
		lastResetTime:                   opts.now(),
		now:                             opts.now,
	}
	if len(h.upperBounds) == 0 && opts.NativeHistogramBucketFactor <= 1 {
		h.upperBounds = DefBuckets
	}
	if opts.NativeHistogramBucketFactor <= 1 {
		h.nativeHistogramSchema = math.MinInt32 // To mark that there are no sparse buckets.
	} else {
		switch {
		case opts.NativeHistogramZeroThreshold > 0:
			h.nativeHistogramZeroThreshold = opts.NativeHistogramZeroThreshold
		case opts.NativeHistogramZeroThreshold == 0:
			h.nativeHistogramZeroThreshold = DefNativeHistogramZeroThreshold
		} // Leave h.nativeHistogramZeroThreshold at 0 otherwise.
		h.nativeHistogramSchema = pickSchema(opts.NativeHistogramBucketFactor)
	}
	for i, upperBound := range h.upperBounds {
		if i < len(h.upperBounds)-1 {
			if upperBound >= h.upperBounds[i+1] {
				panic(fmt.Errorf(
					"histogram buckets must be in increasing order: %f >= %f",
					upperBound, h.upperBounds[i+1],
				))
			}
		} else {
			if math.IsInf(upperBound, +1) {
				// The +Inf bucket is implicit. Remove it here.
				h.upperBounds = h.upperBounds[:i]
			}
		}
	}
	// Finally we know the final length of h.upperBounds and can make buckets
	// for both counts as well as exemplars:
	h.counts[0].buckets = make([]uint64, len(h.upperBounds))
	h.counts[1].buckets = make([]uint64, len(h.upperBounds))
	h.exemplars = make([]atomic.Value, len(h.upperBounds)+1)

	h.init(h) // Init self-collection.
	return h
}

type histogramCounts struct {
	// sumBits contains the bits of the float64 representing the sum of all
	// observations. sumBits and count have to go first in the struct to
	// guarantee alignment for atomic operations.
	// http://golang.org/pkg/sync/atomic/#pkg-note-BUG
	sumBits uint64
	count   uint64
	buckets []uint64
}

type histogram struct {
	// countAndHotIdx enables lock-free writes with use of atomic updates.
	// The most significant bit is the hot index [0 or 1] of the count field
	// below. Observe calls update the hot one. All remaining bits count the
	// number of Observe calls. Observe starts by incrementing this counter,
	// and finish by incrementing the count field in the respective
	// histogramCounts, as a marker for completion.
	//
	// Calls of the Write method (which are non-mutating reads from the
	// perspective of the histogram) swap the hot–cold under the writeMtx
	// lock. A cooldown is awaited (while locked) by comparing the number of
	// observations with the initiation count. Once they match, then the
	// last observation on the now cool one has completed. All cool fields must
	// be merged into the new hot before releasing writeMtx.
	//
	// Fields with atomic access first! See alignment constraint:
	// http://golang.org/pkg/sync/atomic/#pkg-note-BUG
	countAndHotIdx uint64

	selfCollector
	desc     *Desc
	writeMtx sync.Mutex // Only used in the Write method.

	// Two counts, one is "hot" for lock-free observations, the other is
	// "cold" for writing out a dto.Metric. It has to be an array of
	// pointers to guarantee 64bit alignment of the histogramCounts, see
	// http://golang.org/pkg/sync/atomic/#pkg-note-BUG.
	counts [2]*histogramCounts

	upperBounds                     []float64
	labelPairs                      []*dto.LabelPair
	exemplars                       []atomic.Value // One more than buckets (to include +Inf), each a *dto.Exemplar.
	nativeHistogramSchema           int32          // The initial schema. Set to math.MinInt32 if no sparse buckets are used.
	nativeHistogramZeroThreshold    float64        // The initial zero threshold.
	nativeHistogramMaxZeroThreshold float64
	nativeHistogramMaxBuckets       uint32
	nativeHistogramMinResetDuration time.Duration
	// lastResetTime is protected by mtx. It is also used as created timestamp.
	lastResetTime time.Time

	// now is for testing purposes, by default it's time.Now.
	now func() time.Time
}

func (h *histogram) Desc() *Desc {
	return h.desc
}

func (h *histogram) Observe(v float64) {
	h.observe(v, h.findBucket(v))
}

func (h *histogram) ObserveWithExemplar(v float64, e Labels) {
	i := h.findBucket(v)
	h.observe(v, i)
	h.updateExemplar(v, i, e)
}

func (h *histogram) Write(out *dto.Metric) error {
	// For simplicity, we protect this whole method by a mutex. It is not in
	// the hot path, i.e. Observe is called much more often than Write. The
	// complication of making Write lock-free isn't worth it, if possible at
	// all.
	h.writeMtx.Lock()
	defer h.writeMtx.Unlock()

	// Adding 1<<63 switches the hot index (from 0 to 1 or from 1 to 0)
	// without touching the count bits. See the struct comments for a full
	// description of the algorithm.
	n := atomic.AddUint64(&h.countAndHotIdx, 1<<63)
	// count is contained unchanged in the lower 63 bits.
	count := n & ((1 << 63) - 1)
	// The most significant bit tells us which counts is hot. The complement
	// is thus the cold one.
	hotCounts := h.counts[n>>63]
	coldCounts := h.counts[(^n)>>63]

	// Await cooldown.
	for count != atomic.LoadUint64(&coldCounts.count) {
		runtime.Gosched() // Let observations get work done.
	}

	his := &dto.Histogram{
		Bucket:      make([]*dto.Bucket, len(h.upperBounds)),
		SampleCount: proto.Uint64(count),
		SampleSum:   proto.Float64(math.Float64frombits(atomic.LoadUint64(&coldCounts.sumBits))),
	}
	var cumCount uint64
	for i, upperBound := range h.upperBounds {
		cumCount += atomic.LoadUint64(&coldCounts.buckets[i])
		his.Bucket[i] = &dto.Bucket{
			CumulativeCount: proto.Uint64(cumCount),
			UpperBound:      proto.Float64(upperBound),
		}
		if e := h.exemplars[i].Load(); e != nil {
			his.Bucket[i].Exemplar = e.(*dto.Exemplar)
		}
	}
	// If there is an exemplar for the +Inf bucket, we have to add that bucket explicitly.
	if e := h.exemplars[len(h.upperBounds)].Load(); e != nil {
		b := &dto.Bucket{
			CumulativeCount: proto.Uint64(count),
			UpperBound:      proto.Float64(math.Inf(1)),
			Exemplar:        e.(*dto.Exemplar),
		}
		his.Bucket = append(his.Bucket, b)
	}

	out.Histogram = his
	out.Label = h.labelPairs

	// Finally add all the cold counts to the new hot counts and reset the cold counts.
	atomic.AddUint64(&hotCounts.count, count)
	atomic.StoreUint64(&coldCounts.count, 0)
	for {
		oldBits := atomic.LoadUint64(&hotCounts.sumBits)
		newBits := math.Float64bits(math.Float64frombits(oldBits) + his.GetSampleSum())
		if atomic.CompareAndSwapUint64(&hotCounts.sumBits, oldBits, newBits) {
			atomic.StoreUint64(&coldCounts.sumBits, 0)
			break
		}
	}
	for i := range h.upperBounds {
		atomic.AddUint64(&hotCounts.buckets[i], atomic.LoadUint64(&coldCounts.buckets[i]))
		atomic.StoreUint64(&coldCounts.buckets[i], 0)
	}
	return nil
}

// findBucket returns the index of the bucket for the provided value, or
// len(h.upperBounds) for the +Inf bucket.
func (h *histogram) findBucket(v float64) int {
	// TODO(beorn7): For small numbers of buckets (<30), a linear search is
	// slightly faster than the binary search. If we really care, we could
	// switch from one search strategy to the other depending on the number
	// of buckets.
	//
	// Microbenchmarks (BenchmarkHistogramNoLabels):
	// 11 buckets: 38.3 ns/op linear - binary 48.7 ns/op
	// 100 buckets: 78.1 ns/op linear - binary 54.9 ns/op
	// 300 buckets: 154 ns/op linear - binary 61.6 ns/op
	return sort.SearchFloat64s(h.upperBounds, v)
}

// observe is the implementation for Observe without the findBucket part.
func (h *histogram) observe(v float64, bucket int) {
	// We increment h.countAndHotIdx so that the counter in the lower
	// 63 bits gets incremented. At the same time, we get the new value
	// back, which we can use to find the currently-hot counts.
	n := atomic.AddUint64(&h.countAndHotIdx, 1)
	hotCounts := h.counts[n>>63]

	if bucket < len(h.upperBounds) {
		atomic.AddUint64(&hotCounts.buckets[bucket], 1)
	}
	if h.nativeHistogramMaxBuckets >= atomic.LoadUint32(&counts.nativeHistogramBucketsNumber) {
		return // Bucket limit not exceeded yet.
	}

	h.mtx.Lock()
	defer h.mtx.Unlock()

	// The hot counts might have been swapped just before we acquired the
	// lock. Re-fetch the hot counts first...
	n := atomic.LoadUint64(&h.countAndHotIdx)
	hotIdx := n >> 63
	coldIdx := (^n) >> 63
	hotCounts := h.counts[hotIdx]
	coldCounts := h.counts[coldIdx]
	// ...and then check again if we really have to reduce the bucket count.
	if h.nativeHistogramMaxBuckets >= atomic.LoadUint32(&hotCounts.nativeHistogramBucketsNumber) {
		return // Bucket limit not exceeded after all.
	}
	// Try the various strategies in order.
	if h.maybeReset(hotCounts, coldCounts, coldIdx, value, bucket) {
		return
	}
	if h.maybeWidenZeroBucket(hotCounts, coldCounts) {
		return
	}
	h.doubleBucketWidth(hotCounts, coldCounts)
}

// maybeReset resets the whole histogram if at least h.nativeHistogramMinResetDuration
// has been passed. It returns true if the histogram has been reset. The caller
// must have locked h.mtx.
func (h *histogram) maybeReset(
	hot, cold *histogramCounts, coldIdx uint64, value float64, bucket int,
) bool {
	// We are using the possibly mocked h.now() rather than
	// time.Since(h.lastResetTime) to enable testing.
	if h.nativeHistogramMinResetDuration == 0 ||
		h.now().Sub(h.lastResetTime) < h.nativeHistogramMinResetDuration {
		return false
	}
	// Completely reset coldCounts.
	h.resetCounts(cold)
	// Repeat the latest observation to not lose it completely.
	cold.observe(value, bucket, true)
	// Make coldCounts the new hot counts while resetting countAndHotIdx.
	n := atomic.SwapUint64(&h.countAndHotIdx, (coldIdx<<63)+1)
	count := n & ((1 << 63) - 1)
	waitForCooldown(count, hot)
	// Finally, reset the formerly hot counts, too.
	h.resetCounts(hot)
	h.lastResetTime = h.now()
	return true
}

// maybeWidenZeroBucket widens the zero bucket until it includes the existing
// buckets closest to the zero bucket (which could be two, if an equidistant
// negative and a positive bucket exists, but usually it's only one bucket to be
// merged into the new wider zero bucket). h.nativeHistogramMaxZeroThreshold
// limits how far the zero bucket can be extended, and if that's not enough to
// include an existing bucket, the method returns false. The caller must have
// locked h.mtx.
func (h *histogram) maybeWidenZeroBucket(hot, cold *histogramCounts) bool {
	currentZeroThreshold := math.Float64frombits(atomic.LoadUint64(&hot.nativeHistogramZeroThresholdBits))
	if currentZeroThreshold >= h.nativeHistogramMaxZeroThreshold {
		return false
	}
	// Find the key of the bucket closest to zero.
	smallestKey := findSmallestKey(&hot.nativeHistogramBucketsPositive)
	smallestNegativeKey := findSmallestKey(&hot.nativeHistogramBucketsNegative)
	if smallestNegativeKey < smallestKey {
		smallestKey = smallestNegativeKey
	}
	if smallestKey == math.MaxInt32 {
		return false
	}
	newZeroThreshold := getLe(smallestKey, atomic.LoadInt32(&hot.nativeHistogramSchema))
	if newZeroThreshold > h.nativeHistogramMaxZeroThreshold {
		return false // New threshold would exceed the max threshold.
	}
	atomic.StoreUint64(&cold.nativeHistogramZeroThresholdBits, math.Float64bits(newZeroThreshold))
	// Remove applicable buckets.
	if _, loaded := cold.nativeHistogramBucketsNegative.LoadAndDelete(smallestKey); loaded {
		atomicDecUint32(&cold.nativeHistogramBucketsNumber)
	}
	if _, loaded := cold.nativeHistogramBucketsPositive.LoadAndDelete(smallestKey); loaded {
		atomicDecUint32(&cold.nativeHistogramBucketsNumber)
	}
	// Make cold counts the new hot counts.
	n := atomic.AddUint64(&h.countAndHotIdx, 1<<63)
	count := n & ((1 << 63) - 1)
	// Swap the pointer names to represent the new roles and make
	// the rest less confusing.
	hot, cold = cold, hot
	waitForCooldown(count, cold)
	// Add all the now cold counts to the new hot counts...
	addAndResetCounts(hot, cold)
	// ...adjust the new zero threshold in the cold counts, too...
	atomic.StoreUint64(&cold.nativeHistogramZeroThresholdBits, math.Float64bits(newZeroThreshold))
	// ...and then merge the newly deleted buckets into the wider zero
	// bucket.
	mergeAndDeleteOrAddAndReset := func(hotBuckets, coldBuckets *sync.Map) func(k, v interface{}) bool {
		return func(k, v interface{}) bool {
			key := k.(int)
			bucket := v.(*int64)
			if key == smallestKey {
				// Merge into hot zero bucket...
				atomic.AddUint64(&hot.nativeHistogramZeroBucket, uint64(atomic.LoadInt64(bucket)))
				// ...and delete from cold counts.
				coldBuckets.Delete(key)
				atomicDecUint32(&cold.nativeHistogramBucketsNumber)
			} else {
				// Add to corresponding hot bucket...
				if addToBucket(hotBuckets, key, atomic.LoadInt64(bucket)) {
					atomic.AddUint32(&hot.nativeHistogramBucketsNumber, 1)
				}
				// ...and reset cold bucket.
				atomic.StoreInt64(bucket, 0)
			}
			return true
		}
	}
	// Increment count last as we take it as a signal that the observation
	// is complete.
	atomic.AddUint64(&hotCounts.count, 1)
}

// updateExemplar replaces the exemplar for the provided bucket. With empty
// labels, it's a no-op. It panics if any of the labels is invalid.
func (h *histogram) updateExemplar(v float64, bucket int, l Labels) {
	if l == nil {
		return
	}
	e, err := newExemplar(v, h.now(), l)
	if err != nil {
		panic(err)
	}
	h.exemplars[bucket].Store(e)
}

// HistogramVec is a Collector that bundles a set of Histograms that all share the
// same Desc, but have different values for their variable labels. This is used
// if you want to count the same thing partitioned by various dimensions
// (e.g. HTTP request latencies, partitioned by status code and method). Create
// instances with NewHistogramVec.
type HistogramVec struct {
	*MetricVec
}

// NewHistogramVec creates a new HistogramVec based on the provided HistogramOpts and
// partitioned by the given label names.
func NewHistogramVec(opts HistogramOpts, labelNames []string) *HistogramVec {
	desc := NewDesc(
		BuildFQName(opts.Namespace, opts.Subsystem, opts.Name),
		opts.Help,
		labelNames,
		opts.ConstLabels,
	)
	return &HistogramVec{
		MetricVec: NewMetricVec(desc, func(lvs ...string) Metric {
			return newHistogram(desc, opts, lvs...)
		}),
	}
}

// GetMetricWithLabelValues returns the Histogram for the given slice of label
// values (same order as the variable labels in Desc). If that combination of
// label values is accessed for the first time, a new Histogram is created.
//
// It is possible to call this method without using the returned Histogram to only
// create the new Histogram but leave it at its starting value, a Histogram without
// any observations.
//
// Keeping the Histogram for later use is possible (and should be considered if
// performance is critical), but keep in mind that Reset, DeleteLabelValues and
// Delete can be used to delete the Histogram from the HistogramVec. In that case, the
// Histogram will still exist, but it will not be exported anymore, even if a
// Histogram with the same label values is created later. See also the CounterVec
// example.
//
// An error is returned if the number of label values is not the same as the
// number of variable labels in Desc (minus any curried labels).
//
// Note that for more than one label value, this method is prone to mistakes
// caused by an incorrect order of arguments. Consider GetMetricWith(Labels) as
// an alternative to avoid that type of mistake. For higher label numbers, the
// latter has a much more readable (albeit more verbose) syntax, but it comes
// with a performance overhead (for creating and processing the Labels map).
// See also the GaugeVec example.
func (v *HistogramVec) GetMetricWithLabelValues(lvs ...string) (Observer, error) {
	metric, err := v.MetricVec.GetMetricWithLabelValues(lvs...)
	if metric != nil {
		return metric.(Observer), err
	}
	return nil, err
}

// GetMetricWith returns the Histogram for the given Labels map (the label names
// must match those of the variable labels in Desc). If that label map is
// accessed for the first time, a new Histogram is created. Implications of
// creating a Histogram without using it and keeping the Histogram for later use
// are the same as for GetMetricWithLabelValues.
//
// An error is returned if the number and names of the Labels are inconsistent
// with those of the variable labels in Desc (minus any curried labels).
//
// This method is used for the same purpose as
// GetMetricWithLabelValues(...string). See there for pros and cons of the two
// methods.
func (v *HistogramVec) GetMetricWith(labels Labels) (Observer, error) {
	metric, err := v.MetricVec.GetMetricWith(labels)
	if metric != nil {
		return metric.(Observer), err
	}
	return nil, err
}

// WithLabelValues works as GetMetricWithLabelValues, but panics where
// GetMetricWithLabelValues would have returned an error. Not returning an
// error allows shortcuts like
//     myVec.WithLabelValues("404", "GET").Observe(42.21)
func (v *HistogramVec) WithLabelValues(lvs ...string) Observer {
	h, err := v.GetMetricWithLabelValues(lvs...)
	if err != nil {
		panic(err)
	}
	return h
}

// With works as GetMetricWith but panics where GetMetricWithLabels would have
// returned an error. Not returning an error allows shortcuts like
//     myVec.With(prometheus.Labels{"code": "404", "method": "GET"}).Observe(42.21)
func (v *HistogramVec) With(labels Labels) Observer {
	h, err := v.GetMetricWith(labels)
	if err != nil {
		panic(err)
	}
	return h
}

// CurryWith returns a vector curried with the provided labels, i.e. the
// returned vector has those labels pre-set for all labeled operations performed
// on it. The cardinality of the curried vector is reduced accordingly. The
// order of the remaining labels stays the same (just with the curried labels
// taken out of the sequence – which is relevant for the
// (GetMetric)WithLabelValues methods). It is possible to curry a curried
// vector, but only with labels not yet used for currying before.
//
// The metrics contained in the HistogramVec are shared between the curried and
// uncurried vectors. They are just accessed differently. Curried and uncurried
// vectors behave identically in terms of collection. Only one must be
// registered with a given registry (usually the uncurried version). The Reset
// method deletes all metrics, even if called on a curried vector.
func (v *HistogramVec) CurryWith(labels Labels) (ObserverVec, error) {
	vec, err := v.MetricVec.CurryWith(labels)
	if vec != nil {
		return &HistogramVec{vec}, err
	}
	return nil, err
}

// MustCurryWith works as CurryWith but panics where CurryWith would have
// returned an error.
func (v *HistogramVec) MustCurryWith(labels Labels) ObserverVec {
	vec, err := v.CurryWith(labels)
	if err != nil {
		panic(err)
	}
	return vec
}

type constHistogram struct {
	desc       *Desc
	count      uint64
	sum        float64
	buckets    map[float64]uint64
	labelPairs []*dto.LabelPair
}

func (h *constHistogram) Desc() *Desc {
	return h.desc
}

func (h *constHistogram) Write(out *dto.Metric) error {
	his := &dto.Histogram{}
	buckets := make([]*dto.Bucket, 0, len(h.buckets))

	his.SampleCount = proto.Uint64(h.count)
	his.SampleSum = proto.Float64(h.sum)

	for upperBound, count := range h.buckets {
		buckets = append(buckets, &dto.Bucket{
			CumulativeCount: proto.Uint64(count),
			UpperBound:      proto.Float64(upperBound),
		})
	}

	if len(buckets) > 0 {
		sort.Sort(buckSort(buckets))
	}
	his.Bucket = buckets

	out.Histogram = his
	out.Label = h.labelPairs

	return nil
}

// NewConstHistogram returns a metric representing a Prometheus histogram with
// fixed values for the count, sum, and bucket counts. As those parameters
// cannot be changed, the returned value does not implement the Histogram
// interface (but only the Metric interface). Users of this package will not
// have much use for it in regular operations. However, when implementing custom
// Collectors, it is useful as a throw-away metric that is generated on the fly
// to send it to Prometheus in the Collect method.
//
// buckets is a map of upper bounds to cumulative counts, excluding the +Inf
// bucket.
//
// NewConstHistogram returns an error if the length of labelValues is not
// consistent with the variable labels in Desc or if Desc is invalid.
func NewConstHistogram(
	desc *Desc,
	count uint64,
	sum float64,
	buckets map[float64]uint64,
	labelValues ...string,
) (Metric, error) {
	if desc.err != nil {
		return nil, desc.err
	}
	if err := validateLabelValues(labelValues, len(desc.variableLabels)); err != nil {
		return nil, err
	}
	return &constHistogram{
		desc:       desc,
		count:      count,
		sum:        sum,
		buckets:    buckets,
		labelPairs: MakeLabelPairs(desc, labelValues),
	}, nil
}

// MustNewConstHistogram is a version of NewConstHistogram that panics where
// NewConstHistogram would have returned an error.
func MustNewConstHistogram(
	desc *Desc,
	count uint64,
	sum float64,
	buckets map[float64]uint64,
	labelValues ...string,
) Metric {
	m, err := NewConstHistogram(desc, count, sum, buckets, labelValues...)
	if err != nil {
		panic(err)
	}
	return m
}

type buckSort []*dto.Bucket

func (s buckSort) Len() int {
	return len(s)
}

func (s buckSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s buckSort) Less(i, j int) bool {
	return s[i].GetUpperBound() < s[j].GetUpperBound()
}
