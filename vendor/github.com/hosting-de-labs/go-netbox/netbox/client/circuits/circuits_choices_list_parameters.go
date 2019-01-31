// Code generated by go-swagger; DO NOT EDIT.

// Copyright 2018 The go-netbox Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package circuits

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"

	strfmt "github.com/go-openapi/strfmt"
)

// NewCircuitsChoicesListParams creates a new CircuitsChoicesListParams object
// with the default values initialized.
func NewCircuitsChoicesListParams() *CircuitsChoicesListParams {

	return &CircuitsChoicesListParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewCircuitsChoicesListParamsWithTimeout creates a new CircuitsChoicesListParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewCircuitsChoicesListParamsWithTimeout(timeout time.Duration) *CircuitsChoicesListParams {

	return &CircuitsChoicesListParams{

		timeout: timeout,
	}
}

// NewCircuitsChoicesListParamsWithContext creates a new CircuitsChoicesListParams object
// with the default values initialized, and the ability to set a context for a request
func NewCircuitsChoicesListParamsWithContext(ctx context.Context) *CircuitsChoicesListParams {

	return &CircuitsChoicesListParams{

		Context: ctx,
	}
}

// NewCircuitsChoicesListParamsWithHTTPClient creates a new CircuitsChoicesListParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewCircuitsChoicesListParamsWithHTTPClient(client *http.Client) *CircuitsChoicesListParams {

	return &CircuitsChoicesListParams{
		HTTPClient: client,
	}
}

/*CircuitsChoicesListParams contains all the parameters to send to the API endpoint
for the circuits choices list operation typically these are written to a http.Request
*/
type CircuitsChoicesListParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the circuits choices list params
func (o *CircuitsChoicesListParams) WithTimeout(timeout time.Duration) *CircuitsChoicesListParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the circuits choices list params
func (o *CircuitsChoicesListParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the circuits choices list params
func (o *CircuitsChoicesListParams) WithContext(ctx context.Context) *CircuitsChoicesListParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the circuits choices list params
func (o *CircuitsChoicesListParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the circuits choices list params
func (o *CircuitsChoicesListParams) WithHTTPClient(client *http.Client) *CircuitsChoicesListParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the circuits choices list params
func (o *CircuitsChoicesListParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *CircuitsChoicesListParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
