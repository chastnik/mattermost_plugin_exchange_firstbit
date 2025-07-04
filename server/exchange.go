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
	// Create HTTP client with TLS config for Russian servers like 1cbit.ru
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,             // For self-signed certificates
			MinVersion:         tls.VersionTLS10, // Support older TLS versions
			MaxVersion:         tls.VersionTLS13,
		},
		DisableKeepAlives:     true,             // Disable keep-alives to avoid 440 timeouts
		IdleConnTimeout:       30 * time.Second, // Shorter idle timeout
		MaxIdleConns:          1,                // Minimal connection pooling
		MaxIdleConnsPerHost:   1,                // One connection per host
		ResponseHeaderTimeout: 30 * time.Second, // Response header timeout
	}

	return &ExchangeClient{
		serverURL:   serverURL,
		credentials: credentials,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second, // Shorter timeout to avoid 440 errors
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

	// Create HTTP request - try to find working EWS endpoint
	ewsURL := c.findWorkingEWSEndpoint()
	if ewsURL == "" {
		ewsURL = c.serverURL + "/EWS/Exchange.asmx" // fallback
	}
	req, err := http.NewRequest("POST", ewsURL, strings.NewReader(soapRequest))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", `"http://schemas.microsoft.com/exchange/services/2006/messages/FindItem"`)

	// Set NTLM auth
	var username string
	if c.credentials.Domain != "" {
		username = c.credentials.Domain + "\\" + c.credentials.Username
	} else {
		username = c.credentials.Username
	}
	req.SetBasicAuth(username, c.credentials.Password)

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

// TestConnection tests the connection to Exchange without fetching events
func (c *ExchangeClient) TestConnection() error {
	// Try different username formats for authentication
	userFormats := []string{}

	if c.credentials.Domain != "" {
		// Try various domain formats
		userFormats = append(userFormats, c.credentials.Domain+"\\"+c.credentials.Username)
		userFormats = append(userFormats, c.credentials.Username+"@"+c.credentials.Domain)
		userFormats = append(userFormats, c.credentials.Username+"@"+c.credentials.Domain+".local")
		userFormats = append(userFormats, c.credentials.Username+"@"+c.credentials.Domain+".ru")

		// Try without domain prefix for 1cbit.ru server
		if c.credentials.Domain == "pbr" {
			userFormats = append(userFormats, c.credentials.Username+"@1cbit.ru")
			userFormats = append(userFormats, c.credentials.Username+"@pbr.1cbit.ru")
			userFormats = append(userFormats, c.credentials.Username+"@mail.1cbit.ru")
		}
	} else {
		// Just username
		userFormats = append(userFormats, c.credentials.Username)
	}

	// Try different EWS endpoints - some servers use different paths
	ewsPaths := []string{
		"/owa/EWS/Exchange.asmx", // Приоритет для российских серверов типа 1cbit.ru
		"/EWS/Exchange.asmx",
		"/ews/exchange.asmx",
		"/Exchange/ews/Exchange.asmx",
		"/exchange/ews/exchange.asmx",
		"/EWS/Services.wsdl",           // Alternative EWS path
		"/microsoft-server-activesync", // Sometimes used for OWA
	}

	var attemptResults []string
	var lastError error
	attemptCount := 0

	// First, try to discover the correct EWS endpoint
	for _, ewsPath := range ewsPaths {
		ewsURL := c.serverURL + ewsPath

		// Test with the first username format only for endpoint discovery
		username := userFormats[0]

		req, err := http.NewRequest("GET", ewsURL, nil)
		if err != nil {
			continue
		}

		req.SetBasicAuth(username, c.credentials.Password)

		// Send request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		// If we get anything other than 404, this endpoint exists
		if resp.StatusCode != 404 {
			// Now try all username formats with this working endpoint
			for _, userFormat := range userFormats {
				attemptCount++
				req, err := http.NewRequest("GET", ewsURL, nil)
				if err != nil {
					attemptResults = append(attemptResults, fmt.Sprintf("Попытка %d (%s → %s): Ошибка создания запроса - %v", attemptCount, userFormat, ewsPath, err))
					lastError = err
					continue
				}

				req.SetBasicAuth(userFormat, c.credentials.Password)

				// Send request
				resp, err := c.httpClient.Do(req)
				if err != nil {
					attemptResults = append(attemptResults, fmt.Sprintf("Попытка %d (%s → %s): Ошибка отправки запроса - %v", attemptCount, userFormat, ewsPath, err))
					lastError = err
					continue
				}
				defer resp.Body.Close()

				// Check status code
				if resp.StatusCode == 200 || resp.StatusCode == 405 { // 405 Method Not Allowed is OK for EWS
					return nil // Success!
				}

				if resp.StatusCode == 401 {
					attemptResults = append(attemptResults, fmt.Sprintf("Попытка %d (%s → %s): HTTP 401 - Неверные учетные данные", attemptCount, userFormat, ewsPath))
					lastError = fmt.Errorf("HTTP 401: Неверные учетные данные")
					continue // Try next format
				}

				if resp.StatusCode == 440 {
					// HTTP 440 Login Timeout - retry with fresh connection
					attemptResults = append(attemptResults, fmt.Sprintf("Попытка %d (%s → %s): HTTP 440 - Login Timeout, повтор...", attemptCount, userFormat, ewsPath))

					// Wait a moment and retry with fresh connection
					time.Sleep(2 * time.Second)

					retryReq, retryErr := http.NewRequest("GET", ewsURL, nil)
					if retryErr == nil {
						retryReq.SetBasicAuth(userFormat, c.credentials.Password)
						retryResp, retryErr := c.httpClient.Do(retryReq)
						if retryErr == nil {
							retryResp.Body.Close()
							if retryResp.StatusCode == 200 || retryResp.StatusCode == 405 {
								return nil // Success on retry!
							}
						}
					}

					lastError = fmt.Errorf("HTTP 440: Login Timeout (повтор не помог)")
					continue
				}

				if resp.StatusCode >= 400 {
					body, _ := io.ReadAll(resp.Body)
					bodyStr := strings.TrimSpace(string(body))
					if len(bodyStr) > 100 {
						bodyStr = bodyStr[:100] + "..."
					}
					attemptResults = append(attemptResults, fmt.Sprintf("Попытка %d (%s → %s): HTTP %d - %s", attemptCount, userFormat, ewsPath, resp.StatusCode, bodyStr))
					lastError = fmt.Errorf("HTTP %d", resp.StatusCode)
					continue
				}

				attemptResults = append(attemptResults, fmt.Sprintf("Попытка %d (%s → %s): HTTP %d - Неожиданный ответ", attemptCount, userFormat, ewsPath, resp.StatusCode))
			}
			break // We found a working endpoint, no need to try others
		}
	}

	// If no working endpoint found
	if attemptCount == 0 {
		attemptResults = append(attemptResults, "Не найден ни один рабочий EWS endpoint")
		for _, path := range ewsPaths {
			attemptResults = append(attemptResults, fmt.Sprintf("  • Проверен путь: %s%s", c.serverURL, path))
		}
	}

	// Build detailed error message
	errorMsg := fmt.Sprintf("❌ Не удалось подключиться к Exchange сервер %s ни с одним форматом имени пользователя.\n\nПодробности:\n", c.serverURL)
	for _, result := range attemptResults {
		errorMsg += "• " + result + "\n"
	}

	errorMsg += "\n🔍 Рекомендации для сервера mail.1cbit.ru:\n"
	errorMsg += "• Попробуйте разные форматы URL:\n"
	errorMsg += "  - https://mail.1cbit.ru\n"
	errorMsg += "  - https://mail.1cbit.ru/owa\n"
	errorMsg += "  - https://owa.1cbit.ru\n"
	errorMsg += "• Убедитесь, что EWS включен на сервере\n"
	errorMsg += "• Проверьте, что пользователь имеет права на EWS\n"
	errorMsg += "• Обратитесь к системному администратору за точным URL\n"

	if lastError != nil {
		errorMsg += fmt.Sprintf("\nПоследняя ошибка: %v", lastError)
	}

	return fmt.Errorf(errorMsg)
}

// findWorkingEWSEndpoint discovers the correct EWS endpoint for the server
func (c *ExchangeClient) findWorkingEWSEndpoint() string {
	ewsPaths := []string{
		"/owa/EWS/Exchange.asmx", // Приоритет для российских серверов типа 1cbit.ru
		"/EWS/Exchange.asmx",
		"/ews/exchange.asmx",
		"/Exchange/ews/Exchange.asmx",
		"/exchange/ews/exchange.asmx",
	}

	username := c.credentials.Username
	if c.credentials.Domain != "" {
		username = c.credentials.Domain + "\\" + c.credentials.Username
	}

	for _, path := range ewsPaths {
		ewsURL := c.serverURL + path
		req, err := http.NewRequest("GET", ewsURL, nil)
		if err != nil {
			continue
		}

		req.SetBasicAuth(username, c.credentials.Password)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		// If we don't get 404, this endpoint exists
		if resp.StatusCode != 404 {
			return ewsURL
		}
	}

	return "" // No working endpoint found
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
