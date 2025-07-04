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
            state.plugins?.plugins?.['com.mattermost.exchange-plugin']?.isSettingsModalOpen,
            state['plugins/com.mattermost.exchange-plugin']?.isSettingsModalOpen,
            state.plugins?.['com.mattermost.exchange-plugin']?.isSettingsModalOpen,
            state.exchangePlugin?.isSettingsModalOpen
        ];
        
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

            const result = await response.json();
            setTestResult(result);
        } catch (error) {
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

    return (
        <div className={`modal fade ${forceShow ? 'show' : ''}`} style={{display: forceShow ? 'block' : 'none', zIndex: 9999}}>
            <div className="modal-dialog modal-lg">
                <div className="modal-content">
                    <div className="modal-header">
                        <h4 className="modal-title">
                            📧 Настройка Exchange Integration
                        </h4>
                        <button type="button" className="close" onClick={handleClose}>
                            <span>&times;</span>
                        </button>
                    </div>
            
                    <div className="modal-body">
                <div className="form-group">
                    <label className="control-label">
                        Имя пользователя <span className="error-text">*</span>
                    </label>
                    <input
                        type="text"
                        className="form-control"
                        placeholder="Введите имя пользователя"
                        value={credentials.username}
                        onChange={(e) => handleInputChange('username', e.target.value)}
                    />
                    <div className="help-text">
                        Ваше имя пользователя в домене (например: ivan.petrov)
                    </div>
                </div>

                <div className="form-group">
                    <label className="control-label">
                        Пароль <span className="error-text">*</span>
                    </label>
                    <input
                        type="password"
                        className="form-control"
                        placeholder="Введите пароль"
                        value={credentials.password}
                        onChange={(e) => handleInputChange('password', e.target.value)}
                    />
                    <div className="help-text">
                        Ваш пароль для доступа к Exchange
                    </div>
                </div>

                <div className="form-group">
                    <label className="control-label">
                        Домен
                    </label>
                    <input
                        type="text"
                        className="form-control"
                        placeholder="DOMAIN (опционально)"
                        value={credentials.domain}
                        onChange={(e) => handleInputChange('domain', e.target.value)}
                    />
                    <div className="help-text">
                        Домен Active Directory (если требуется)
                    </div>
                </div>

                {testResult && (
                    <div className={`alert ${testResult.success ? 'alert-success' : 'alert-danger'}`}>
                        {testResult.message}
                    </div>
                )}

                <div className="form-group">
                    <div className="help-text">
                        <strong>Примечание:</strong> Ваши учетные данные будут зашифрованы и надежно сохранены. 
                        После настройки плагин будет автоматически синхронизировать ваш календарь и отправлять уведомления.
                    </div>
                </div>
                    </div>
                    
                    <div className="modal-footer">
                        <button
                            type="button"
                            className="btn btn-default"
                            onClick={handleClose}
                        >
                            Отмена
                        </button>
                        
                        <button
                            type="button"
                            className="btn btn-secondary"
                            onClick={testConnection}
                            disabled={isTestingConnection || !credentials.username || !credentials.password}
                        >
                            {isTestingConnection ? 'Тестирование...' : 'Тест подключения'}
                        </button>
                        
                        <button
                            type="button"
                            className="btn btn-primary"
                            onClick={saveCredentials}
                            disabled={isSaving || !credentials.username || !credentials.password}
                        >
                            {isSaving ? 'Сохранение...' : 'Сохранить'}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default ExchangeSettingsModal; 