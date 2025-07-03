package main

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// EWS SOAP request structures
type SOAPEnvelope struct {
	XMLName  xml.Name    `xml:"soap:Envelope"`
	Soap     string      `xml:"xmlns:soap,attr"`
	Types    string      `xml:"xmlns:t,attr"`
	Messages string      `xml:"xmlns:m,attr"`
	Header   *SOAPHeader `xml:"soap:Header,omitempty"`
	Body     SOAPBody    `xml:"soap:Body"`
}

type SOAPHeader struct {
	RequestServerVersion *RequestServerVersion `xml:"t:RequestServerVersion,omitempty"`
}

type RequestServerVersion struct {
	Version string `xml:"Version,attr"`
}

type SOAPBody struct {
	FindItem *FindItem `xml:"m:FindItem,omitempty"`
	GetItem  *GetItem  `xml:"m:GetItem,omitempty"`
}

type FindItem struct {
	Traversal       string           `xml:"Traversal,attr"`
	ItemShape       *ItemShape       `xml:"m:ItemShape"`
	CalendarView    *CalendarView    `xml:"m:CalendarView"`
	ParentFolderIds *ParentFolderIds `xml:"m:ParentFolderIds"`
}

type ItemShape struct {
	BaseShape string `xml:"t:BaseShape"`
}

type CalendarView struct {
	MaxEntriesReturned string `xml:"MaxEntriesReturned,attr"`
	StartDate          string `xml:"StartDate,attr"`
	EndDate            string `xml:"EndDate,attr"`
}

type ParentFolderIds struct {
	DistinguishedFolderId *DistinguishedFolderId `xml:"t:DistinguishedFolderId"`
}

type DistinguishedFolderId struct {
	Id string `xml:"Id,attr"`
}

type GetItem struct {
	ItemShape *ItemShape `xml:"m:ItemShape"`
	ItemIds   *ItemIds   `xml:"m:ItemIds"`
}

type ItemIds struct {
	ItemId []ItemId `xml:"t:ItemId"`
}

type ItemId struct {
	Id        string `xml:"Id,attr"`
	ChangeKey string `xml:"ChangeKey,attr"`
}

// Response structures
type SOAPResponse struct {
	XMLName xml.Name         `xml:"Envelope"`
	Body    SOAPResponseBody `xml:"Body"`
}

type SOAPResponseBody struct {
	FindItemResponse *FindItemResponse `xml:"FindItemResponse"`
	GetItemResponse  *GetItemResponse  `xml:"GetItemResponse"`
	Fault            *SOAPFault        `xml:"Fault"`
}

type SOAPFault struct {
	Code   string `xml:"faultcode"`
	String string `xml:"faultstring"`
}

type FindItemResponse struct {
	ResponseMessages *ResponseMessages `xml:"ResponseMessages"`
}

type ResponseMessages struct {
	FindItemResponseMessage *FindItemResponseMessage `xml:"FindItemResponseMessage"`
}

type FindItemResponseMessage struct {
	ResponseClass string      `xml:"ResponseClass,attr"`
	ResponseCode  string      `xml:"ResponseCode"`
	RootFolder    *RootFolder `xml:"RootFolder"`
}

type RootFolder struct {
	TotalItemsInView string `xml:"TotalItemsInView,attr"`
	Items            *Items `xml:"Items"`
}

type Items struct {
	CalendarItem []CalendarItem `xml:"CalendarItem"`
}

type CalendarItem struct {
	ItemId               ItemId     `xml:"ItemId"`
	Subject              string     `xml:"Subject"`
	Start                string     `xml:"Start"`
	End                  string     `xml:"End"`
	Location             string     `xml:"Location"`
	Organizer            *Organizer `xml:"Organizer"`
	LegacyFreeBusyStatus string     `xml:"LegacyFreeBusyStatus"`
	IsAllDayEvent        string     `xml:"IsAllDayEvent"`
	IsMeeting            string     `xml:"IsMeeting"`
}

type Organizer struct {
	Mailbox *Mailbox `xml:"Mailbox"`
}

type Mailbox struct {
	Name         string `xml:"Name"`
	EmailAddress string `xml:"EmailAddress"`
}

type GetItemResponse struct {
	ResponseMessages *GetItemResponseMessages `xml:"ResponseMessages"`
}

type GetItemResponseMessages struct {
	GetItemResponseMessage *GetItemResponseMessage `xml:"GetItemResponseMessage"`
}

type GetItemResponseMessage struct {
	ResponseClass string         `xml:"ResponseClass,attr"`
	ResponseCode  string         `xml:"ResponseCode"`
	Items         *ResponseItems `xml:"Items"`
}

type ResponseItems struct {
	CalendarItem []CalendarItem `xml:"CalendarItem"`
}

// ExchangeClient handles communication with Exchange Web Services
type ExchangeClient struct {
	serverURL   string
	credentials *ExchangeCredentials
	httpClient  *http.Client
}

