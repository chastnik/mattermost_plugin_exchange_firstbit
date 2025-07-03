export enum ActionTypes {
    OPEN_EXCHANGE_SETTINGS_MODAL = 'OPEN_EXCHANGE_SETTINGS_MODAL',
    CLOSE_EXCHANGE_SETTINGS_MODAL = 'CLOSE_EXCHANGE_SETTINGS_MODAL',
    SET_EXCHANGE_CREDENTIALS = 'SET_EXCHANGE_CREDENTIALS',
    TEST_EXCHANGE_CONNECTION = 'TEST_EXCHANGE_CONNECTION',
}

export interface ExchangeCredentials {
    username: string;
    password: string;
    domain: string;
}

export interface CalendarEvent {
    id: string;
    subject: string;
    start: string;
    end: string;
    location: string;
    organizer: string;
    is_all_day: boolean;
    is_meeting: boolean;
    status: string;
}

export interface PluginState {
    isSettingsModalOpen: boolean;
    credentials: ExchangeCredentials | null;
    isTestingConnection: boolean;
    connectionTestResult: {
        success: boolean;
        message: string;
    } | null;
} 