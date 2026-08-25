// Package client is a thin wrapper around the Stackryze DNS REST API.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	if baseURL == "" {
		baseURL = "https://api.stackryze.com/api"
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 20 * time.Second},
	}
}

type apiError struct {
	Error string `json:"error"`
}

func (c *Client) do(method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)

	if res.StatusCode >= 400 {
		var e apiError
		_ = json.Unmarshal(data, &e)
		if e.Error != "" {
			return fmt.Errorf("stackryze api %s %s: %s", method, path, e.Error)
		}
		return fmt.Errorf("stackryze api %s %s: %s", method, path, res.Status)
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

// Zone as returned by the API.
type Zone struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
}

type zonesResponse struct {
	Zones []Zone `json:"zones"`
}

// RRSet mirrors the PowerDNS-style record grouping the API returns.
type RRSet struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	TTL     int    `json:"ttl"`
	Records []struct {
		Content  string `json:"content"`
		Disabled bool   `json:"disabled"`
	} `json:"records"`
}

func strip(s string) string { return strings.TrimSuffix(s, ".") }

// GetZoneByName finds a zone id by its name.
func (c *Client) GetZoneByName(name string) (*Zone, error) {
	var resp zonesResponse
	if err := c.do("GET", "/zones", nil, &resp); err != nil {
		return nil, err
	}
	name = strip(strings.ToLower(name))
	for i := range resp.Zones {
		if strip(strings.ToLower(resp.Zones[i].Name)) == name {
			return &resp.Zones[i], nil
		}
	}
	return nil, fmt.Errorf("zone %q not found (create it in Stackryze first)", name)
}

// Record is a single flattened DNS record.
type Record struct {
	Name    string
	Type    string
	Content string
	TTL     int
}

// FindRecord returns the matching record within a zone, or nil if absent.
func (c *Client) FindRecord(zoneID, name, recordType, content string) (*Record, error) {
	var rrsets []RRSet
	if err := c.do("GET", "/zones/"+zoneID+"/records?max=1000", nil, &rrsets); err != nil {
		return nil, err
	}
	want := strip(strings.ToLower(name))
	for _, rr := range rrsets {
		if strip(strings.ToLower(rr.Name)) != want || rr.Type != recordType {
			continue
		}
		for _, r := range rr.Records {
			if strip(r.Content) == strip(content) {
				return &Record{Name: name, Type: recordType, Content: content, TTL: rr.TTL}, nil
			}
		}
	}
	return nil, nil
}

type recordPayload struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

func (c *Client) AddRecord(zoneID string, r Record) error {
	return c.do("POST", "/zones/"+zoneID+"/records", recordPayload{Type: r.Type, Name: r.Name, Content: r.Content, TTL: r.TTL}, nil)
}

type deletePayload struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (c *Client) DeleteRecord(zoneID string, r Record) error {
	return c.do("DELETE", "/zones/"+zoneID+"/records", deletePayload{Type: r.Type, Name: r.Name, Content: r.Content}, nil)
}
