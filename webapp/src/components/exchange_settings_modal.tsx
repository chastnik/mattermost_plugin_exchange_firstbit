import React, {useState} from 'react';
import {useSelector, useDispatch} from 'react-redux';

import {closeExchangeSettingsModal} from '../actions';
import {ExchangeCredentials} from '../types';

const ExchangeSettingsModal: React.FC = () => {
    const dispatch = useDispatch();
    // Try multiple paths for different Mattermost versions
    const isOpen = useSelector((state: any) => {
        console.log('Exchange Plugin: Current Redux state:', state);
        
        // Check different possible state paths for different Mattermost versions
        const paths = [
            // Mattermost 9.x format with dashes
            state['plugins-com.mattermost.exchange-plugin']?.isSettingsModalOpen,
            
            // Alternative formats
            state.plugins?.plugins?.['com.mattermost.exchange-plugin']?.isSettingsModalOpen,
            state['plugins/com.mattermost.exchange-plugin']?.isSettingsModalOpen,
            state.plugins?.['com.mattermost.exchange-plugin']?.isSettingsModalOpen,
            state.exchangePlugin?.isSettingsModalOpen,
            
            // Direct plugin key format
            state['com.mattermost.exchange-plugin']?.isSettingsModalOpen
        ];
        
        console.log('Exchange Plugin: Checking state paths:');
        paths.forEach((path, index) => {
            console.log(`  Path ${index}: ${path !== undefined ? path : 'undefined'}`);
        });
        
        const result = paths.find(path => path !== undefined) || false;
        console.log('Exchange Plugin: Modal isOpen:', result);
        return result;
    });
    
    const [credentials, setCredentials] = useState<ExchangeCredentials>({
        username: '',
        password: '',
        domain: '',
    });
    
    const [isTestingConnection, setIsTestingConnection] = useState(false);
    const [testResult, setTestResult] = useState<{success: boolean; message: string} | null>(null);
    const [isSaving, setIsSaving] = useState(false);

    const handleClose = () => {
        dispatch(closeExchangeSettingsModal());
        setTestResult(null);
    };

    const handleInputChange = (field: keyof ExchangeCredentials, value: string) => {
        setCredentials(prev => ({
            ...prev,
            [field]: value,
        }));
        setTestResult(null); // Clear test result when credentials change
    };

    const testConnection = async () => {
        if (!credentials.username || !credentials.password) {
            setTestResult({
                success: false,
                message: 'Пожалуйста, заполните имя пользователя и пароль',
            });
            return;
        }

        setIsTestingConnection(true);
        
        try {
            const response = await fetch(`/plugins/com.mattermost.exchange-plugin/api/v1/test-connection`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest',
                },
                body: JSON.stringify(credentials),
            });

            if (response.ok) {
                const result = await response.json();
                console.log('Exchange Plugin: Test connection response:', result);
                console.log('Exchange Plugin: Response message length:', result.message?.length || 0);
                
                if (result.success) {
                    setTestResult({
                        success: true,
                        message: result.message || 'Подключение к Exchange успешно установлено!'
                    });
                } else {
                    const errorMessage = result.message || result.error || 'Ошибка подключения к Exchange';
                    console.log('Exchange Plugin: Full error message:', errorMessage);
                    setTestResult({
                        success: false,
                        message: errorMessage
                    });
                }
            } else {
                const errorText = await response.text();
                console.error('Exchange Plugin: Test connection failed:', errorText);
                setTestResult({
                    success: false,
                    message: errorText || `Ошибка сервера: ${response.status}`
                });
            }
        } catch (error) {
            console.error('Exchange Plugin: Test connection error:', error);
            setTestResult({
                success: false,
                message: 'Ошибка подключения к серверу',
            });
        } finally {
            setIsTestingConnection(false);
        }
    };

    const saveCredentials = async () => {
        if (!credentials.username || !credentials.password) {
            setTestResult({
                success: false,
                message: 'Пожалуйста, заполните все обязательные поля',
            });
            return;
        }

        setIsSaving(true);
        
        try {
            const response = await fetch(`/plugins/com.mattermost.exchange-plugin/api/v1/credentials`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest',
                },
                body: JSON.stringify(credentials),
            });

            if (response.ok) {
                setTestResult({
                    success: true,
                    message: 'Учетные данные сохранены успешно!',
                });
                // Close modal after 2 seconds
                setTimeout(() => {
                    handleClose();
                }, 2000);
            } else {
                const errorText = await response.text();
                setTestResult({
                    success: false,
                    message: errorText || 'Ошибка сохранения учетных данных',
                });
            }
        } catch (error) {
            setTestResult({
                success: false,
                message: 'Ошибка подключения к серверу',
            });
        } finally {
            setIsSaving(false);
        }
    };

    // Force show modal for debugging
    const forceShow = window.exchangePluginForceShowModal || isOpen;
    console.log('Exchange Plugin: Modal render - isOpen:', isOpen, 'forceShow:', forceShow);
    
    if (!forceShow) {
        return null;
    }

    console.log('Exchange Plugin: Rendering modal with forceShow:', forceShow);
    
    // Simple test element without DOM manipulation
    if (forceShow && typeof document !== 'undefined') {
        console.log('Exchange Plugin: Modal should be visible now!');
    }
    
    return (
        <div 
            style={{
                display: forceShow ? 'block' : 'none', 
                position: 'fixed',
                top: 0,
                left: 0,
                width: '100vw',
                height: '100vh',
                backgroundColor: 'rgba(0, 0, 0, 0.5)',
                zIndex: 1050,
                padding: '20px',
                boxSizing: 'border-box'
            }}
            onClick={(e) => {
                if (e.target === e.currentTarget) {
                    handleClose();
                }
            }}
        >
            <div style={{
                maxWidth: '600px',
                margin: '0 auto',
                marginTop: '50px',
                backgroundColor: 'var(--center-channel-bg, white)',
                borderRadius: '8px',
                boxShadow: '0 4px 12px rgba(0, 0, 0, 0.3)',
                maxHeight: 'calc(100vh - 100px)',
                overflow: 'auto',
                border: '1px solid var(--center-channel-color-16, #ddd)'
            }}>
                <div style={{padding: '15px', borderBottom: '1px solid var(--center-channel-color-16, #e5e5e5)', display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
                    <h4 style={{margin: 0, fontSize: '18px', fontWeight: 'bold', color: 'var(--center-channel-color, #3f4350)'}}>
                        📧 Настройка Exchange Integration
                    </h4>
                    <button type="button" onClick={handleClose} style={{background: 'none', border: 'none', fontSize: '24px', cursor: 'pointer', color: 'var(--center-channel-color-56, #999)'}}>
                        <span>&times;</span>
                    </button>
                </div>
        
                <div style={{padding: '20px'}}>
                    <div style={{marginBottom: '15px'}}>
                        <label style={{display: 'block', marginBottom: '5px', fontWeight: 'bold', fontSize: '14px', color: 'var(--center-channel-color, #3f4350)'}}>
                            Имя пользователя <span style={{color: 'var(--error-text, red)'}}>*</span>
                        </label>
                        <input
                            type="text"
                            style={{
                                width: '100%', 
                                padding: '8px 12px', 
                                border: '1px solid var(--center-channel-color-16, #ddd)', 
                                borderRadius: '4px', 
                                fontSize: '14px', 
                                boxSizing: 'border-box',
                                backgroundColor: 'var(--center-channel-bg, white)',
                                color: 'var(--center-channel-color, #3f4350)'
                            }}
                            placeholder="Введите имя пользователя"
                            value={credentials.username}
                            onChange={(e) => handleInputChange('username', e.target.value)}
                        />
                        <div style={{fontSize: '12px', color: 'var(--center-channel-color-56, #666)', marginTop: '5px'}}>
                            Ваше имя пользователя в домене (например: ivan.petrov)
                        </div>
                    </div>

                    <div style={{marginBottom: '15px'}}>
                        <label style={{display: 'block', marginBottom: '5px', fontWeight: 'bold', fontSize: '14px', color: 'var(--center-channel-color, #3f4350)'}}>
                            Пароль <span style={{color: 'var(--error-text, red)'}}>*</span>
                        </label>
                        <input
                            type="password"
                            style={{
                                width: '100%', 
                                padding: '8px 12px', 
                                border: '1px solid var(--center-channel-color-16, #ddd)', 
                                borderRadius: '4px', 
                                fontSize: '14px', 
                                boxSizing: 'border-box',
                                backgroundColor: 'var(--center-channel-bg, white)',
                                color: 'var(--center-channel-color, #3f4350)'
                            }}
                            placeholder="Введите пароль"
                            value={credentials.password}
                            onChange={(e) => handleInputChange('password', e.target.value)}
                        />
                        <div style={{fontSize: '12px', color: 'var(--center-channel-color-56, #666)', marginTop: '5px'}}>
                            Ваш пароль для доступа к Exchange
                        </div>
                    </div>

                    <div style={{marginBottom: '15px'}}>
                        <label style={{display: 'block', marginBottom: '5px', fontWeight: 'bold', fontSize: '14px', color: 'var(--center-channel-color, #3f4350)'}}>
                            Домен
                        </label>
                        <input
                            type="text"
                            style={{
                                width: '100%', 
                                padding: '8px 12px', 
                                border: '1px solid var(--center-channel-color-16, #ddd)', 
                                borderRadius: '4px', 
                                fontSize: '14px', 
                                boxSizing: 'border-box',
                                backgroundColor: 'var(--center-channel-bg, white)',
                                color: 'var(--center-channel-color, #3f4350)'
                            }}
                            placeholder="DOMAIN (опционально)"
                            value={credentials.domain}
                            onChange={(e) => handleInputChange('domain', e.target.value)}
                        />
                        <div style={{fontSize: '12px', color: 'var(--center-channel-color-56, #666)', marginTop: '5px'}}>
                            Домен Active Directory (если требуется)
                        </div>
                    </div>

                    {testResult && (
                        <div style={{
                            padding: '12px', 
                            marginBottom: '15px',
                            borderRadius: '4px',
                            backgroundColor: testResult.success ? 'var(--online-indicator, #28a745)' : 'var(--error-text, #dc3545)',
                            border: `1px solid ${testResult.success ? 'var(--online-indicator, #28a745)' : 'var(--error-text, #dc3545)'}`,
                            color: 'white',
                            fontSize: '12px',
                            fontWeight: 'normal',
                            whiteSpace: 'pre-wrap',
                            fontFamily: 'monospace',
                            maxHeight: '300px',
                            overflow: 'auto'
                        }}>
                            {testResult.success ? '✅ ' : '❌ '}{testResult.message}
                        </div>
                    )}

                    <div style={{
                        marginBottom: '15px', 
                        padding: '10px', 
                        backgroundColor: 'var(--center-channel-color-08, #f8f9fa)', 
                        borderRadius: '4px', 
                        fontSize: '13px',
                        color: 'var(--center-channel-color-56, #666)',
                        border: '1px solid var(--center-channel-color-16, #ddd)'
                    }}>
                        <strong style={{color: 'var(--center-channel-color, #3f4350)'}}>💡 Примечание:</strong> Ваши учетные данные будут зашифрованы и надежно сохранены. 
                        После настройки плагин будет автоматически синхронизировать ваш календарь и отправлять уведомления.
                    </div>
                </div>
                
                <div style={{padding: '15px', borderTop: '1px solid var(--center-channel-color-16, #e5e5e5)', display: 'flex', justifyContent: 'flex-end', gap: '10px'}}>
                    <button
                        type="button"
                        style={{
                            padding: '8px 16px', 
                            border: '1px solid var(--center-channel-color-24, #ddd)', 
                            backgroundColor: 'var(--center-channel-color-04, #f8f9fa)', 
                            borderRadius: '4px', 
                            cursor: 'pointer', 
                            fontSize: '14px',
                            color: 'var(--center-channel-color, #3f4350)'
                        }}
                        onClick={handleClose}
                    >
                        Отмена
                    </button>
                    
                    <button
                        type="button"
                        style={{
                            padding: '8px 16px', 
                            border: '1px solid var(--center-channel-color-40, #6c757d)', 
                            backgroundColor: isTestingConnection || !credentials.username || !credentials.password ? 'var(--center-channel-color-24, #ccc)' : 'var(--center-channel-color-40, #6c757d)', 
                            color: 'white', 
                            borderRadius: '4px', 
                            cursor: isTestingConnection || !credentials.username || !credentials.password ? 'not-allowed' : 'pointer',
                            fontSize: '14px'
                        }}
                        onClick={testConnection}
                        disabled={isTestingConnection || !credentials.username || !credentials.password}
                    >
                        {isTestingConnection ? '🔄 Тестирование...' : '🔧 Тест подключения'}
                    </button>
                    
                    <button
                        type="button"
                        style={{
                            padding: '8px 16px', 
                            border: '1px solid var(--button-bg, #007bff)', 
                            backgroundColor: isSaving || !credentials.username || !credentials.password ? 'var(--center-channel-color-24, #ccc)' : 'var(--button-bg, #007bff)', 
                            color: 'white', 
                            borderRadius: '4px', 
                            cursor: isSaving || !credentials.username || !credentials.password ? 'not-allowed' : 'pointer',
                            fontSize: '14px'
                        }}
                        onClick={saveCredentials}
                        disabled={isSaving || !credentials.username || !credentials.password}
                    >
                        {isSaving ? '⏳ Сохранение...' : '💾 Сохранить'}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default ExchangeSettingsModal; 