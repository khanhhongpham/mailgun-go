// TODO(sfalvo):
// Document how to run acceptance tests.

// Package mailgun provides methods for interacting with the Mailgun API.
// It automates the HTTP request/response cycle, encodings, and other details needed by the API.
// This SDK lets you do everything the API lets you, in a more Go-friendly way.
//
// For further information please see the Mailgun documentation at
// http://documentation.mailgun.com/
//
//  Original Author: Michael Banzon
//  Contributions:   Samuel A. Falvo II <sam.falvo %at% rackspace.com>
//  Version:         0.99.0
//
// Examples
//
// This document includes a number of examples which illustrates some aspects of the GUI which might be misleading or confusing.
// All examples included are derived from an acceptance test.
// Note that every SDK function has a corresponding acceptance test, so
// if you don't find an example for a function you'd like to know more about,
// please check the acceptance sub-package for a corresponding test.
// Of course, contributions to the documentation are always welcome as well.
// Feel free to submit a pull request or open a Github issue if you cannot find an example to suit your needs.
//
// Limit and Skip Settings
//
// Many SDK functions consume a pair of parameters called limit and skip.
// These help control how much data Mailgun sends over the wire.
// Limit, as you'd expect, gives a count of the number of records you want to receive.
// Note that, at present, Mailgun imposes its own cap of 100, for all API endpoints.
// Skip indicates where in the data set you want to start receiving from.
// Mailgun defaults to the very beginning of the dataset if not specified explicitly.
//
// If you don't particularly care how much data you receive, you may specify DefaultLimit.
// If you similarly don't care about where the data starts, you may specify DefaultSkip.
//
// Functions that Return Totals
//
// Functions which accept a limit and skip setting, in general,
// will also return a total count of the items returned.
// Note that this total count is not the total in the bundle returned by the call.
// You can determine that easily enough with Go's len() function.
// The total that you receive actually refers to the complete set of data on the server.
// This total may well exceed the size returned from the API.
//
// If this happens, you may find yourself needing to iterate over the dataset of interest.
// For example:
//
//		// Get total amount of stuff we have to work with.
// 		mg := NewMailgun("example.com", "my_api_key", "")
// 		n, _, err := mg.GetStats(1, 0, nil, "sent", "opened")
// 		if err != nil {
// 			t.Fatal(err)
// 		}
//		// Loop over it all.
//		for sk := 0; sk < n; sk += limit {
//			_, stats, err := mg.GetStats(limit, sk, nil, "sent", "opened")
//		 	if err != nil {
//		 		t.Fatal(err)
//		 	}
//			doSomethingWith(stats)
//		}
//
// License
//
// Copyright (c) 2013-2014, Michael Banzon.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// * Redistributions of source code must retain the above copyright notice, this
// list of conditions and the following disclaimer.
//
// * Redistributions in binary form must reproduce the above copyright notice, this
// list of conditions and the following disclaimer in the documentation and/or
// other materials provided with the distribution.
//
// * Neither the names of Mailgun, Michael Banzon, nor the names of their
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
// ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package mailgun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gobuffalo/envy"
)

var Debug = false

// ENV is used to help switch settings based on where the
// application is being run. Default is "development".
var ENV = envy.Get("GO_ENV", "development")

const (
	ApiBase              = "https://api.mailgun.net/v3"
	messagesEndpoint     = "messages"
	mimeMessagesEndpoint = "messages.mime"
	bouncesEndpoint      = "bounces"
	statsTotalEndpoint   = "stats/total"
	domainsEndpoint      = "domains"
	tagsEndpoint         = "tags"
	eventsEndpoint       = "events"
	unsubscribesEndpoint = "unsubscribes"
	routesEndpoint       = "routes"
	ipsEndpoint          = "ips"
	exportsEndpoint      = "exports"
	webhooksEndpoint     = "webhooks"
	listsEndpoint        = "lists"
	basicAuthUser        = "api"
)

