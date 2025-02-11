# News_TG_Bot
This project is a **Telegram bot for news distribution**, which automatically collects, processes, and sends news updates to a specified Telegram channel.

# THE PROJECT IS IN THE PROCESS OF DEVELOPMENT.

Key features and components include:
1. **News Collection**:
    - Utilizes the `fetcher` component to periodically fetch news from predefined sources and store them in a PostgreSQL database.

2. **Processing and Filtering**:
    - Filters news based on specific keywords.
    - Leverages machine learning (via OpenAI API) to create short textual summaries of the news.

3. **Notifications**:
    - Delivers prepared news digests to a Telegram channel using the `notifier` component and Telegram Bot API.

4. **Admin Commands**:
    - Allows adding, deleting, and viewing news sources (`addsource`, `deletesource`, `listsources`).
    - Restricts access to commands based on admin rights.

5. **Technologies**:
    - Programming Language: **Go**.
    - Data Storage: **PostgreSQL** (managed with the `sqlx` library).
    - Telegram API: **go-telegram-bot-api/v5**.
