package main

import (
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

const (
	CommandTriggerExchange = "exchange"
)

func (p *Plugin) registerCommands() error {
	if err := p.API.RegisterCommand(createExchangeCommand()); err != nil {
		return errors.Wrap(err, "failed to register exchange command")
	}
	return nil
}

func createExchangeCommand() *model.Command {
	return &model.Command{
		Trigger:          CommandTriggerExchange,
		Method:           "POST",
		Username:         "exchange",
		IconURL:          "",
		AutoComplete:     true,
		AutoCompleteDesc: "Управление интеграцией с Exchange",
		AutoCompleteHint: "[setup|status|calendar|help]",
		DisplayName:      "Exchange Integration",
		Description:      "Команды для управления интеграцией с Microsoft Exchange",
		URL:              "",
	}
}

func (p *Plugin) unregisterCommands() error {
	return p.API.UnregisterCommand("", CommandTriggerExchange)
}
