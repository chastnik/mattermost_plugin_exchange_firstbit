package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
)

// Plugin implements the interface expected by the Mattermost server to communicate with the plugin
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// scheduler for periodic tasks
	scheduler *Scheduler

	// reminder manager for meeting notifications
	reminderManager *ReminderManager
}

// ExchangeCredentials represents user's Exchange credentials
type ExchangeCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Domain   string `json:"domain"`
}

// CalendarEvent represents a calendar event from Exchange
type CalendarEvent struct {
	ID        string    `json:"id"`
	Subject   string    `json:"subject"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Location  string    `json:"location"`
	Organizer string    `json:"organizer"`
	IsAllDay  bool      `json:"is_all_day"`
	IsMeeting bool      `json:"is_meeting"`
	Status    string    `json:"status"` // Free, Busy, Tentative, OutOfOffice
}

// OnActivate is called when the plugin is activated
func (p *Plugin) OnActivate() error {
	p.API.LogInfo("Exchange Integration Plugin активирован")

	// Register slash commands
	if err := p.registerCommands(); err != nil {
		return err
	}

	// Initialize reminder manager
	p.reminderManager = NewReminderManager(p)

	// Initialize scheduler
	p.scheduler = NewScheduler(p)

	// Start periodic tasks
	go p.startPeriodicTasks()

	return nil
}

// OnDeactivate is called when the plugin is deactivated
func (p *Plugin) OnDeactivate() error {
	p.API.LogInfo("Exchange Integration Plugin деактивирован")

	// Unregister commands
	p.unregisterCommands()

	if p.scheduler != nil {
		p.scheduler.Stop()
	}

	return nil
}

// startPeriodicTasks starts all periodic background tasks
func (p *Plugin) startPeriodicTasks() {
	// Start calendar sync every 5 minutes
	p.scheduler.AddJob("calendar_sync", 5*time.Minute, p.syncAllUsersCalendars)

	// Start daily summary job
	p.scheduler.AddJob("daily_summary", 24*time.Hour, p.sendDailySummaries)

	// Start meeting notifications check every minute
	p.scheduler.AddJob("meeting_notifications", 1*time.Minute, p.checkMeetingNotifications)

	// Start reminder checks every minute
	p.scheduler.AddJob("reminder_check", 1*time.Minute, p.reminderManager.CheckAndSendReminders)

	// Update reminders every 30 minutes
	p.scheduler.AddJob("reminder_update", 30*time.Minute, p.updateAllUsersReminders)
}

// syncAllUsersCalendars syncs calendars for all connected users
func (p *Plugin) syncAllUsersCalendars() {
	config := p.getConfiguration()
	if !config.EnableCalendarSync {
		return
	}

	users, err := p.API.GetUsers(&model.UserGetOptions{
		Page:    0,
		PerPage: 1000,
	})
	if err != nil {
		p.API.LogError("Ошибка получения пользователей", "error", err.Error())
		return
	}

	for _, user := range users {
		go p.syncUserCalendar(user.Id)
	}
}

// syncUserCalendar syncs calendar for a specific user and updates their status
func (p *Plugin) syncUserCalendar(userID string) {
	credentials, err := p.getUserExchangeCredentials(userID)
	if err != nil {
		// User hasn't configured Exchange credentials
		return
	}

	events, err := p.getCalendarEvents(credentials)
	if err != nil {
		p.API.LogError("Ошибка получения календарных событий", "user_id", userID, "error", err.Error())
		return
	}

	// Update user status based on current calendar events
	p.updateUserStatusFromCalendar(userID, events)

	// Update reminders for the user
	if err := p.reminderManager.UpdateRemindersForUser(userID); err != nil {
		p.API.LogError("Ошибка обновления напоминаний", "user_id", userID, "error", err.Error())
	}
}

// updateUserStatusFromCalendar updates user's Mattermost status based on calendar events
func (p *Plugin) updateUserStatusFromCalendar(userID string, events []CalendarEvent) {
	now := time.Now()
	var currentEvent *CalendarEvent

	// Find current active event
	for _, event := range events {
		if now.After(event.Start) && now.Before(event.End) {
			currentEvent = &event
			break
		}
	}

	var status string
	var statusText string

	if currentEvent != nil {
		switch currentEvent.Status {
		case "Busy":
			status = "dnd"
			statusText = fmt.Sprintf("На встрече: %s", currentEvent.Subject)
		case "OutOfOffice":
			status = "away"
			statusText = "Не в офисе"
		case "Tentative":
			status = "away"
			statusText = fmt.Sprintf("Возможно занят: %s", currentEvent.Subject)
		default:
			status = "online"
			statusText = ""
		}
	} else {
		// Check if user has events in the next hour
		nextHour := now.Add(time.Hour)
		hasUpcomingEvents := false

		for _, event := range events {
			if event.Start.After(now) && event.Start.Before(nextHour) {
				hasUpcomingEvents = true
				break
			}
		}

		if hasUpcomingEvents {
			status = "online"
			statusText = "Скоро встреча"
		} else {
			status = "online"
			statusText = ""
		}
	}

	// Update user status
	_, err := p.API.UpdateUserStatus(userID, status)
	if err != nil {
		p.API.LogError("Ошибка обновления статуса пользователя", "user_id", userID, "error", err.Error())
	}

	// Update custom status if available
	if statusText != "" {
		p.API.UpdateUserCustomStatus(userID, &model.CustomStatus{
			Text: statusText,
		})
	}
}

// sendDailySummaries sends daily meeting summaries to all users
func (p *Plugin) sendDailySummaries() {
	config := p.getConfiguration()
	summaryTime := config.DailySummaryTime

	// Check if it's the right time for daily summary
	now := time.Now()
	targetTime, err := time.Parse("15:04", summaryTime)
	if err != nil {
		p.API.LogError("Неверный формат времени для ежедневной сводки", "time", summaryTime)
		return
	}

	currentTime := time.Date(0, 1, 1, now.Hour(), now.Minute(), 0, 0, time.UTC)
	if currentTime.Hour() != targetTime.Hour() || currentTime.Minute() != targetTime.Minute() {
		return
	}

	users, getUsersErr := p.API.GetUsers(&model.UserGetOptions{
		Page:    0,
		PerPage: 1000,
	})
	if getUsersErr != nil {
		p.API.LogError("Ошибка получения пользователей для ежедневной сводки", "error", getUsersErr.Error())
		return
	}

	for _, user := range users {
		go p.sendUserDailySummary(user.Id)
	}
}

// sendUserDailySummary sends daily meeting summary to a specific user
func (p *Plugin) sendUserDailySummary(userID string) {
	credentials, err := p.getUserExchangeCredentials(userID)
	if err != nil {
		return
	}

	// Get today's events
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	events, err := p.getCalendarEventsInRange(credentials, startOfDay, endOfDay)
	if err != nil {
		p.API.LogError("Ошибка получения событий для ежедневной сводки", "user_id", userID, "error", err.Error())
		return
	}

	if len(events) == 0 {
		return
	}

	// Create summary message
	message := "📅 **Ваши встречи на сегодня:**\n\n"
	for _, event := range events {
		startTime := event.Start.Format("15:04")
		endTime := event.End.Format("15:04")

		message += fmt.Sprintf("🕐 **%s - %s**: %s", startTime, endTime, event.Subject)
		if event.Location != "" {
			message += fmt.Sprintf(" (📍 %s)", event.Location)
		}
		message += "\n"
	}

	// Send direct message to user
	bot, botErr := p.API.GetBot("", true)
	if botErr != nil {
		p.API.LogError("Ошибка получения бота", "user_id", userID, "error", botErr.Error())
		return
	}

	channel, channelErr := p.API.GetDirectChannel(userID, bot.UserId)
	if channelErr != nil {
		p.API.LogError("Ошибка создания прямого канала", "user_id", userID, "error", channelErr.Error())
		return
	}

	post := &model.Post{
		ChannelId: channel.Id,
		UserId:    bot.UserId,
		Message:   message,
	}

	_, postErr := p.API.CreatePost(post)
	if postErr != nil {
		p.API.LogError("Ошибка отправки ежедневной сводки", "user_id", userID, "error", postErr.Error())
	}
}

// checkMeetingNotifications checks for new meeting invitations
func (p *Plugin) checkMeetingNotifications() {
	config := p.getConfiguration()
	if !config.EnableMeetingNotifications {
		return
	}

	users, err := p.API.GetUsers(&model.UserGetOptions{
		Page:    0,
		PerPage: 1000,
	})
	if err != nil {
		p.API.LogError("Ошибка получения пользователей для уведомлений", "error", err.Error())
		return
	}

	for _, user := range users {
		go p.checkUserMeetingNotifications(user.Id)
	}
}

// checkUserMeetingNotifications checks for new meeting invitations for a specific user
func (p *Plugin) checkUserMeetingNotifications(userID string) {
	credentials, err := p.getUserExchangeCredentials(userID)
	if err != nil {
		return
	}

	// Get new meeting invitations (this would need to be implemented with Exchange Web Services)
	invitations, err := p.getNewMeetingInvitations(credentials)
	if err != nil {
		p.API.LogError("Ошибка получения приглашений на встречи", "user_id", userID, "error", err.Error())
		return
	}

	for _, invitation := range invitations {
		p.sendMeetingInvitationNotification(userID, invitation)
	}
}

// sendMeetingInvitationNotification sends a notification about a new meeting invitation
func (p *Plugin) sendMeetingInvitationNotification(userID string, event CalendarEvent) {
	startTime := event.Start.Format("02.01.2006 15:04")

	message := "📧 **Новое приглашение на встречу**\n\n"
	message += fmt.Sprintf("**Тема:** %s\n", event.Subject)
	message += fmt.Sprintf("**Время:** %s\n", startTime)
	if event.Location != "" {
		message += fmt.Sprintf("**Место:** %s\n", event.Location)
	}
	message += fmt.Sprintf("**Организатор:** %s\n\n", event.Organizer)

	// Add action buttons
	attachments := []*model.SlackAttachment{
		{
			Actions: []*model.PostAction{
				{
					Id:   "accept_meeting",
					Name: "✅ Принять",
					Type: "button",
					Integration: &model.PostActionIntegration{
						URL: "/plugins/com.mattermost.exchange-plugin/api/v1/meeting/accept",
						Context: map[string]interface{}{
							"event_id": event.ID,
							"user_id":  userID,
						},
					},
				},
				{
					Id:   "decline_meeting",
					Name: "❌ Отклонить",
					Type: "button",
					Integration: &model.PostActionIntegration{
						URL: "/plugins/com.mattermost.exchange-plugin/api/v1/meeting/decline",
						Context: map[string]interface{}{
							"event_id": event.ID,
							"user_id":  userID,
						},
					},
				},
				{
					Id:   "tentative_meeting",
					Name: "❓ Под вопросом",
					Type: "button",
					Integration: &model.PostActionIntegration{
						URL: "/plugins/com.mattermost.exchange-plugin/api/v1/meeting/tentative",
						Context: map[string]interface{}{
							"event_id": event.ID,
							"user_id":  userID,
						},
					},
				},
			},
		},
	}

	// Send direct message to user
	bot, botErr := p.API.GetBot("", true)
	if botErr != nil {
		p.API.LogError("Ошибка получения бота для уведомления", "user_id", userID, "error", botErr.Error())
		return
	}

	channel, channelErr := p.API.GetDirectChannel(userID, bot.UserId)
	if channelErr != nil {
		p.API.LogError("Ошибка создания прямого канала для уведомления", "user_id", userID, "error", channelErr.Error())
		return
	}

	post := &model.Post{
		ChannelId: channel.Id,
		UserId:    bot.UserId,
		Message:   message,
		Props: map[string]interface{}{
			"attachments": attachments,
		},
	}

	_, postErr := p.API.CreatePost(post)
	if postErr != nil {
		p.API.LogError("Ошибка отправки уведомления о встрече", "user_id", userID, "error", postErr.Error())
	}
}

// getUserExchangeCredentials retrieves user's Exchange credentials
func (p *Plugin) getUserExchangeCredentials(userID string) (*ExchangeCredentials, error) {
	data, err := p.API.KVGet(fmt.Sprintf("exchange_creds_%s", userID))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user credentials")
	}

	if data == nil {
		return nil, errors.New("no credentials found")
	}

	var credentials ExchangeCredentials
	if err := json.Unmarshal(data, &credentials); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal credentials")
	}

	return &credentials, nil
}

// This will be implemented in exchange.go

// updateAllUsersReminders updates reminders for all users
func (p *Plugin) updateAllUsersReminders() {
	users, err := p.API.GetUsers(&model.UserGetOptions{
		Page:    0,
		PerPage: 1000,
	})
	if err != nil {
		p.API.LogError("Ошибка получения пользователей для обновления напоминаний", "error", err.Error())
		return
	}

	for _, user := range users {
		go func(userID string) {
			if err := p.reminderManager.UpdateRemindersForUser(userID); err != nil {
				p.API.LogError("Ошибка обновления напоминаний пользователя", "user_id", userID, "error", err.Error())
			}
		}(user.Id)
	}
}
