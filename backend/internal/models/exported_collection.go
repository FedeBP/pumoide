package models

import "github.com/FedeBP/pumoide/backend/internal/domain"

type ExportedCollection struct {
	Info struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Schema      string `json:"schema"`
	} `json:"info"`
	Item []ExportedItem `json:"item"`
}

type ExportedItem struct {
	Name    string           `json:"name"`
	Request *ExportedRequest `json:"request,omitempty"`
	Item    []ExportedItem   `json:"item,omitempty"`
}

type ExportedRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Header  []domain.Header   `json:"header"`
	Body    map[string]string `json:"body,omitempty"`
	GraphQL *struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	} `json:"graphql,omitempty"`
}

func (c *Collection) ToExportedCollection() ExportedCollection {
	exported := ExportedCollection{}
	exported.Info.Name = c.Name
	exported.Info.Description = c.Description
	exported.Info.Schema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

	exported.Item = convertToExportedItems(c.Items)

	return exported
}

func convertToExportedItems(items []Item) []ExportedItem {
	var exportedItems []ExportedItem
	for _, item := range items {
		exportedItem := ExportedItem{
			Name: item.Name,
		}

		if item.Request != nil {
			req, err := CreateRequestFromJSON(item.Request)
			if err == nil {
				exportedItem.Request = convertToExportedRequest(req)
			}
		} else if item.Folder != nil {
			exportedItem.Item = convertToExportedItems(item.Folder.Items)
		}

		exportedItems = append(exportedItems, exportedItem)
	}
	return exportedItems
}

func convertToExportedRequest(req domain.Request) *ExportedRequest {
	exportedReq := &ExportedRequest{
		URL:    req.GetURL(),
		Header: req.GetHeaders(),
	}

	switch r := req.(type) {
	case *RESTRequest:
		exportedReq.Method = string(r.Method)
		exportedReq.Body = map[string]string{
			"mode": "raw",
			"raw":  r.Body,
		}
	case *WebSocketRequest:
		exportedReq.Method = "websocket"
		exportedReq.Body = map[string]string{
			"mode": "raw",
			"raw":  r.Message,
		}
	case *GraphQLRequest:
		exportedReq.Method = "POST"
		exportedReq.GraphQL = &struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}{
			Query:     r.Query,
			Variables: r.Variables,
		}
	}

	return exportedReq
}
