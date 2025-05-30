// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package dynamicconfig

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/mitchellh/mapstructure"

	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/primitives/timestamp"
)

type (
	// Collection implements lookup and constraint logic on top of a Client.
	// The rest of the server code should use Collection as the interface to dynamic config,
	// instead of the low-level Client.
	Collection struct {
		client   Client
		logger   log.Logger
		errCount int64
	}

	// These function types follow a similar pattern:
	//   {X}PropertyFn - returns a value of type X that is global (no filters)
	//   {X}PropertyFnWith{Y}Filter - returns a value of type X with the given filters
	// Available value types:
	//   Bool: bool
	//   Duration: time.Duration
	//   Float: float64
	//   Int: int
	//   Map: map[string]any
	//   String: string
	// Available filters:
	//   Namespace func(namespace string)
	//   NamespaceID func(namespaceID string)
	//   TaskQueue func(namespace string, taskQueue string, taskType enumspb.TaskQueueType)  (matching task queue)
	//   TaskType func(taskType enumspsb.TaskType)  (history task type)
	//   ShardID func(shardID int32)
)

const (
	errCountLogThreshold = 1000
)

var (
	errKeyNotPresent        = errors.New("key not present")
	errNoMatchingConstraint = errors.New("no matching constraint in key")
)

// NewCollection creates a new collection
func NewCollection(client Client, logger log.Logger) *Collection {
	return &Collection{
		client:   client,
		logger:   logger,
		errCount: -1,
	}
}

func (c *Collection) throttleLog() bool {
	// TODO: This is a lot of unnecessary contention with little benefit. Consider using
	// https://github.com/cespare/percpu here.
	errCount := atomic.AddInt64(&c.errCount, 1)
	// log only the first x errors and then one every x after that to reduce log noise
	return errCount < errCountLogThreshold || errCount%errCountLogThreshold == 0
}

func (c *Collection) HasKey(key Key) bool {
	cvs := c.client.GetValue(key)
	return len(cvs) > 0
}

func findMatch[T any](cvs []ConstrainedValue, defaultCVs []TypedConstrainedValue[T], precedence []Constraints) (any, error) {
	if len(cvs)+len(defaultCVs) == 0 {
		return nil, errKeyNotPresent
	}
	for _, m := range precedence {
		for _, cv := range cvs {
			if m == cv.Constraints {
				return cv.Value, nil
			}
		}
		for _, cv := range defaultCVs {
			if m == cv.Constraints {
				return cv.Value, nil
			}
		}
	}
	// key is present but no constraint section matches
	return nil, errNoMatchingConstraint
}

// matchAndConvert can't be a method of Collection because methods can't be generic, but we can
// take a *Collection as an argument.
func matchAndConvert[T any](
	c *Collection,
	key Key,
	def T,
	cdef []TypedConstrainedValue[T],
	convert func(value any) (T, error),
	precedence []Constraints,
) T {
	cvs := c.client.GetValue(key)

	defaultCVs := cdef
	if defaultCVs == nil {
		defaultCVs = []TypedConstrainedValue[T]{{Value: def}}
	}

	val, matchErr := findMatch(cvs, defaultCVs, precedence)
	if matchErr != nil {
		if c.throttleLog() {
			c.logger.Debug("No such key in dynamic config, using default", tag.Key(key.String()), tag.Error(matchErr))
		}
		// couldn't find a constrained match, use default
		val = def
	}

	typedVal, convertErr := convert(val)
	if convertErr != nil && matchErr == nil {
		// We failed to convert the value to the desired type. Try converting the default. note
		// that if matchErr != nil then val _is_ defaultValue and we don't have to try this again.
		if c.throttleLog() {
			c.logger.Warn("Failed to convert value, using default", tag.Key(key.String()), tag.IgnoredValue(val), tag.Error(convertErr))
		}
		typedVal, convertErr = convert(def)
	}
	if convertErr != nil {
		// If we can't convert the default, that's a bug in our code, use Warn level.
		c.logger.Warn("Can't convert default value (this is a bug; fix server code)", tag.Key(key.String()), tag.IgnoredValue(def), tag.Error(convertErr))
		// Return typedVal anyway since we have to return something.
	}
	return typedVal
}

func convertInt(val any) (int, error) {
	switch val := val.(type) {
	case int:
		return int(val), nil
	case int8:
		return int(val), nil
	case int16:
		return int(val), nil
	case int32:
		return int(val), nil
	case int64:
		return int(val), nil
	case uint:
		return int(val), nil
	case uint8:
		return int(val), nil
	case uint16:
		return int(val), nil
	case uint32:
		return int(val), nil
	case uint64:
		return int(val), nil
	case uintptr:
		return int(val), nil
	default:
		return 0, errors.New("value type is not int")
	}
}

func convertFloat(val any) (float64, error) {
	switch val := val.(type) {
	case float32:
		return float64(val), nil
	case float64:
		return float64(val), nil
	}
	if ival, err := convertInt(val); err == nil {
		return float64(ival), nil
	}
	return 0, errors.New("value type is not float64")
}

func convertDuration(val any) (time.Duration, error) {
	switch v := val.(type) {
	case time.Duration:
		return v, nil
	case string:
		d, err := timestamp.ParseDurationDefaultSeconds(v)
		if err != nil {
			return 0, fmt.Errorf("failed to parse duration: %v", err)
		}
		return d, nil
	}
	// treat numeric values as seconds
	if ival, err := convertInt(val); err == nil {
		return time.Duration(ival) * time.Second, nil
	} else if fval, err := convertFloat(val); err == nil {
		return time.Duration(fval * float64(time.Second)), nil
	}
	return 0, errors.New("value not convertible to Duration")
}

func convertString(val any) (string, error) {
	if stringVal, ok := val.(string); ok {
		return stringVal, nil
	}
	return "", errors.New("value type is not string")
}

func convertBool(val any) (bool, error) {
	switch v := val.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, errors.New("value type is not bool")
	}
}

func convertMap(val any) (map[string]any, error) {
	if mapVal, ok := val.(map[string]any); ok {
		return mapVal, nil
	}
	return nil, errors.New("value type is not map")
}

// ConvertStructure can be used as a conversion function for New*TypedSettingWithConverter.
// The value from dynamic config will be converted to T, on top of the given default.
//
// Note that any failure in conversion of _any_ field will result in the overall default being used,
// ignoring the fields that successfully converted.
//
// Note that the default value will be shallow-copied, so it should not have any deep structure.
// Scalar types and values are fine, and slice and map types are fine too as long as they're set to
// nil in the default.
//
// To avoid confusion, the default passed to ConvertStructure should be either the same as the
// overall default for the setting (if you want any value set to be merged over the default, i.e.
// treat the fields independently), or the zero value of its type (if you want to treat the fields
// as a group and default unset fields to zero).
func ConvertStructure[T any](def T) func(v any) (T, error) {
	return func(v any) (T, error) {
		// if we already have the right type, no conversion is necessary
		if typedV, ok := v.(T); ok {
			return typedV, nil
		}

		out := def
		dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result: &out,
			// If we want more than one hook in the future, combine them with mapstructure.OrComposeDecodeHookFunc
			DecodeHook: mapstructureHookDuration,
		})
		if err != nil {
			return out, err
		}
		err = dec.Decode(v)
		return out, err
	}
}

// Parses string into time.Duration. mapstructure has an implementation of this already but it
// calls time.ParseDuration and we want to use our own method.
func mapstructureHookDuration(f, t reflect.Type, data any) (any, error) {
	if t != reflect.TypeOf(time.Duration(0)) {
		return data, nil
	}
	return convertDuration(data)
}
