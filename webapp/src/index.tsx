import React from 'react';

import ExchangeSettingsModal from './components/exchange_settings_modal';
import {openExchangeSettingsModal} from './actions';
import reducer from './reducers';

export default class Plugin {
    public async initialize(registry: any, store: any) {
        // Register reducer
        registry.registerReducer(reducer);

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