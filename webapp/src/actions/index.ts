import {ActionTypes} from '../types';

export const openExchangeSettingsModal = () => ({
    type: ActionTypes.OPEN_EXCHANGE_SETTINGS_MODAL,
    // For Mattermost 9.x compatibility
    meta: {
        pluginId: 'com.mattermost.exchange-plugin'
    }
});

export const closeExchangeSettingsModal = () => ({
    type: ActionTypes.CLOSE_EXCHANGE_SETTINGS_MODAL,
    // For Mattermost 9.x compatibility  
    meta: {
        pluginId: 'com.mattermost.exchange-plugin'
    }
});

export const setExchangeCredentials = (credentials: any) => ({
    type: ActionTypes.SET_EXCHANGE_CREDENTIALS,
    payload: credentials,
});

export const testExchangeConnection = (credentials: any) => ({
    type: ActionTypes.TEST_EXCHANGE_CONNECTION,
    payload: credentials,
}); 