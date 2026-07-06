package dmp

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
)

const (
	baseUrl         = "http://platform.debezium.local"
	connUrl         = "/api/connections"
	connValidateUrl = "/api/connections/validate"
	sourceUrl       = "/api/sources"
	destUrl         = "/api/destinations"
	pipelineUrl     = "/api/pipelines"
	transformUrl    = "/api/transforms"
)

type HTTPClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewHTTPClient() *HTTPClient {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &HTTPClient{
		BaseURL:    baseUrl,
		HTTPClient: client,
	}
}

func (c *HTTPClient) doRequest(method string, resourceType ResourceType, body []byte) (*http.Response, error) {
	var endpoint string
	switch resourceType {
	case ConnectionType:
		endpoint = connUrl
	case SourceType:
		endpoint = sourceUrl
	case DestinationType:
		endpoint = destUrl
	case PipelineType:
		endpoint = pipelineUrl
	case TransformType:
		endpoint = transformUrl
	default:
		return nil, fmt.Errorf("unsupported resource type: %v", resourceType)
	}

	return c.doEndpointRequest(method, endpoint, body)
}

func (c *HTTPClient) doEndpointRequest(method string, endpoint string, body []byte) (*http.Response, error) {
	url := c.BaseURL + endpoint

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

func (c *HTTPClient) Post(resourceType ResourceType, body []byte) (*http.Response, error) {
	return c.doRequest("POST", resourceType, body)
}

func (c *HTTPClient) Put(resourceType ResourceType, body []byte) (*http.Response, error) {
	return c.doRequest("PUT", resourceType, body)
}

func (c *HTTPClient) Get(resourceType ResourceType) (*http.Response, error) {
	return c.doRequest("GET", resourceType, nil)
}

func (c *HTTPClient) Delete(resourceType ResourceType) (*http.Response, error) {
	return c.doRequest("DELETE", resourceType, nil)
}

func (c *HTTPClient) ValidateConnection(body []byte) (*http.Response, error) {
	return c.doEndpointRequest("POST", connValidateUrl, body)
}
