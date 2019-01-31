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
	"github.com/go-openapi/swag"

	strfmt "github.com/go-openapi/strfmt"
)

// NewCircuitsProvidersListParams creates a new CircuitsProvidersListParams object
// with the default values initialized.
func NewCircuitsProvidersListParams() *CircuitsProvidersListParams {
	var ()
	return &CircuitsProvidersListParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewCircuitsProvidersListParamsWithTimeout creates a new CircuitsProvidersListParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewCircuitsProvidersListParamsWithTimeout(timeout time.Duration) *CircuitsProvidersListParams {
	var ()
	return &CircuitsProvidersListParams{

		timeout: timeout,
	}
}

// NewCircuitsProvidersListParamsWithContext creates a new CircuitsProvidersListParams object
// with the default values initialized, and the ability to set a context for a request
func NewCircuitsProvidersListParamsWithContext(ctx context.Context) *CircuitsProvidersListParams {
	var ()
	return &CircuitsProvidersListParams{

		Context: ctx,
	}
}

// NewCircuitsProvidersListParamsWithHTTPClient creates a new CircuitsProvidersListParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewCircuitsProvidersListParamsWithHTTPClient(client *http.Client) *CircuitsProvidersListParams {
	var ()
	return &CircuitsProvidersListParams{
		HTTPClient: client,
	}
}

