package models

import (
	"encoding/json"
	"fmt"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/google/uuid"
)

type ImportedCollection struct {
	Info struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"info"`
	Item []ImportedItem `json:"item"`
}

type ImportedItem struct {
	Name    string           `json:"name"`
	Request *ImportedRequest `json:"request,omitempty"`
	Item    []ImportedItem   `json:"item,omitempty"`
}

type ImportedRequest struct {
	Method string          `json:"method"`
	URL    interface{}     `json:"url"`
	Header []domain.Header `json:"header"`
	Body   struct {
		Mode string `json:"mode"`
		Raw  string `json:"raw"`
	} `json:"body"`
	GraphQL *struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	} `json:"graphql,omitempty"`
}

func NewCollectionFromImported(imported ImportedCollection) (*Collection, error) {
	newCollection := &Collection{
		ID:          uuid.New().String(),
		Name:        imported.Info.Name,
		Description: imported.Info.Description,
	}

	newCollection.Items = convertImportedItems(imported.Item)

	if err := newCollection.Validate(); err != nil {
		return nil, err
	}

	return newCollection, nil
}

func convertImportedItems(importedItems []ImportedItem) []Item {
	var items []Item
	for _, importedItem := range importedItems {
		item := Item{
			ID:   uuid.New().String(),
			Name: importedItem.Name,
		}

		if importedItem.Request != nil {
			request, err := createRequestFromImported(importedItem.Name, importedItem.Request)
			if err == nil {
				requestJSON, _ := json.Marshal(request)
				item.Request = requestJSON
			}
		} else if len(importedItem.Item) > 0 {
			item.Folder = &Folder{
				ID:    uuid.New().String(),
				Name:  importedItem.Name,
				Items: convertImportedItems(importedItem.Item),
			}
		}

		items = append(items, item)
	}
	return items
}

func createRequestFromImported(name string, importedReq *ImportedRequest) (domain.Request, error) {
	if importedReq == nil {
		return nil, fmt.Errorf("empty request")
	}

	if importedReq.GraphQL != nil {
		return createGraphQLRequestFromImported(name, importedReq)
	}

	method := domain.Method(importedReq.Method)
	if !method.Validate() {
		return nil, fmt.Errorf("invalid HTTP method: %s", importedReq.Method)
	}

	url := constants.EmptyString
	switch u := importedReq.URL.(type) {
	case string:
		url = u
	case map[string]interface{}:
		if raw, ok := u["raw"].(string); ok {
			url = raw
		} else {
			return nil, fmt.Errorf("invalid URL format")
		}
	default:
		return nil, fmt.Errorf("unsupported URL format")
	}

	restReq := &RESTRequest{
		ID:      uuid.New().String(),
		Name:    name,
		Type:    domain.RequestTypeREST,
		Method:  method,
		URL:     url,
		Headers: importedReq.Header,
	}

	if importedReq.Body.Mode == "raw" {
		restReq.Body = importedReq.Body.Raw
	}

	return restReq, nil
}

func createGraphQLRequestFromImported(name string, importedReq *ImportedRequest) (domain.Request, error) {
	graphqlReq := &GraphQLRequest{
		ID:        uuid.New().String(),
		Name:      name,
		Type:      domain.RequestTypeGraphQL,
		URL:       importedReq.URL.(string),
		Query:     importedReq.GraphQL.Query,
		Variables: importedReq.GraphQL.Variables,
		Headers:   importedReq.Header,
	}

	return graphqlReq, nil
}
