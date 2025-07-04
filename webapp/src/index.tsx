import React from 'react';

import ExchangeSettingsModal from './components/exchange_settings_modal';
import {openExchangeSettingsModal} from './actions';
import reducer from './reducers';

class Plugin {
    public async initialize(registry: any, store: any) {
        console.log('Exchange Plugin: Initializing...');
        
        try {
            // Register reducer
            registry.registerReducer(reducer);
            console.log('Exchange Plugin: Reducer registered');

            // Register modal for Exchange settings
            registry.registerRootComponent(ExchangeSettingsModal);
            console.log('Exchange Plugin: Modal component registered');

            // Add menu item in account settings
            registry.registerMainMenuAction(
                'Exchange Settings',
                () => store.dispatch(openExchangeSettingsModal()),
                () => true
            );
            console.log('Exchange Plugin: Main menu action registered');

            // Add channel header button
            registry.registerChannelHeaderButtonAction(
                '📧',
                () => store.dispatch(openExchangeSettingsModal()),
                'Exchange Integration',
                'Exchange Integration Settings'
            );
            console.log('Exchange Plugin: Channel header button registered');
            
            console.log('Exchange Plugin: Initialization complete');
        } catch (error) {
            console.error('Exchange Plugin: Initialization failed', error);
        }
    }
}

export default Plugin;

// Make plugin available globally
if (typeof window !== 'undefined') {
    (window as any).ExchangePlugin = {
        default: Plugin
    };
} 