export interface PluginRegistry {
    registerRootComponent(component: React.ComponentType): void;
    registerMainMenuAction(
        text: string,
        action: () => void,
        mobileIcon?: string
    ): void;
    registerChannelHeaderButtonAction(
        icon: string,
        action: () => void,
        dropdownText?: string,
        tooltipText?: string
    ): void;
    registerPostTypeComponent(typeName: string, component: React.ComponentType): void;
    registerReducer(reducer: any): void;
    unregisterComponent(componentId: string): void;
}

declare global {
    interface Window {
        MattermostRedux: {
            Client4: any;
            Types: any;
        };
        plugins: {
            [key: string]: any;
        };
    }
} 