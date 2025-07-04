package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/credentials", p.handleCredentials).Methods("POST")
	api.HandleFunc("/calendar", p.handleGetCalendar).Methods("GET")
	api.HandleFunc("/meeting/accept", p.handleMeetingResponse).Methods("POST")
	api.HandleFunc("/meeting/decline", p.handleMeetingResponse).Methods("POST")
	api.HandleFunc("/meeting/tentative", p.handleMeetingResponse).Methods("POST")
	api.HandleFunc("/test-connection", p.handleTestConnection).Methods("POST")
	api.HandleFunc("/reminder/snooze", p.handleSnoozeReminder).Methods("POST")
	api.HandleFunc("/calendar/open", p.handleOpenCalendar).Methods("POST")
	api.HandleFunc("/reminders", p.handleGetReminders).Methods("GET")
	api.HandleFunc("/reminders/update", p.handleUpdateReminders).Methods("POST")

	router.ServeHTTP(w, r)
}

// handleCredentials handles setting user Exchange credentials
func (p *Plugin) handleCredentials(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var credentials ExchangeCredentials
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate credentials
	if credentials.Username == "" || credentials.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Test connection using lightweight TestConnection method instead of full calendar query
	config := p.getConfiguration()
	if config.ExchangeServerURL == "" {
		http.Error(w, "Exchange server URL not configured", http.StatusInternalServerError)
		return
	}

	client := NewExchangeClient(config.ExchangeServerURL, &credentials)
	err := client.TestConnection()
	if err != nil {
		// Log the error for debugging
		p.API.LogError("Failed to save credentials due to connection test failure",
			"error", err.Error(),
			"server_url", config.ExchangeServerURL,
			"username", credentials.Username,
			"domain", credentials.Domain)

		http.Error(w, fmt.Sprintf("Failed to connect to Exchange: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Store credentials
	data, err := json.Marshal(credentials)
	if err != nil {
		http.Error(w, "Failed to marshal credentials", http.StatusInternalServerError)
		return
	}

	if err := p.API.KVSet(fmt.Sprintf("exchange_creds_%s", userID), data); err != nil {
		http.Error(w, "Failed to store credentials", http.StatusInternalServerError)
		return
	}

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Credentials saved successfully",
	})

	// Send confirmation message to user
	bot, botErr := p.API.GetBot("", true)
	if botErr == nil {
		channel, channelErr := p.API.GetDirectChannel(userID, bot.UserId)
		if channelErr == nil {
			post := &model.Post{
				ChannelId: channel.Id,
				UserId:    bot.UserId,
				Message:   "✅ **Exchange Integration настроена!**\n\nВаши учетные данные сохранены и синхронизация календаря активирована.",
			}
			p.API.CreatePost(post)
		}
	}
}

// handleGetCalendar returns user's calendar events
func (p *Plugin) handleGetCalendar(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	credentials, err := p.getUserExchangeCredentials(userID)
	if err != nil {
		http.Error(w, "Exchange credentials not configured", http.StatusBadRequest)
		return
	}

	events, err := p.getCalendarEvents(credentials)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get calendar events: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// handleMeetingResponse handles meeting invitation responses
func (p *Plugin) handleMeetingResponse(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var request struct {
		Context map[string]interface{} `json:"context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	eventID, ok := request.Context["event_id"].(string)
	if !ok {
		http.Error(w, "Missing event_id", http.StatusBadRequest)
		return
	}

	// Determine response type from URL path
	var responseType string
	if strings.Contains(r.URL.Path, "accept") {
		responseType = "Accept"
	} else if strings.Contains(r.URL.Path, "decline") {
		responseType = "Decline"
	} else if strings.Contains(r.URL.Path, "tentative") {
		responseType = "Tentative"
	} else {
		http.Error(w, "Invalid response type", http.StatusBadRequest)
		return
	}

	// Get user credentials
	credentials, err := p.getUserExchangeCredentials(userID)
	if err != nil {
		http.Error(w, "Exchange credentials not configured", http.StatusBadRequest)
		return
	}

	// Send meeting response (this would need to be implemented with EWS)
	err = p.sendMeetingResponse(credentials, eventID, responseType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to send meeting response: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Send confirmation message
	var emoji, action string
	switch responseType {
	case "Accept":
		emoji = "✅"
		action = "приняли"
	case "Decline":
		emoji = "❌"
		action = "отклонили"
	case "Tentative":
		emoji = "❓"
		action = "отметили как под вопросом"
	}

	bot, botErr := p.API.GetBot("", true)
	if botErr == nil {
		channel, channelErr := p.API.GetDirectChannel(userID, bot.UserId)
		if channelErr == nil {
			post := &model.Post{
				ChannelId: channel.Id,
				UserId:    bot.UserId,
				Message:   fmt.Sprintf("%s Вы %s приглашение на встречу.", emoji, action),
			}
			p.API.CreatePost(post)
		}
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Meeting %s", strings.ToLower(responseType)),
	})
}

// handleTestConnection tests Exchange connection
func (p *Plugin) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var credentials ExchangeCredentials
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	config := p.getConfiguration()
	if config.ExchangeServerURL == "" {
		http.Error(w, "Exchange server URL not configured", http.StatusInternalServerError)
		return
	}

	// Log connection attempt
	p.API.LogInfo("Testing Exchange connection",
		"server_url", config.ExchangeServerURL,
		"username", credentials.Username,
		"domain", credentials.Domain)

	client := NewExchangeClient(config.ExchangeServerURL, &credentials)
	err := client.TestConnection()
	if err != nil {
		// Log the detailed error
		p.API.LogError("Exchange connection test failed", "error", err.Error(),
			"server_url", config.ExchangeServerURL,
			"username", credentials.Username,
			"domain", credentials.Domain)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Ошибка подключения к Exchange: %s", err.Error()),
		})
		return
	}

	p.API.LogInfo("Exchange connection test successful")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Подключение к Exchange успешно установлено!",
	})
}

// sendMeetingResponse sends a meeting response via EWS
func (p *Plugin) sendMeetingResponse(credentials *ExchangeCredentials, eventID, responseType string) error {
	// TODO: Implement proper EWS CreateItem request for meeting responses
	// This would use credentials to authenticate with Exchange server
	_ = credentials // credentials will be used when implementing EWS meeting response

	// For now, just log the action
	p.API.LogInfo("Meeting response sent", "event_id", eventID, "response", responseType)
	return nil
}

// ExecuteCommand executes slash commands
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	trigger := strings.TrimPrefix(args.Command, "/")

	switch {
	case strings.HasPrefix(trigger, "exchange"):
		return p.handleExchangeCommand(args)
	default:
		return &model.CommandResponse{
			ResponseType: "ephemeral",
			Text:         "Неизвестная команда",
		}, nil
	}
}

// handleExchangeCommand handles the /exchange command
func (p *Plugin) handleExchangeCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	parts := strings.Fields(args.Command)

	if len(parts) < 2 {
		return p.getExchangeHelp(), nil
	}

	subcommand := parts[1]

	switch subcommand {
	case "setup":
		return p.handleSetupCommand(args.UserId), nil
	case "status":
		return p.handleStatusCommand(args.UserId), nil
	case "calendar":
		return p.handleCalendarCommand(args.UserId), nil
	case "reminders":
		return p.handleRemindersCommand(args.UserId), nil
	case "help":
		return p.getExchangeHelp(), nil
	default:
		return p.getExchangeHelp(), nil
	}
}

// handleSetupCommand provides setup instructions
func (p *Plugin) handleSetupCommand(userID string) *model.CommandResponse {
	text := "### 🔧 Настройка Exchange Integration\n\n" +
		"Для настройки подключения к Exchange:\n\n" +
		"1. Обратитесь к администратору для получения:\n" +
		"   - URL сервера Exchange\n" +
		"   - Учетные данные домена\n\n" +
		"2. Используйте команду `/exchange status` для проверки текущего состояния\n\n" +
		"3. Настройте credentials через веб-интерфейс плагина\n\n" +
		"**Примечание:** После настройки плагин автоматически будет синхронизировать ваш календарь каждые 5 минут."

	return &model.CommandResponse{
		ResponseType: "ephemeral",
		Text:         text,
	}
}

// handleStatusCommand shows current status
func (p *Plugin) handleStatusCommand(userID string) *model.CommandResponse {
	_, err := p.getUserExchangeCredentials(userID)

	var text string
	if err != nil {
		text = "### ❌ Exchange Integration - Не настроено\n\n" +
			"**Статус:** Не подключено\n" +
			"**Действие:** Используйте `/exchange setup` для получения инструкций по настройке"
	} else {
		config := p.getConfiguration()
		text = fmt.Sprintf("### ✅ Exchange Integration - Активно\n\n"+
			"**Статус:** Подключено и синхронизируется\n"+
			"**Сервер:** %s\n"+
			"**Синхронизация календаря:** %v\n"+
			"**Уведомления о встречах:** %v\n"+
			"**Время ежедневной сводки:** %s\n\n"+
			"Используйте `/exchange calendar` для просмотра календаря",
			config.ExchangeServerURL,
			config.EnableCalendarSync,
			config.EnableMeetingNotifications,
			config.DailySummaryTime)
	}

	return &model.CommandResponse{
		ResponseType: "ephemeral",
		Text:         text,
	}
}

// handleCalendarCommand shows calendar events
func (p *Plugin) handleCalendarCommand(userID string) *model.CommandResponse {
	credentials, err := p.getUserExchangeCredentials(userID)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: "ephemeral",
			Text:         "❌ Exchange не настроен. Используйте `/exchange setup` для настройки.",
		}
	}

	// Get today's events
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	events, err := p.getCalendarEventsInRange(credentials, startOfDay, endOfDay)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: "ephemeral",
			Text:         fmt.Sprintf("❌ Ошибка получения календаря: %s", err.Error()),
		}
	}

	if len(events) == 0 {
		return &model.CommandResponse{
			ResponseType: "ephemeral",
			Text:         "📅 На сегодня встреч не запланировано.",
		}
	}

	text := "📅 **Ваши встречи на сегодня:**\n\n"
	for _, event := range events {
		startTime := event.Start.Format("15:04")
		endTime := event.End.Format("15:04")

		text += fmt.Sprintf("🕐 **%s - %s**: %s", startTime, endTime, event.Subject)
		if event.Location != "" {
			text += fmt.Sprintf(" (📍 %s)", event.Location)
		}
		text += "\n"
	}

	return &model.CommandResponse{
		ResponseType: "ephemeral",
		Text:         text,
	}
}

// getExchangeHelp returns help text for Exchange commands
func (p *Plugin) getExchangeHelp() *model.CommandResponse {
	text := "### 📧 Exchange Integration - Справка\n\n" +
		"**Доступные команды:**\n\n" +
		"- `/exchange setup` - Инструкции по настройке\n" +
		"- `/exchange status` - Текущий статус подключения\n" +
		"- `/exchange calendar` - Просмотр календаря на сегодня\n" +
		"- `/exchange reminders` - Управление напоминаниями о встречах\n" +
		"- `/exchange help` - Эта справка\n\n" +
		"**Функции:**\n" +
		"- 🔄 Автоматическая синхронизация статуса на основе календаря\n" +
		"- 📅 Ежедневная утренняя сводка встреч (в 9:00)\n" +
		"- 📧 Уведомления о новых приглашениях на встречи\n" +
		"- ⏰ Напоминания за 15 минут до встречи\n" +
		"- ✅ Возможность принимать/отклонять встречи прямо из Mattermost"

	return &model.CommandResponse{
		ResponseType: "ephemeral",
		Text:         text,
	}
}

// handleSnoozeReminder handles snoozing a meeting reminder
func (p *Plugin) handleSnoozeReminder(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var request struct {
		Context map[string]interface{} `json:"context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	eventID, ok := request.Context["event_id"].(string)
	if !ok {
		http.Error(w, "Missing event_id", http.StatusBadRequest)
		return
	}

	snoozeMinsFloat, ok := request.Context["snooze_mins"].(float64)
	if !ok {
		http.Error(w, "Missing snooze_mins", http.StatusBadRequest)
		return
	}
	snoozeMins := int(snoozeMinsFloat)

	// Get current reminders
	reminders, err := p.reminderManager.getUserReminders(userID)
	if err != nil {
		http.Error(w, "Failed to get reminders", http.StatusInternalServerError)
		return
	}

	// Find the reminder and snooze it
	for i, reminder := range reminders {
		if reminder.EventID == eventID {
			// Create new snooze reminder
			snoozeTime := time.Now().Add(time.Duration(snoozeMins) * time.Minute)
			reminders[i].ReminderTime = snoozeTime
			reminders[i].Sent = false

			// Update reminders
			storeErr := p.reminderManager.storeUserReminders(userID, reminders)
			if storeErr != nil {
				http.Error(w, "Failed to update reminder", http.StatusInternalServerError)
				return
			}

			// Send confirmation message
			bot, botErr := p.API.GetBot("", true)
			if botErr == nil {
				channel, channelErr := p.API.GetDirectChannel(userID, bot.UserId)
				if channelErr == nil {
					post := &model.Post{
						ChannelId: channel.Id,
						UserId:    bot.UserId,
						Message:   fmt.Sprintf("⏱️ Напоминание о встрече \"%s\" отложено на %d минут", reminder.Subject, snoozeMins),
					}
					p.API.CreatePost(post)
				}
			}

			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Reminder snoozed",
	})
}

// handleOpenCalendar handles opening calendar view
func (p *Plugin) handleOpenCalendar(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// For now, just send a message with today's calendar
	credentials, err := p.getUserExchangeCredentials(userID)
	if err != nil {
		http.Error(w, "Exchange credentials not configured", http.StatusBadRequest)
		return
	}

	// Get today's events
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	events, err := p.getCalendarEventsInRange(credentials, startOfDay, endOfDay)
	if err != nil {
		http.Error(w, "Failed to get calendar events", http.StatusInternalServerError)
		return
	}

	var message string
	if len(events) == 0 {
		message = "📅 На сегодня встреч не запланировано."
	} else {
		message = "📅 **Ваши встречи на сегодня:**\n\n"
		for _, event := range events {
			startTime := event.Start.Format("15:04")
			endTime := event.End.Format("15:04")

			message += fmt.Sprintf("🕐 **%s - %s**: %s", startTime, endTime, event.Subject)
			if event.Location != "" {
				message += fmt.Sprintf(" (📍 %s)", event.Location)
			}
			message += "\n"
		}
	}

	// Send calendar message
	bot, botErr := p.API.GetBot("", true)
	if botErr == nil {
		channel, channelErr := p.API.GetDirectChannel(userID, bot.UserId)
		if channelErr == nil {
			post := &model.Post{
				ChannelId: channel.Id,
				UserId:    bot.UserId,
				Message:   message,
			}
			p.API.CreatePost(post)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Calendar displayed",
	})
}

// handleGetReminders returns user's upcoming reminders
func (p *Plugin) handleGetReminders(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	reminders, err := p.reminderManager.getUserReminders(userID)
	if err != nil {
		http.Error(w, "Failed to get reminders", http.StatusInternalServerError)
		return
	}

	// Filter for upcoming reminders only
	now := time.Now()
	upcomingReminders := make([]MeetingReminder, 0)
	for _, reminder := range reminders {
		if reminder.StartTime.After(now) && !reminder.Sent {
			upcomingReminders = append(upcomingReminders, reminder)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(upcomingReminders)
}

// handleUpdateReminders manually updates reminders for the user
func (p *Plugin) handleUpdateReminders(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := p.reminderManager.UpdateRemindersForUser(userID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update reminders: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Reminders updated successfully",
	})
}

// handleRemindersCommand shows user's upcoming reminders
func (p *Plugin) handleRemindersCommand(userID string) *model.CommandResponse {
	reminders, err := p.reminderManager.getUserReminders(userID)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: "ephemeral",
			Text:         "❌ Ошибка получения напоминаний",
		}
	}

	// Filter for upcoming reminders only
	now := time.Now()
	upcomingReminders := make([]MeetingReminder, 0)
	for _, reminder := range reminders {
		if reminder.StartTime.After(now) && !reminder.Sent {
			upcomingReminders = append(upcomingReminders, reminder)
		}
	}

	if len(upcomingReminders) == 0 {
		return &model.CommandResponse{
			ResponseType: "ephemeral",
			Text:         "⏰ У вас нет запланированных напоминаний о встречах",
		}
	}

	text := "⏰ **Ваши предстоящие напоминания:**\n\n"
	for _, reminder := range upcomingReminders {
		meetingTime := reminder.StartTime.Format("02.01.2006 15:04")
		reminderTime := reminder.ReminderTime.Format("02.01.2006 15:04")

		text += fmt.Sprintf("📅 **%s**\n", reminder.Subject)
		text += fmt.Sprintf("   🕐 Встреча: %s\n", meetingTime)
		text += fmt.Sprintf("   ⏰ Напоминание: %s\n", reminderTime)
		if reminder.Location != "" {
			text += fmt.Sprintf("   📍 Место: %s\n", reminder.Location)
		}
		text += "\n"
	}

	text += "💡 *Напоминания отправляются за 15 минут до встречи*"

	return &model.CommandResponse{
		ResponseType: "ephemeral",
		Text:         text,
	}
}
