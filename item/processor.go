package item

import "fmt"

// Pipeline is effectively middleware for items.
type Pipeline interface {
	HandleItem(item Item) (Item, error)
}

// Processor runs a list of item pipelines over the items given as input to it.
type Processor struct {
	pipelines []Pipeline
}

func NewProcessor(pipelines ...Pipeline) Processor {
	return Processor{pipelines: pipelines}
}

func (p Processor) Process(item Item) (Item, error) {
	var err error
	for _, p := range p.pipelines {
		item, err = p.HandleItem(item)
		if err != nil {
			return nil, fmt.Errorf("pipeline: %w", err)
		}
	}
	return item, nil
}