// Mailgun defines the supported subset of the Mailgun API.
// The Mailgun API may contain additional features which have been deprecated since writing this SDK.
// This SDK only covers currently supported interface endpoints.
//
// Note that Mailgun reserves the right to deprecate endpoints.
// Some endpoints listed in this interface may, at any time, become obsolete.
// Always double-check with the Mailgun API Documentation to
// determine the currently supported feature set.
type Mailgun interface {
	APIBase() string
	Domain() string
	APIKey() string
	Client() *http.Client
	SetClient(client *http.Client)
	SetAPIBase(url string)

	Send(ctx context.Context, m *Message) (string, string, error)
	NewMessage(from, subject, text string, to ...string) *Message
	NewMIMEMessage(body io.ReadCloser, to ...string) *Message

	ListBounces(ctx context.Context, opts *ListOptions) ([]Bounce, error)
	GetBounce(ctx context.Context, address string) (Bounce, error)
	AddBounce(ctx context.Context, address, code, error string) error
	DeleteBounce(ctx context.Context, address string) error

	ListStats(ctx context.Context, events []string, opts *ListStatOptions) ([]Stats, error)
	GetTag(ctx context.Context, tag string) (Tag, error)
	DeleteTag(ctx context.Context, tag string) error
	ListTags(*ListTagOptions) *TagIterator

	ListDomains(ctx context.Context, opts *ListOptions) (int, []Domain, error)
	GetDomain(ctx context.Context, domain string) (Domain, []DNSRecord, []DNSRecord, error)
	CreateDomain(ctx context.Context, name string, pass string, opts *CreateDomainOptions) error
	DeleteDomain(ctx context.Context, name string) error
	UpdateDomainConnection(ctx context.Context, domain string, dc DomainConnection) error
	GetDomainConnection(ctx context.Context, domain string) (DomainConnection, error)
	GetDomainTracking(ctx context.Context, domain string) (DomainTracking, error)

	GetStoredMessage(ctx context.Context, id string) (StoredMessage, error)
	GetStoredMessageRaw(ctx context.Context, id string) (StoredMessageRaw, error)
	GetStoredMessageForURL(ctx context.Context, url string) (StoredMessage, error)
	GetStoredMessageRawForURL(ctx context.Context, url string) (StoredMessageRaw, error)
	DeleteStoredMessage(ctx context.Context, id string) error

	ListCredentials(opts *ListOptions) *CredentialsIterator
	CreateCredential(ctx context.Context, login, password string) error
	ChangeCredentialPassword(ctx context.Context, id, password string) error
	DeleteCredential(ctx context.Context, id string) error

	ListUnsubscribes(opts *ListOptions) *UnsubscribesIterator
	GetUnsubscribe(ctx context.Context, address string) (Unsubscribe, error)
	CreateUnsubscribe(ctx context.Context, address, tag string) error
	DeleteUnsubscribe(ctx context.Context, address string) error
	DeleteUnsubscribeWithTag(ctx context.Context, a, t string) error

	ListComplaints(opts *ListOptions) *ComplaintsIterator
	GetComplaint(ctx context.Context, address string) (Complaint, error)
	CreateComplaint(ctx context.Context, address string) error
	DeleteComplaint(ctx context.Context, address string) error

	ListRoutes(opts *ListOptions) *RoutesIterator
	GetRoute(ctx context.Context, address string) (Route, error)
	CreateRoute(ctx context.Context, address Route) (Route, error)
	DeleteRoute(ctx context.Context, address string) error
	UpdateRoute(ctx context.Context, address string, r Route) (Route, error)

	ListWebhooks(ctx context.Context) (map[string]string, error)
	CreateWebhook(ctx context.Context, kind string, url []string) error
	DeleteWebhook(ctx context.Context, kind string) error
	GetWebhook(ctx context.Context, kind string) (string, error)
	UpdateWebhook(ctx context.Context, kind string, url []string) error
	VerifyWebhookRequest(req *http.Request) (verified bool, err error)

	ListMailingLists(opts *ListOptions) *ListsIterator
	CreateMailingList(ctx context.Context, address MailingList) (MailingList, error)
	DeleteMailingList(ctx context.Context, address string) error
	GetMailingList(ctx context.Context, address string) (MailingList, error)
	UpdateMailingList(ctx context.Context, address string, ml MailingList) (MailingList, error)

	ListMembers(address string, opts *ListOptions) *MemberListIterator
	GetMember(ctx context.Context, MemberAddr, listAddr string) (Member, error)
	CreateMember(ctx context.Context, merge bool, addr string, prototype Member) error
	CreateMemberList(ctx context.Context, subscribed *bool, addr string, newMembers []interface{}) error
	UpdateMember(ctx context.Context, Member, list string, prototype Member) (Member, error)
	DeleteMember(ctx context.Context, Member, list string) error

	ListEvents(*ListEventOptions) *EventIterator
	PollEvents(*ListEventOptions) *EventPoller

	ListIPS(ctx context.Context, dedicated bool) ([]IPAddress, error)
	GetIP(ctx context.Context, ip string) (IPAddress, error)
	ListDomainIPS(ctx context.Context) ([]IPAddress, error)
	AddDomainIP(ctx context.Context, ip string) error
	DeleteDomainIP(ctx context.Context, ip string) error

	ListExports(ctx context.Context, url string) ([]Export, error)
	GetExport(ctx context.Context, id string) (Export, error)
	GetExportLink(ctx context.Context, id string) (string, error)
	CreateExport(ctx context.Context, url string) error

	GetTagLimits(ctx context.Context, domain string) (TagLimits, error)
}

// MailgunImpl bundles data needed by a large number of methods in order to interact with the Mailgun API.
// Colloquially, we refer to instances of this structure as "clients."
type MailgunImpl struct {
	apiBase string
	domain  string
	apiKey  string
	client  *http.Client
	baseURL string
}

// NewMailGun creates a new client instance.
func NewMailgun(domain, apiKey string) *MailgunImpl {
	return &MailgunImpl{
		apiBase: ApiBase,
		domain:  domain,
		apiKey:  apiKey,
		client:  http.DefaultClient,
	}
}

