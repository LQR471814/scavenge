package items

import "reflect"

// Item represents some information retrieved by a spider that may or may not have gone through processing
// in item pipelines.
//
// The item itself is simply an array of anys that go from first-added to last-added. This is because
// item pipelines often need to add information to a given item, so that information is kept in the anys
// that come after the first one (the one returned by the spider).
//
// To write information to an item, call Add which will return an item with your any added.
//
// To read information from an item, call CastItem with the type of the value you wish to find in the array.
// For item pipelines that need to read generic information, it is better to export an interface that any
// struct can implement. CastItem will find the first added struct that makes the type cast successful.
//
// Notes:
//
//   - Item is supposed to be immutable, that means you should not do something like: `item[0] = ...`
//   - All values in an Item should be serializable with [encoding/gob], as that is the encoding used
//     to store scraping state when pausing and resuming a scraping run.
type Item []any

// Add creates a new item with the value appended to its entries.
func (i Item) Add(value any) Item {
	entries := append(i, value)
	return entries
}

// Clone returns a shallow-copy of items.
func (i Item) Clone() Item {
	var out Item
	copy(i, out)
	return out
}

// CastItem finds the first struct based on the following behavior:
//
//   - If the given type, T is an interface, find the first value that fulfills its interface.
//   - If the given type, T is a concrete type, find the first value that has exactly the same tyep as T.
func CastItem[T any](item Item) (T, bool) {
	var tmp T

	// cannot directly use TypeOf(tmp) since tmp may be a nil interface which will cause reflect.TypeOf to return nil
	if reflect.TypeOf((*T)(nil)).Elem().Kind() == reflect.Interface {
		for _, e := range item {
			cast, ok := e.(T)
			if ok {
				return cast, true
			}
		}
		return tmp, false
	}

	// in contrast, if tmp is certainly not an interface, TypeOf(tmp) will always return the type
	// even if tmp is nil
	t := reflect.TypeOf(tmp)
	for _, e := range item {
		if reflect.TypeOf(e) == t {
			return e.(T), true
		}
	}
	return tmp, false
}

// CastAllItems finds all the structs fulfilling the conditions in Castitems.
func CastAllItems[T any](item Item) []T {
	var tmp T

	// cannot directly use TypeOf(tmp) since tmp may be a nil interface which will cause reflect.TypeOf to return nil
	if reflect.TypeOf((*T)(nil)).Elem().Kind() == reflect.Interface {
		var results []T
		for _, e := range item {
			cast, ok := e.(T)
			if ok {
				results = append(results, cast)
			}
		}
		return results
	}

	// in contrast, if tmp is certainly not an interface, TypeOf(tmp) will always return the type
	// even if tmp is nil
	t := reflect.TypeOf(tmp)
	var results []T
	for _, e := range item {
		if reflect.TypeOf(e) == t {
			results = append(results, e.(T))
		}
	}
	return results
}
