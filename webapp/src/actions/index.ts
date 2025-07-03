import {ActionTypes} from '../types';

export const openExchangeSettingsModal = () => ({
    type: ActionTypes.OPEN_EXCHANGE_SETTINGS_MODAL,
});

export const closeExchangeSettingsModal = () => ({
    type: ActionTypes.CLOSE_EXCHANGE_SETTINGS_MODAL,
});

export const setExchangeCredentials = (credentials: any) => ({
    type: ActionTypes.SET_EXCHANGE_CREDENTIALS,
    payload: credentials,
});

export const testExchangeConnection = (credentials: any) => ({
    type: ActionTypes.TEST_EXCHANGE_CONNECTION,
    payload: credentials,
}); 