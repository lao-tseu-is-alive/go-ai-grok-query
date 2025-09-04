package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HttpRequest performs a generic HTTP POST request and unmarshals the response.
// It's designed to be used by providers that don't follow the OpenAI API schema.
func HttpRequest[ReqT any, RespT any](
	ctx context.Context,
	client *http.Client,
	url string,
	headers http.Header,
	requestBody ReqT,
) (*RespT, []byte, error) {

	// 1. Marshal the request body
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// 2. Create and configure the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create new request: %w", err)
	}
	httpReq.Header = headers

	// 3. Execute the request
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 4. Read and check the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, respBody, fmt.Errorf("received non-2xx status code %d", resp.StatusCode)
	}

	// 5. Unmarshal the successful response
	var responsePayload RespT
	if err := json.Unmarshal(respBody, &responsePayload); err != nil {
		return nil, respBody, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &responsePayload, respBody, nil
}
