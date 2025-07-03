package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
)

// MeetingReminder represents a scheduled reminder
type MeetingReminder struct {
	UserID       string    `json:"user_id"`
	EventID      string    `json:"event_id"`
	Subject      string    `json:"subject"`
	StartTime    time.Time `json:"start_time"`
	Location     string    `json:"location"`
	ReminderTime time.Time `json:"reminder_time"`
	Sent         bool      `json:"sent"`
}

// ReminderManager manages meeting reminders
type ReminderManager struct {
	plugin *Plugin
}

// NewReminderManager creates a new reminder manager
func NewReminderManager(plugin *Plugin) *ReminderManager {
	return &ReminderManager{
		plugin: plugin,
	}
}

// CheckAndSendReminders checks for due reminders and sends them
func (rm *ReminderManager) CheckAndSendReminders() {
	config := rm.plugin.getConfiguration()
	if !config.EnableMeetingReminders {
		return
	}

	now := time.Now()

	// Get all users to check their reminders
	users, err := rm.plugin.API.GetUsers(&model.UserGetOptions{
		Page:    0,
		PerPage: 1000,
	})
	if err != nil {
		rm.plugin.API.LogError("Ошибка получения пользователей для напоминаний", "error", err.Error())
		return
	}

	for _, user := range users {
		rm.checkUserReminders(user.Id, now)
	}
}

// checkUserReminders checks and sends due reminders for a specific user
func (rm *ReminderManager) checkUserReminders(userID string, now time.Time) {
	reminders, err := rm.getUserReminders(userID)
	if err != nil {
		rm.plugin.API.LogError("Ошибка получения напоминаний пользователя", "user_id", userID, "error", err.Error())
		return
	}

	for _, reminder := range reminders {
		// Skip already sent reminders
		if reminder.Sent {
			continue
		}

		// Check if it's time to send the reminder (with 1-minute tolerance)
		if now.After(reminder.ReminderTime) && now.Before(reminder.ReminderTime.Add(2*time.Minute)) {
			if err := rm.sendReminder(reminder); err != nil {
				rm.plugin.API.LogError("Ошибка отправки напоминания", "user_id", userID, "event_id", reminder.EventID, "error", err.Error())
				continue
			}

			// Mark reminder as sent
			reminder.Sent = true
			addErr := rm.addUserReminder(reminder)
			if addErr != nil {
				rm.plugin.API.LogError("Ошибка обновления статуса напоминания", "user_id", userID, "event_id", reminder.EventID, "error", addErr.Error())
			}
		}

		// Clean up old reminders (older than 1 hour after meeting start)
		if now.After(reminder.StartTime.Add(time.Hour)) {
			rm.deleteReminder(userID, reminder.EventID)
		}
	}
}

// sendReminder sends a reminder notification to the user
func (rm *ReminderManager) sendReminder(reminder MeetingReminder) error {
	timeUntilMeeting := time.Until(reminder.StartTime)
	minutesLeft := int(timeUntilMeeting.Minutes())

	var timeText string
	if minutesLeft <= 1 {
		timeText = "менее чем через минуту"
	} else if minutesLeft < 60 {
		timeText = fmt.Sprintf("через %d мин", minutesLeft)
	} else {
		timeText = fmt.Sprintf("через %d час %d мин", minutesLeft/60, minutesLeft%60)
	}

	config := rm.plugin.getConfiguration()
	reminderMins := "15"
	if config.ReminderMinutesBefore != "" {
		reminderMins = config.ReminderMinutesBefore
	}

	message := "⏰ **Напоминание о встрече**\n\n"
	message += fmt.Sprintf("**Встреча:** %s\n", reminder.Subject)
	message += fmt.Sprintf("**Начало:** %s (%s)\n", reminder.StartTime.Format("15:04"), timeText)

	if reminder.Location != "" {
		message += fmt.Sprintf("**Место:** %s\n", reminder.Location)
	}

	message += fmt.Sprintf("\n💡 *Это напоминание отправлено за %s минут до встречи*", reminderMins)

	// Create attachment with quick actions
	attachments := []*model.SlackAttachment{
		{
			Actions: []*model.PostAction{
				{
					Id:   "snooze_reminder",
					Name: "⏱️ Напомнить через 5 мин",
					Type: "button",
					Integration: &model.PostActionIntegration{
						URL: "/plugins/com.mattermost.exchange-plugin/api/v1/reminder/snooze",
						Context: map[string]interface{}{
							"event_id":    reminder.EventID,
							"user_id":     reminder.UserID,
							"snooze_mins": 5,
						},
					},
				},
				{
					Id:   "view_calendar",
					Name: "📅 Открыть календарь",
					Type: "button",
					Integration: &model.PostActionIntegration{
						URL: "/plugins/com.mattermost.exchange-plugin/api/v1/calendar/open",
						Context: map[string]interface{}{
							"user_id": reminder.UserID,
						},
					},
				},
			},
		},
	}

	// Send direct message to user
	bot, err := rm.plugin.API.GetBot("", true)
	if err != nil {
		return fmt.Errorf("failed to get bot: %w", err)
	}

	channel, err := rm.plugin.API.GetDirectChannel(reminder.UserID, bot.UserId)
	if err != nil {
		return fmt.Errorf("failed to get direct channel: %w", err)
	}

	post := &model.Post{
		ChannelId: channel.Id,
		UserId:    bot.UserId,
		Message:   message,
		Props: map[string]interface{}{
			"attachments": attachments,
		},
	}

	_, err = rm.plugin.API.CreatePost(post)
	if err != nil {
		return fmt.Errorf("failed to create reminder post: %w", err)
	}

	rm.plugin.API.LogInfo("Напоминание отправлено", "user_id", reminder.UserID, "event_id", reminder.EventID, "subject", reminder.Subject)
	return nil
}

