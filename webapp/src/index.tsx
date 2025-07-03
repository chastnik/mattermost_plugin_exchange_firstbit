import React from 'react';
import {Store, Action} from 'redux';

import {GlobalState} from 'mattermost-redux/types/store';

import {PluginRegistry} from 'types/mattermost-webapp';

import ExchangeSettingsModal from './components/exchange_settings_modal';
import {openExchangeSettingsModal} from './actions';

export default class Plugin {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    public async initialize(registry: PluginRegistry, store: Store<GlobalState, Action<Record<string, unknown>>>) {
        // @see https://developers.mattermost.com/extend/plugins/webapp/reference/

        // Register modal for Exchange settings
        registry.registerRootComponent(ExchangeSettingsModal);

        // Add menu item in account settings
        registry.registerMainMenuAction(
            'Exchange Settings',
            () => store.dispatch(openExchangeSettingsModal()),
            () => true
        );

        // Add channel header button
        registry.registerChannelHeaderButtonAction(
            '📧',
            () => store.dispatch(openExchangeSettingsModal()),
            'Exchange Integration',
            'Exchange Integration Settings'
        );
    }
} 