// Return a new Mailgun client using the environment variables
// MG_API_KEY, MG_DOMAIN, and MG_URL
func NewMailgunFromEnv() (*MailgunImpl, error) {
	apiKey := os.Getenv("MG_API_KEY")
	if apiKey == "" {
		return nil, errors.New("required environment variable MG_API_KEY not defined")
	}
	domain := os.Getenv("MG_DOMAIN")
	if domain == "" {
		return nil, errors.New("required environment variable MG_DOMAIN not defined")
	}

	mg := NewMailgun(domain, apiKey)

	url := os.Getenv("MG_URL")
	if url != "" {
		mg.SetAPIBase(url)
	}

	return mg, nil
}

// ApiBase returns the API Base URL configured for this client.
func (mg *MailgunImpl) APIBase() string {
	return mg.apiBase
}

// Domain returns the domain configured for this client.
func (mg *MailgunImpl) Domain() string {
	return mg.domain
}

// ApiKey returns the API key configured for this client.
func (mg *MailgunImpl) APIKey() string {
	return mg.apiKey
}

// Client returns the HTTP client configured for this client.
func (mg *MailgunImpl) Client() *http.Client {
	return mg.client
}

// SetClient updates the HTTP client for this client.
func (mg *MailgunImpl) SetClient(c *http.Client) {
	mg.client = c
}

// SetAPIBase updates the API Base URL for this client.
func (mg *MailgunImpl) SetAPIBase(address string) {
	mg.apiBase = address
}

// generateApiUrl renders a URL for an API endpoint using the domain and endpoint name.
func generateApiUrl(m Mailgun, endpoint string) string {
	return fmt.Sprintf("%s/%s/%s", m.APIBase(), m.Domain(), endpoint)
}

// generateApiUrlWithDomain renders a URL for an API endpoint using a separate domain and endpoint name.
func generateApiUrlWithDomain(m Mailgun, endpoint, domain string) string {
	return fmt.Sprintf("%s/%s/%s", m.APIBase(), domain, endpoint)
}

// generateMemberApiUrl renders a URL relevant for specifying mailing list members.
// The address parameter refers to the mailing list in question.
func generateMemberApiUrl(m Mailgun, endpoint, address string) string {
	return fmt.Sprintf("%s/%s/%s/members", m.APIBase(), endpoint, address)
}

// generateApiUrlWithTarget works as generateApiUrl,
// but consumes an additional resource parameter called 'target'.
func generateApiUrlWithTarget(m Mailgun, endpoint, target string) string {
	tail := ""
	if target != "" {
		tail = fmt.Sprintf("/%s", target)
	}
	return fmt.Sprintf("%s%s", generateApiUrl(m, endpoint), tail)
}

// generateDomainApiUrl renders a URL as generateApiUrl, but
// addresses a family of functions which have a non-standard URL structure.
// Most URLs consume a domain in the 2nd position, but some endpoints
// require the word "domains" to be there instead.
func generateDomainApiUrl(m Mailgun, endpoint string) string {
	return fmt.Sprintf("%s/domains/%s/%s", m.APIBase(), m.Domain(), endpoint)
}

// generateCredentialsUrl renders a URL as generateDomainApiUrl,
// but focuses on the SMTP credentials family of API functions.
func generateCredentialsUrl(m Mailgun, id string) string {
	tail := ""
	if id != "" {
		tail = fmt.Sprintf("/%s", id)
	}
	return generateDomainApiUrl(m, fmt.Sprintf("credentials%s", tail))
	// return fmt.Sprintf("%s/domains/%s/credentials%s", apiBase, m.Domain(), tail)
}

// generateStoredMessageUrl generates the URL needed to acquire a copy of a stored message.
func generateStoredMessageUrl(m Mailgun, endpoint, id string) string {
	return generateDomainApiUrl(m, fmt.Sprintf("%s/%s", endpoint, id))
	// return fmt.Sprintf("%s/domains/%s/%s/%s", apiBase, m.Domain(), endpoint, id)
}

// generatePublicApiUrl works as generateApiUrl, except that generatePublicApiUrl has no need for the domain.
func generatePublicApiUrl(m Mailgun, endpoint string) string {
	return fmt.Sprintf("%s/%s", m.APIBase(), endpoint)
}

// generateParameterizedUrl works as generateApiUrl, but supports query parameters.
func generateParameterizedUrl(m Mailgun, endpoint string, payload payload) (string, error) {
	paramBuffer, err := payload.getPayloadBuffer()
	if err != nil {
		return "", err
	}
	params := string(paramBuffer.Bytes())
	return fmt.Sprintf("%s?%s", generateApiUrl(m, eventsEndpoint), params), nil
}

// parseMailgunTime translates a timestamp as returned by Mailgun into a Go standard timestamp.
func parseMailgunTime(ts string) (t time.Time, err error) {
	t, err = time.Parse("Mon, 2 Jan 2006 15:04:05 MST", ts)
	return
}

// formatMailgunTime translates a timestamp into a human-readable form.
func formatMailgunTime(t *time.Time) string {
	return t.Format("Mon, 2 Jan 2006 15:04:05 -0700")
}