// getUserReminders retrieves all reminders for a user
func (rm *ReminderManager) getUserReminders(userID string) ([]MeetingReminder, error) {
	key := fmt.Sprintf("user_reminders_%s", userID)

	data, err := rm.plugin.API.KVGet(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get user reminders: %w", err)
	}

	if data == nil {
		return []MeetingReminder{}, nil
	}

	var reminders []MeetingReminder
	if err := json.Unmarshal(data, &reminders); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reminders: %w", err)
	}

	return reminders, nil
}

// storeUserReminders stores all reminders for a user
func (rm *ReminderManager) storeUserReminders(userID string, reminders []MeetingReminder) error {
	key := fmt.Sprintf("user_reminders_%s", userID)

	data, err := json.Marshal(reminders)
	if err != nil {
		return fmt.Errorf("failed to marshal reminders: %w", err)
	}

	return rm.plugin.API.KVSet(key, data)
}

// deleteReminder deletes a specific reminder
func (rm *ReminderManager) deleteReminder(userID, eventID string) error {
	reminders, err := rm.getUserReminders(userID)
	if err != nil {
		return err
	}

	// Filter out the reminder to delete
	filteredReminders := make([]MeetingReminder, 0, len(reminders))
	for _, reminder := range reminders {
		if reminder.EventID != eventID {
			filteredReminders = append(filteredReminders, reminder)
		}
	}

	return rm.storeUserReminders(userID, filteredReminders)
}

// addUserReminder adds a reminder to user's reminder list
func (rm *ReminderManager) addUserReminder(reminder MeetingReminder) error {
	reminders, err := rm.getUserReminders(reminder.UserID)
	if err != nil {
		return err
	}

	// Check if reminder already exists (prevent duplicates)
	for i, existing := range reminders {
		if existing.EventID == reminder.EventID {
			// Update existing reminder
			reminders[i] = reminder
			return rm.storeUserReminders(reminder.UserID, reminders)
		}
	}

	// Add new reminder
	reminders = append(reminders, reminder)
	return rm.storeUserReminders(reminder.UserID, reminders)
}

// UpdateRemindersForUser updates reminders for a specific user based on their calendar
func (rm *ReminderManager) UpdateRemindersForUser(userID string) error {
	config := rm.plugin.getConfiguration()
	if !config.EnableMeetingReminders {
		return nil
	}

	credentials, err := rm.plugin.getUserExchangeCredentials(userID)
	if err != nil {
		// User doesn't have Exchange configured
		return nil
	}

	// Parse reminder minutes from config
	reminderMins := 15 // default
	if mins, err := strconv.Atoi(config.ReminderMinutesBefore); err == nil && mins > 0 {
		reminderMins = mins
	}

	// Get upcoming events (next 7 days)
	now := time.Now()
	endTime := now.Add(7 * 24 * time.Hour)

	events, err := rm.plugin.getCalendarEventsInRange(credentials, now, endTime)
	if err != nil {
		return fmt.Errorf("failed to get calendar events: %w", err)
	}

	// Clear existing reminders and schedule new ones
	if err := rm.storeUserReminders(userID, []MeetingReminder{}); err != nil {
		return fmt.Errorf("failed to clear existing reminders: %w", err)
	}

	// Schedule new reminders
	for _, event := range events {
		// Only schedule reminders for future meetings
		if event.Start.Before(now) {
			continue
		}

		// Calculate reminder time based on configuration
		reminderTime := event.Start.Add(-time.Duration(reminderMins) * time.Minute)

		// Don't schedule reminders for meetings starting within the reminder window
		if reminderTime.Before(now) {
			continue
		}

		reminder := MeetingReminder{
			UserID:       userID,
			EventID:      event.ID,
			Subject:      event.Subject,
			StartTime:    event.Start,
			Location:     event.Location,
			ReminderTime: reminderTime,
			Sent:         false,
		}

		if err := rm.addUserReminder(reminder); err != nil {
			rm.plugin.API.LogError("Ошибка добавления напоминания", "user_id", userID, "event_id", event.ID, "error", err.Error())
		}
	}

	return nil
}
