package search

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo"
)

type Query struct {
	Keyword    string `json:"keyword" query:"keyword"`
	Prefecture string `json:"prefecture" query:"prefecture"`
}

type Result struct {
	Name        string   `json:"name"`
	Prefecture  string   `json:"prefecture"`
	Rulers      []string `json:"rulers"`
	Description string   `json:"description"`
}

type Response struct {
	Message error `json:"message"`
	Results []Result
}

func CastleSearch(c echo.Context) (err error) {
	q := new(Query)
	if err = c.Bind(q); err != nil {
		return
	}

	res := new(Response)
	var (
		b   map[string]interface{}
		buf bytes.Buffer
	)

	query := createQuery(q)

	json.NewEncoder(&buf).Encode(query)

	es, err := connectElasticsearch()
	if err != nil {
		res.Message = err
		return c.JSON(http.StatusInternalServerError, res)
	}

	r, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("castle"),
		es.Search.WithBody(&buf),
		es.Search.WithTrackTotalHits(true),
		es.Search.WithPretty(),
	)
	if err != nil {
		res.Message = err
		return c.JSON(http.StatusInternalServerError, res)
	}
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		res.Message = err
		return c.JSON(http.StatusInternalServerError, res)
	}

	for _, hit := range b["hits"].(map[string]interface{})["hits"].([]interface{}) {
		result := new(Result)
		doc := hit.(map[string]interface{})

		result.Name = doc["_source"].(map[string]interface{})["name"].(string)
		result.Prefecture = doc["_source"].(map[string]interface{})["prefecture"].(string)
		for _, str := range doc["_source"].(map[string]interface{})["rulers"].([]interface{}) {
			result.Rulers = append(result.Rulers, str.(string))
		}
		result.Description = doc["_source"].(map[string]interface{})["description"].(string)

		res.Results = append(res.Results, *result)
	}

	return c.JSON(http.StatusOK, res)
}
