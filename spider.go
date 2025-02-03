package scavenge

type Spider interface {
	StartingRequests() []Request
	HandleResponse(navigator Navigator, response Response) error
}
