# Релизы Mattermost Exchange Integration Plugin

В этой папке находятся готовые к установке версии плагина.

## Доступные версии

### v1.0.0 (2025-07-03)
**Файл:** `com.mattermost.exchange-plugin-1.0.0.tar.gz` (21MB)

**Основные возможности:**
- 🔄 Автоматическая синхронизация статуса пользователей с календарем Exchange
- ⏰ Система напоминаний о встречах с интерактивными уведомлениями
- 📧 Уведомления о новых приглашениях на встречи
- 💬 Slash-команды для управления (/exchange setup/status/calendar/reminders/help)
- 🌐 Веб-интерфейс для настройки учетных данных Exchange
- 📅 Ежедневная утренняя сводка встреч

**Технические характеристики:**
- Поддержка Mattermost Server v6.0+
- Интеграция с Microsoft Exchange Server 2013+
- Exchange Web Services (EWS) протокол
- Размер сборки: 21MB
- Качество кода: ✅ Все тесты пройдены

## Как установить

1. **Скачайте** нужную версию из этой папки
2. **Откройте** Mattermost Admin Console
3. **Перейдите** в System Console → Plugins → Plugin Management
4. **Нажмите** "Choose File" и выберите скачанный `.tar.gz` файл
5. **Нажмите** "Upload" для загрузки плагина
6. **Нажмите** "Enable" для активации плагина

## Системные требования

- **Mattermost Server:** v6.0 или выше
- **Exchange Server:** 2013 или выше с поддержкой EWS
- **Доступ:** К Exchange Web Services (обычно `/EWS/Exchange.asmx`)

## Конфигурация

После установки настройте плагин в:
**System Console → Plugins → Exchange Integration**

Обязательные параметры:
- URL Exchange сервера (например: `https://mail.company.com`)
- Включить синхронизацию календаря
- Настроить время напоминаний (по умолчанию: 15 минут)

## Поддержка

Вопросы и сообщения о проблемах: [GitHub Issues](https://github.com/chastnik/mattermost_plugin_exchange_firstbit/issues) 