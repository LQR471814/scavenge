package scavenge

// Item represents some information retrieved by a spider that may or may not have gone through processing
// in item pipelines.
//
// The item itself is simply an array of [any]s that go from first-added to last-added. This is because
// item pipelines often need to add information to a given item, so that information is kept in the [any]s
// that come after the first one (the one returned by the spider).
//
// To write information to an item, call [Add] which will return an item with your [any] added.
//
// To read information from an item, call [CastItem] with the type of the value you wish to find in the array.
// For item pipelines that need to read generic information, it is better to export an interface that any
// struct can implement. [CastItem] will find the first added struct that makes the type cast successful.
type Item []any

// Add creates a new item with the value appended to its entries.
func (i Item) Add(value any) Item {
	entries := append(i, value)
	return entries
}

// Entries returns a copy of the entries held within the item.
func (i Item) Entries() []any {
	var out []any
	copy(i, out)
	return out
}

// CastItem finds the first struct in the item's entries that can be cast to the generic type given.
func CastItem[T any](i Item) (T, bool) {
	for _, e := range i {
		cast, ok := e.(T)
		if ok {
			return cast, true
		}
	}
	var tmp T
	return tmp, false
}