// NewExchangeClient creates a new Exchange client
func NewExchangeClient(serverURL string, credentials *ExchangeCredentials) *ExchangeClient {
	// Create HTTP client with basic auth and TLS config
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // For self-signed certificates
	}

	return &ExchangeClient{
		serverURL:   serverURL,
		credentials: credentials,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

// GetCalendarEvents retrieves calendar events for the user
func (p *Plugin) getCalendarEvents(credentials *ExchangeCredentials) ([]CalendarEvent, error) {
	config := p.getConfiguration()
	client := NewExchangeClient(config.ExchangeServerURL, credentials)

	now := time.Now()
	start := now.Add(-24 * time.Hour)  // Last 24 hours
	end := now.Add(7 * 24 * time.Hour) // Next 7 days

	return client.GetCalendarEventsInRange(start, end)
}

// GetCalendarEventsInRange retrieves calendar events within a specific time range
func (p *Plugin) getCalendarEventsInRange(credentials *ExchangeCredentials, start, end time.Time) ([]CalendarEvent, error) {
	config := p.getConfiguration()
	client := NewExchangeClient(config.ExchangeServerURL, credentials)

	return client.GetCalendarEventsInRange(start, end)
}

// GetNewMeetingInvitations retrieves new meeting invitations
func (p *Plugin) getNewMeetingInvitations(credentials *ExchangeCredentials) ([]CalendarEvent, error) {
	// TODO: Implement invitation tracking using Exchange Web Services
	// For now, return empty slice
	_ = credentials // credentials will be used when implementing EWS invitation tracking
	return []CalendarEvent{}, nil
}

// GetCalendarEventsInRange gets events in a date range
func (c *ExchangeClient) GetCalendarEventsInRange(start, end time.Time) ([]CalendarEvent, error) {
	// Construct SOAP request
	envelope := &SOAPEnvelope{
		Soap:     "http://schemas.xmlsoap.org/soap/envelope/",
		Types:    "http://schemas.microsoft.com/exchange/services/2006/types",
		Messages: "http://schemas.microsoft.com/exchange/services/2006/messages",
		Header: &SOAPHeader{
			RequestServerVersion: &RequestServerVersion{
				Version: "Exchange2013",
			},
		},
		Body: SOAPBody{
			FindItem: &FindItem{
				Traversal: "Shallow",
				ItemShape: &ItemShape{
					BaseShape: "Default",
				},
				CalendarView: &CalendarView{
					MaxEntriesReturned: "1000",
					StartDate:          start.UTC().Format("2006-01-02T15:04:05Z"),
					EndDate:            end.UTC().Format("2006-01-02T15:04:05Z"),
				},
				ParentFolderIds: &ParentFolderIds{
					DistinguishedFolderId: &DistinguishedFolderId{
						Id: "calendar",
					},
				},
			},
		},
	}

	// Marshal to XML
	xmlData, err := xml.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SOAP request: %w", err)
	}

	// Add XML declaration
	soapRequest := `<?xml version="1.0" encoding="utf-8"?>` + string(xmlData)

	// Create HTTP request
	ewsURL := c.serverURL + "/EWS/Exchange.asmx"
	req, err := http.NewRequest("POST", ewsURL, strings.NewReader(soapRequest))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", `"http://schemas.microsoft.com/exchange/services/2006/messages/FindItem"`)

	// Set basic auth
	if c.credentials.Domain != "" {
		req.SetBasicAuth(c.credentials.Domain+"\\"+c.credentials.Username, c.credentials.Password)
	} else {
		req.SetBasicAuth(c.credentials.Username, c.credentials.Password)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	// Parse SOAP response
	var soapResp SOAPResponse
	if err := xml.Unmarshal(body, &soapResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SOAP response: %w", err)
	}

	// Check for SOAP fault
	if soapResp.Body.Fault != nil {
		return nil, fmt.Errorf("SOAP fault: %s - %s", soapResp.Body.Fault.Code, soapResp.Body.Fault.String)
	}

	// Extract calendar events
	var events []CalendarEvent
	if soapResp.Body.FindItemResponse != nil &&
		soapResp.Body.FindItemResponse.ResponseMessages != nil &&
		soapResp.Body.FindItemResponse.ResponseMessages.FindItemResponseMessage != nil {

		responseMessage := soapResp.Body.FindItemResponse.ResponseMessages.FindItemResponseMessage

		if responseMessage.ResponseClass != "Success" {
			return nil, fmt.Errorf("EWS error: %s", responseMessage.ResponseCode)
		}

		if responseMessage.RootFolder != nil && responseMessage.RootFolder.Items != nil {
			for _, item := range responseMessage.RootFolder.Items.CalendarItem {
				event, err := c.convertToCalendarEvent(item)
				if err != nil {
					continue // Skip problematic events
				}
				events = append(events, event)
			}
		}
	}

	return events, nil
}

// convertToCalendarEvent converts EWS CalendarItem to our CalendarEvent structure
func (c *ExchangeClient) convertToCalendarEvent(item CalendarItem) (CalendarEvent, error) {
	startTime, err := time.Parse("2006-01-02T15:04:05Z", item.Start)
	if err != nil {
		return CalendarEvent{}, fmt.Errorf("failed to parse start time: %w", err)
	}

	endTime, err := time.Parse("2006-01-02T15:04:05Z", item.End)
	if err != nil {
		return CalendarEvent{}, fmt.Errorf("failed to parse end time: %w", err)
	}

	var organizer string
	if item.Organizer != nil && item.Organizer.Mailbox != nil {
		organizer = item.Organizer.Mailbox.Name
		if organizer == "" {
			organizer = item.Organizer.Mailbox.EmailAddress
		}
	}

	// Map EWS status to our status
	var status string
	switch item.LegacyFreeBusyStatus {
	case "Free":
		status = "Free"
	case "Busy":
		status = "Busy"
	case "Tentative":
		status = "Tentative"
	case "OOF":
		status = "OutOfOffice"
	default:
		status = "Busy" // Default to busy
	}

	return CalendarEvent{
		ID:        item.ItemId.Id,
		Subject:   item.Subject,
		Start:     startTime,
		End:       endTime,
		Location:  item.Location,
		Organizer: organizer,
		IsAllDay:  item.IsAllDayEvent == "true",
		IsMeeting: item.IsMeeting == "true",
		Status:    status,
	}, nil
}
