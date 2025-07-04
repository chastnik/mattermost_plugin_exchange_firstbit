import React from 'react';

import ExchangeSettingsModal from './components/exchange_settings_modal';
import {openExchangeSettingsModal} from './actions';
import reducer from './reducers';

class Plugin {
    public async initialize(registry: any, store: any) {
        console.log('Exchange Plugin: Initializing...');
        console.log('Exchange Plugin: Registry methods:', Object.keys(registry));
        
        try {
            // Register reducer for plugin state
            if (registry.registerReducer) {
                registry.registerReducer(reducer);
                console.log('Exchange Plugin: Reducer registered');
            }

            // Register modal component
            if (registry.registerRootComponent) {
                registry.registerRootComponent(ExchangeSettingsModal);
                console.log('Exchange Plugin: Modal component registered');
            }

            // Register main menu action (works in Mattermost 9.x)
            if (registry.registerMainMenuAction) {
                registry.registerMainMenuAction(
                    'Exchange Settings',
                    () => {
                        console.log('Exchange Plugin: Opening modal from main menu');
                        store.dispatch(openExchangeSettingsModal());
                    },
                    () => true
                );
                console.log('Exchange Plugin: Main menu action registered');
            }

            // Register channel header button (works in Mattermost 9.x)
            if (registry.registerChannelHeaderButtonAction) {
                registry.registerChannelHeaderButtonAction(
                    '📧',
                    () => {
                        console.log('Exchange Plugin: Opening modal from channel header');
                        store.dispatch(openExchangeSettingsModal());
                    },
                    'Exchange Integration',
                    'Exchange Integration Settings'
                );
                console.log('Exchange Plugin: Channel header button registered');
            }

            // Alternative registration for newer Mattermost versions
            if (registry.registerAppBarComponent) {
                registry.registerAppBarComponent(
                    '📧',
                    () => {
                        console.log('Exchange Plugin: Opening modal from app bar');
                        store.dispatch(openExchangeSettingsModal());
                    },
                    'Exchange Integration'
                );
                console.log('Exchange Plugin: App bar component registered');
            }
            
            console.log('Exchange Plugin: Initialization complete');
        } catch (error) {
            console.error('Exchange Plugin: Initialization failed', error);
        }
    }
}

export default Plugin;

// Ensure global availability for Mattermost
if (typeof window !== 'undefined') {
    if (!window.plugins) {
        window.plugins = {};
    }
    window.plugins['com.mattermost.exchange-plugin'] = {
        initialize: (registry: any, store: any) => {
            const plugin = new Plugin();
            return plugin.initialize(registry, store);
        }
    };
    
    // Legacy support
    (window as any).ExchangePlugin = {
        default: Plugin
    };
} 