declare module 'mattermost-redux/types/store' {
    export interface GlobalState {
        plugins?: {
            plugins?: {
                [key: string]: any;
            };
        };
    }
}

declare module 'mattermost-redux/client' {
    export const Client4: {
        getUrl(): string;
    };
} 