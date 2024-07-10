package fornex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

var (
	Uri       = "https://fornex.com/"
	parsedURL *url.URL

	HttpClient = &http.Client{Timeout: time.Second * 10}
)

func init() {
	var err error
	parsedURL, err = url.Parse(Uri)
	if err != nil {
		panic("Failed to parse FornexAPIURI: " + err.Error())
	}
}

type Client struct {
	apiKey string
}

func New(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}

func (c *Client) RetrieveRecords(ctx context.Context, domain string) ([]Record, error) {
	req, err := c.getRequest(ctx, fmt.Sprintf("/api/dns/domain/%s/entry_set/", domain))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	resp, err := HttpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var records []Record
	err = json.NewDecoder(resp.Body).Decode(&records)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode response")
	}
	return records, nil
}

func (c *Client) CreateRecord(ctx context.Context, domain string, record Record) (int, error) {
	body, err := json.Marshal(record)
	if err != nil {
		return 0, errors.Wrap(err, "failed to marshal record")
	}

	req, err := c.postRequest(ctx, fmt.Sprintf("/api/dns/domain/%s/entry_set/", domain), body)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create request")
	}

	resp, err := HttpClient.Do(req)
	if err != nil {
		return 0, errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var respString string
		if resp.Header.Get("Content-Type") == "application/json" {
			errorRespBytes, _ := io.ReadAll(resp.Body)
			respString = string(errorRespBytes)
		}

		return 0, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, respString)
	}

	dec := json.NewDecoder(resp.Body)

	var responseRecord Record
	if err = dec.Decode(&responseRecord); err != nil {
		return 0, errors.Wrap(err, "failed to decode response")
	}

	return responseRecord.ID, nil
}

func (c *Client) DeleteRecord(ctx context.Context, domain string, recordID int) error {
	req, err := c.deleteRequest(ctx, fmt.Sprintf("/api/dns/domain/%s/entry_set/%d", domain, recordID))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	resp, err := HttpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) getRequest(ctx context.Context, path string) (*http.Request, error) {
	u := *parsedURL
	u.Path = path

	klog.Infof("GET %s", u.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Api-Key %s", c.apiKey))

	return req, nil
}

func (c *Client) postRequest(ctx context.Context, path string, body []byte) (*http.Request, error) {
	u := *parsedURL
	u.Path = path

	b := bytes.NewReader(body)

	klog.Infof("POST %s: %s", u.String(), string(body))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	req.Header.Set("Authorization", fmt.Sprintf("Api-Key %s", c.apiKey))

	return req, nil
}

func (c *Client) deleteRequest(ctx context.Context, path string) (*http.Request, error) {
	u := *parsedURL
	u.Path = path

	klog.Infof("DELETE %s", u.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Api-Key %s", c.apiKey))

	return req, nil
}
