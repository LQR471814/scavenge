package pipelines

import (
	"encoding/json"
	"io"
	"scavenge/item"
)

// ExportJson is an item pipeline that exports items in a json format to the specified io.Writer.
type ExportJson struct {
	output io.Writer
}

func NewExportJson(output io.Writer) ExportJson {
	return ExportJson{output: output}
}

func (e ExportJson) HandleItem(item item.Item) (item.Item, error) {
	marshalled, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}
	_, err = e.output.Write(marshalled)
	if err != nil {
		return nil, err
	}
	_, err = e.output.Write([]byte("\n"))
	if err != nil {
		return nil, err
	}
	return item, nil
}
