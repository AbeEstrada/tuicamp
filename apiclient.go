package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type APIClient struct {
	BaseURL     string
	HTTPClient  *http.Client
	dedupeMutex sync.Mutex
	inFlight    map[string]*dedupeRequest
}

type dedupeRequest struct {
	wg       sync.WaitGroup
	response AsyncResponse
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		inFlight: make(map[string]*dedupeRequest),
	}
}

type AsyncResponse struct {
	Response any
	Error    error
}

type CallOptions struct {
	Endpoint    string
	Method      string
	RequestBody any
	Response    any
	Headers     map[string]string
}

func (c *APIClient) generateRequestKey(opts CallOptions) (string, error) {
	key := opts.Method + ":" + opts.Endpoint

	if opts.RequestBody != nil {
		jsonData, err := json.Marshal(opts.RequestBody)
		if err != nil {
			return "", fmt.Errorf("error marshaling request body for dedupe key: %w", err)
		}
		key += ":" + string(jsonData)
	}

	for k, v := range opts.Headers {
		key += ":" + k + "=" + v
	}

	return key, nil
}

func (c *APIClient) executeRequest(opts CallOptions) AsyncResponse {
	key, err := c.generateRequestKey(opts)
	if err != nil {
		return AsyncResponse{Error: err}
	}

	c.dedupeMutex.Lock()
	if dr, exists := c.inFlight[key]; exists {
		c.dedupeMutex.Unlock()
		dr.wg.Wait() // Wait for the existing request to complete
		return dr.response
	}

	dr := &dedupeRequest{}
	dr.wg.Add(1)
	c.inFlight[key] = dr
	c.dedupeMutex.Unlock()

	dr.response = c.doRequest(opts)

	c.dedupeMutex.Lock()
	delete(c.inFlight, key)
	c.dedupeMutex.Unlock()
	dr.wg.Done()

	return dr.response
}

func (c *APIClient) doRequest(opts CallOptions) AsyncResponse {
	url := c.BaseURL + opts.Endpoint

	var body io.Reader
	if opts.RequestBody != nil {
		jsonData, err := json.Marshal(opts.RequestBody)
		if err != nil {
			return AsyncResponse{Error: fmt.Errorf("error marshaling request body: %w", err)}
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(opts.Method, url, body)
	if err != nil {
		return AsyncResponse{Error: fmt.Errorf("error creating request: %w", err)}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return AsyncResponse{Error: fmt.Errorf("error making request: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		errorMsg := string(bodyBytes)
		if readErr != nil {
			errorMsg = "couldn't read response body"
		}
		return AsyncResponse{Error: fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, errorMsg)}
	}

	if opts.Response != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return AsyncResponse{Error: fmt.Errorf("error reading response body: %w", err)}
		}
		if err := json.Unmarshal(bodyBytes, opts.Response); err != nil {
			return AsyncResponse{Error: fmt.Errorf("error decoding response JSON: %w", err)}
		}
	}

	return AsyncResponse{Response: opts.Response}
}

func (c *APIClient) CallAsyncWithChannel(opts CallOptions) <-chan AsyncResponse {
	resultChan := make(chan AsyncResponse, 1)
	go func() {
		resultChan <- c.executeRequest(opts)
		close(resultChan)
	}()
	return resultChan
}

func (c *APIClient) CallAsyncWithCallback(opts CallOptions, callback func(AsyncResponse)) {
	go func() {
		callback(c.executeRequest(opts))
	}()
}

func (c *APIClient) Call(
	endpoint string,
	method string,
	requestBody any,
	response any,
	headers map[string]string,
) error {
	opts := CallOptions{
		Endpoint:    endpoint,
		Method:      method,
		RequestBody: requestBody,
		Response:    response,
		Headers:     headers,
	}
	return c.executeRequest(opts).Error
}
