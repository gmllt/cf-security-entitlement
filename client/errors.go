package client

//go:generate go run gen_error.go

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

type CloudFoundryError struct {
	Code        int    `json:"code"`
	ErrorCode   string `json:"error_code"`
	Description string `json:"description"`
}

type CloudFoundryErrorsV3 struct {
	Errors []CloudFoundryErrorV3 `json:"errors"`
}

type CloudFoundryErrorV3 struct {
	Code   int    `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// NewCloudFoundryErrorFromV3Errors CF APIs v3 can return multiple errors, we take the first one and convert it into a V2 model
func NewCloudFoundryErrorFromV3Errors(cfErrorsV3 CloudFoundryErrorsV3) CloudFoundryErrorV3 {
	if len(cfErrorsV3.Errors) == 0 {
		return CloudFoundryErrorV3{
			0,
			"GO-Client-No-Errors",
			"No Errors in response from V3",
		}
	}

	return CloudFoundryErrorV3{
		cfErrorsV3.Errors[0].Code,
		cfErrorsV3.Errors[0].Title,
		cfErrorsV3.Errors[0].Detail,
	}
}

func (cfErr CloudFoundryError) Error() string {
	return fmt.Sprintf("cfclient error (%s|%d): %s", cfErr.ErrorCode, cfErr.Code, cfErr.Description)
}

func (cfErrV3 CloudFoundryErrorV3) Error() string {
	return fmt.Sprintf("cfclient error (%d|%s): %s", cfErrV3.Code, cfErrV3.Title, cfErrV3.Detail)
}

type CloudFoundryHTTPError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e CloudFoundryHTTPError) Error() string {
	return fmt.Sprintf("cfclient: HTTP error (%d): %s", e.StatusCode, e.Status)
}

func (c *Client) handleError(resp *http.Response) (*http.Response, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, CloudFoundryHTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(body),
		}
	}

	defer resp.Body.Close()

	var cfErrorsV3 CloudFoundryErrorsV3
	var cfErrorV3 CloudFoundryErrorV3
	if err := json.Unmarshal(body, &cfErrorsV3); err != nil {
		return resp, errors.Wrap(err, "Error unmarshalling Errors")
	}

	if len(cfErrorsV3.Errors) == 0 {
		if err := json.Unmarshal(body, &cfErrorV3); err != nil {
			return resp, errors.Wrap(err, "Error unmarshalling Errors")
		}
	} else {
		cfErrorV3 = NewCloudFoundryErrorFromV3Errors(cfErrorsV3)
	}

	return nil, cfErrorV3
}
