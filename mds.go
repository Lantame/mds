package mds

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// UploadInfo describes result of upload
type UploadInfo struct {
	XMLName xml.Name `xml:"post"`
	Obj     string   `xml:"obj,attr"`
	ID      string   `xml:"id,attr"`
	Key     string   `xml:"key,attr"`
	Size    uint64   `xml:"size,attr"`
	Groups  int      `xml:"groups,attr"`

	Complete []struct {
		Addr   string `xml:"addr,attr"`
		Path   string `xml:"path,attr"`
		Group  int    `xml:"group,attr"`
		Status int    `xml:"status,attr"`
	} `xml:"complete"`

	Written int `xml:"written"`
}

func decodeUploadInfo(result *UploadInfo, body io.Reader) error {
	return xml.NewDecoder(body).Decode(result)
}

// Config represents configuration for the client
type Config struct {
	Host       string
	UploadPort int
	ReadPort   int

	AuthHeader string
}

// Client works with MDS
type Client struct {
	Config
}

// NewClient creates a client to MDS
func NewClient(config Config) (*Client, error) {
	return &Client{
		Config: config,
	}, nil
}

func (m *Client) uploadURL(namespace, filename string) string {
	return fmt.Sprintf("http://%s:%d/upload-%s/%s", m.Host, m.UploadPort, namespace, filename)
}

func (m *Client) readURL(namespace, filename string) string {
	return fmt.Sprintf("http://%s:%d/get-%s/%s", m.Host, m.ReadPort, namespace, filename)
}

// Upload stores provided data to a specified namespace. Returns information about upload.
func (m *Client) Upload(namespace string, filename string, body io.ReadCloser) (*UploadInfo, error) {
	urlStr := m.uploadURL(namespace, filename)
	req, err := http.NewRequest("POST", urlStr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", m.AuthHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("[%s]", resp.Status)
	}

	var info UploadInfo
	if err := decodeUploadInfo(&info, resp.Body); err != nil {
		return nil, err
	}

	return &info, nil
}

// Get reads a given key from storage and return ReadCloser to body.
// User is repsonsible for closing returned ReadCloser
func (m *Client) Get(namespace, key string) (io.ReadCloser, error) {
	urlStr := m.readURL(namespace, key)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", m.AuthHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return resp.Body, nil
	default:
		return nil, fmt.Errorf("[%s]", resp.Status)
	}
}

// GetFile like Get but returns bytes
func (m *Client) GetFile(namespace, key string) ([]byte, error) {
	output, err := m.Get(namespace, key)
	if err != nil {
		return nil, err
	}
	defer output.Close()

	return ioutil.ReadAll(output)
}