/*CircuitsProvidersListParams contains all the parameters to send to the API endpoint
for the circuits providers list operation typically these are written to a http.Request
*/
type CircuitsProvidersListParams struct {

	/*Account*/
	Account *string
	/*Asn*/
	Asn *float64
	/*IDIn
	  Multiple values may be separated by commas.

	*/
	IDIn *string
	/*Limit
	  Number of results to return per page.

	*/
	Limit *int64
	/*Name*/
	Name *string
	/*Offset
	  The initial index from which to return the results.

	*/
	Offset *int64
	/*Q*/
	Q *string
	/*Site*/
	Site *string
	/*SiteID*/
	SiteID *int64
	/*Slug*/
	Slug *string
	/*Tag*/
	Tag *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the circuits providers list params
func (o *CircuitsProvidersListParams) WithTimeout(timeout time.Duration) *CircuitsProvidersListParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the circuits providers list params
func (o *CircuitsProvidersListParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the circuits providers list params
func (o *CircuitsProvidersListParams) WithContext(ctx context.Context) *CircuitsProvidersListParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the circuits providers list params
func (o *CircuitsProvidersListParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the circuits providers list params
func (o *CircuitsProvidersListParams) WithHTTPClient(client *http.Client) *CircuitsProvidersListParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the circuits providers list params
func (o *CircuitsProvidersListParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAccount adds the account to the circuits providers list params
func (o *CircuitsProvidersListParams) WithAccount(account *string) *CircuitsProvidersListParams {
	o.SetAccount(account)
	return o
}

// SetAccount adds the account to the circuits providers list params
func (o *CircuitsProvidersListParams) SetAccount(account *string) {
	o.Account = account
}

// WithAsn adds the asn to the circuits providers list params
func (o *CircuitsProvidersListParams) WithAsn(asn *float64) *CircuitsProvidersListParams {
	o.SetAsn(asn)
	return o
}

// SetAsn adds the asn to the circuits providers list params
func (o *CircuitsProvidersListParams) SetAsn(asn *float64) {
	o.Asn = asn
}

// WithIDIn adds the iDIn to the circuits providers list params
func (o *CircuitsProvidersListParams) WithIDIn(iDIn *string) *CircuitsProvidersListParams {
	o.SetIDIn(iDIn)
	return o
}

// SetIDIn adds the idIn to the circuits providers list params
func (o *CircuitsProvidersListParams) SetIDIn(iDIn *string) {
	o.IDIn = iDIn
}

// WithLimit adds the limit to the circuits providers list params
func (o *CircuitsProvidersListParams) WithLimit(limit *int64) *CircuitsProvidersListParams {
	o.SetLimit(limit)
	return o
}

// SetLimit adds the limit to the circuits providers list params
func (o *CircuitsProvidersListParams) SetLimit(limit *int64) {
	o.Limit = limit
}

// WithName adds the name to the circuits providers list params
func (o *CircuitsProvidersListParams) WithName(name *string) *CircuitsProvidersListParams {
	o.SetName(name)
	return o
}

// SetName adds the name to the circuits providers list params
func (o *CircuitsProvidersListParams) SetName(name *string) {
	o.Name = name
}

// WithOffset adds the offset to the circuits providers list params
func (o *CircuitsProvidersListParams) WithOffset(offset *int64) *CircuitsProvidersListParams {
	o.SetOffset(offset)
	return o
}

// SetOffset adds the offset to the circuits providers list params
func (o *CircuitsProvidersListParams) SetOffset(offset *int64) {
	o.Offset = offset
}

// WithQ adds the q to the circuits providers list params
func (o *CircuitsProvidersListParams) WithQ(q *string) *CircuitsProvidersListParams {
	o.SetQ(q)
	return o
}

// SetQ adds the q to the circuits providers list params
func (o *CircuitsProvidersListParams) SetQ(q *string) {
	o.Q = q
}

// WithSite adds the site to the circuits providers list params
func (o *CircuitsProvidersListParams) WithSite(site *string) *CircuitsProvidersListParams {
	o.SetSite(site)
	return o
}

// SetSite adds the site to the circuits providers list params
func (o *CircuitsProvidersListParams) SetSite(site *string) {
	o.Site = site
}

// WithSiteID adds the siteID to the circuits providers list params
func (o *CircuitsProvidersListParams) WithSiteID(siteID *int64) *CircuitsProvidersListParams {
	o.SetSiteID(siteID)
	return o
}

// SetSiteID adds the siteId to the circuits providers list params
func (o *CircuitsProvidersListParams) SetSiteID(siteID *int64) {
	o.SiteID = siteID
}

// WithSlug adds the slug to the circuits providers list params
func (o *CircuitsProvidersListParams) WithSlug(slug *string) *CircuitsProvidersListParams {
	o.SetSlug(slug)
	return o
}

// SetSlug adds the slug to the circuits providers list params
func (o *CircuitsProvidersListParams) SetSlug(slug *string) {
	o.Slug = slug
}

// WithTag adds the tag to the circuits providers list params
func (o *CircuitsProvidersListParams) WithTag(tag *string) *CircuitsProvidersListParams {
	o.SetTag(tag)
	return o
}

// SetTag adds the tag to the circuits providers list params
func (o *CircuitsProvidersListParams) SetTag(tag *string) {
	o.Tag = tag
}

// WriteToRequest writes these params to a swagger request
func (o *CircuitsProvidersListParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Account != nil {

		// query param account
		var qrAccount string
		if o.Account != nil {
			qrAccount = *o.Account
		}
		qAccount := qrAccount
		if qAccount != "" {
			if err := r.SetQueryParam("account", qAccount); err != nil {
				return err
			}
		}

	}

	if o.Asn != nil {

		// query param asn
		var qrAsn float64
		if o.Asn != nil {
			qrAsn = *o.Asn
		}
		qAsn := swag.FormatFloat64(qrAsn)
		if qAsn != "" {
			if err := r.SetQueryParam("asn", qAsn); err != nil {
				return err
			}
		}

	}

	if o.IDIn != nil {

		// query param id__in
		var qrIDIn string
		if o.IDIn != nil {
			qrIDIn = *o.IDIn
		}
		qIDIn := qrIDIn
		if qIDIn != "" {
			if err := r.SetQueryParam("id__in", qIDIn); err != nil {
				return err
			}
		}

	}

	if o.Limit != nil {

		// query param limit
		var qrLimit int64
		if o.Limit != nil {
			qrLimit = *o.Limit
		}
		qLimit := swag.FormatInt64(qrLimit)
		if qLimit != "" {
			if err := r.SetQueryParam("limit", qLimit); err != nil {
				return err
			}
		}

	}

	if o.Name != nil {

		// query param name
		var qrName string
		if o.Name != nil {
			qrName = *o.Name
		}
		qName := qrName
		if qName != "" {
			if err := r.SetQueryParam("name", qName); err != nil {
				return err
			}
		}

	}

	if o.Offset != nil {

		// query param offset
		var qrOffset int64
		if o.Offset != nil {
			qrOffset = *o.Offset
		}
		qOffset := swag.FormatInt64(qrOffset)
		if qOffset != "" {
			if err := r.SetQueryParam("offset", qOffset); err != nil {
				return err
			}
		}

	}

	if o.Q != nil {

		// query param q
		var qrQ string
		if o.Q != nil {
			qrQ = *o.Q
		}
		qQ := qrQ
		if qQ != "" {
			if err := r.SetQueryParam("q", qQ); err != nil {
				return err
			}
		}

	}

	if o.Site != nil {

		// query param site
		var qrSite string
		if o.Site != nil {
			qrSite = *o.Site
		}
		qSite := qrSite
		if qSite != "" {
			if err := r.SetQueryParam("site", qSite); err != nil {
				return err
			}
		}

	}

	if o.SiteID != nil {

		// query param site_id
		var qrSiteID int64
		if o.SiteID != nil {
			qrSiteID = *o.SiteID
		}
		qSiteID := swag.FormatInt64(qrSiteID)
		if qSiteID != "" {
			if err := r.SetQueryParam("site_id", qSiteID); err != nil {
				return err
			}
		}

	}

	if o.Slug != nil {

		// query param slug
		var qrSlug string
		if o.Slug != nil {
			qrSlug = *o.Slug
		}
		qSlug := qrSlug
		if qSlug != "" {
			if err := r.SetQueryParam("slug", qSlug); err != nil {
				return err
			}
		}

	}

	if o.Tag != nil {

		// query param tag
		var qrTag string
		if o.Tag != nil {
			qrTag = *o.Tag
		}
		qTag := qrTag
		if qTag != "" {
			if err := r.SetQueryParam("tag", qTag); err != nil {
				return err
			}
		}

	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
