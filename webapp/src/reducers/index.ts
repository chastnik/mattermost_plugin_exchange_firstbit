import {ActionTypes, PluginState} from '../types';

const initialState: PluginState = {
    isSettingsModalOpen: false,
    credentials: null,
    isTestingConnection: false,
    connectionTestResult: null,
};

export default function reducer(state = initialState, action: any): PluginState {
    console.log('Exchange Plugin: Reducer called with action:', action);
    
    switch (action.type) {
        case ActionTypes.OPEN_EXCHANGE_SETTINGS_MODAL:
        case ActionTypes.LEGACY_OPEN_EXCHANGE_SETTINGS_MODAL:
            console.log('Exchange Plugin: Reducer - Opening modal');
            return {
                ...state,
                isSettingsModalOpen: true,
            };
        case ActionTypes.CLOSE_EXCHANGE_SETTINGS_MODAL:
        case ActionTypes.LEGACY_CLOSE_EXCHANGE_SETTINGS_MODAL:
            console.log('Exchange Plugin: Reducer - Closing modal');
            return {
                ...state,
                isSettingsModalOpen: false,
                connectionTestResult: null,
            };
        case ActionTypes.SET_EXCHANGE_CREDENTIALS:
            return {
                ...state,
                credentials: action.payload,
            };
        case ActionTypes.TEST_EXCHANGE_CONNECTION:
            return {
                ...state,
                isTestingConnection: true,
                connectionTestResult: null,
            };
        default:
            console.log('Exchange Plugin: Reducer - Unknown action type:', action.type);
            return state;
    }
} 