package cloudclient

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// Accepts a Httpaction and a one-way channel to write the results to.
func DoHttpRequest(httpAction HttpAction) ([]byte, error) {
	req := buildHttpRequest(httpAction)
	var DefaultTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: DefaultTransport, Timeout: time.Duration(10 * time.Second)}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("HTTP request failed:", err)
		if resp != nil {
			logger.Error("Response: ", resp.StatusCode)
			logger.Error(resp)
		}
		return nil, err
	}

	if resp != nil {
		defer resp.Body.Close()
		body, err := getBody(resp)
		if err != nil {
			logger.Printf("Error reading response body : %s", err)
			logger.Error("Response body: ", string(body))
			logger.Error("Response: ", resp.StatusCode)
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		} else {
			logger.Error("Response body: ", string(body))
			logger.Error("Response: ", resp.StatusCode)
			return nil, err
		}
	}
	return nil, err
}

func getBody(resp *http.Response) ([]byte, error) {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Unable to read response body")
		return []byte{}, err
	}
	return bodyBytes, nil
}

func buildHttpRequest(httpAction HttpAction) *http.Request {
	var req *http.Request
	var err error
	if httpAction.Body != "" {
		reader := strings.NewReader(httpAction.Body)
		req, err = http.NewRequest(httpAction.Method, httpAction.Url, reader)
	} else if httpAction.Template != "" {
		reader := strings.NewReader(httpAction.Template)
		req, err = http.NewRequest(httpAction.Method, httpAction.Url, reader)
	} else {
		req, err = http.NewRequest(httpAction.Method, httpAction.Url, nil)
	}
	if err != nil {
		logger.Error(err)
	}

	// Add headers
	req.Header.Add("Accept", httpAction.Accept)
	if httpAction.ContentType != "" {
		req.Header.Add("Content-Type", httpAction.ContentType)
	}
	return req
}
