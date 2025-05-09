# Discord Trivia Bot

A Discord bot for running trivia games. Built in Go, it uses SQLite for data storage and runs in a Docker container on a Raspberry Pi (ARM64), although you can edit the Dockerfile or run the Go app on its own as you please. The bot supports individual and team play, with a clean embed-based interface and persistent score tracking.

## Features

- Trivia Games: Start games with `!!trivia start`, answer questions with `!!trivia answer`, and add custom questions with `!!trivia addq`.
- Leaderboard: `!!trivia scores` displays players and teams sorted by score in descending order (highest to lowest).
- Teams: Create and join teams with `!!trivia join`. Team names are case-insensitive (e.g., TeamA, teama, TEAMA are treated as the same).
- Admin Controls: Restricted commands for admins (via ID or role) to manage questions and games.
- Embeds: Rich Discord embeds for questions.
- Persistence: SQLite database (trivia.db) persists questions and scores across container rebuilds using a bind mount.
- Channel allow-list: Only allows commands in specified channels (e.g., trivia, games) to prevent spam in other channels.

## Prerequisites

- Raspberry Pi with Docker installed (ARM64 architecture).
- Discord Bot Token: Create a bot on the [Discord Developer Portal](https://discord.com/developers/applications) and obtain a token.
- Git: For cloning the repository.
- SQLite: For database management (included in the Docker image).

## Setup

### 1. Clone the Repository

Run: `git clone https://github.com/airylvat/trivia-bot.git`
Then: `cd trivia-bot`

### 2. Configure Environment

Create a .env file in the project root with the following contents:

```env
DISCORD_TOKEN=
ADMIN_ID=
ADMIN_ROLE_ID=
ALLOWED_CHANNELS=
```
- Replace `DISCORD_TOKEN` with your bot token.
- Set `ADMIN_ID` to your Discord user ID (for admin commands).
- Set `ADMIN_ROLE_ID` to the role ID of the admin role (for admin commands).
- Set `ALLOWED_CHANNELS` to a comma-separated list of channel IDs where the bot can respond to commands (e.g., `123456789012345678,234567890123456789`).

### 3. Set Up the Database

The bot uses a SQLite database (trivia.db) stored in a data directory, persisted via a Docker bind mount.

Run the following commands:

```bash
mkdir -p ~/trivia-bot/data
cp ~/trivia-bot/trivia.db ~/trivia-bot/data/
chmod 664 ~/trivia-bot/data/trivia.db
chown youruser:yourgroup ~/trivia-bot/data/trivia.db
```

- Ensure trivia.db (with 100 Bible questions) is in the project root before copying.
- If creating a new database, initialize it with the schema in db/db.go.

### 4. Run the Bot with Docker

The `manage-bot.sh` script builds and runs the bot in a Docker container.

Run: `chmod +x manage-bot.sh`
Then: `./manage-bot.sh`

This:
- Builds the Docker image (trivia-bot).
- Runs the container with a bind mount (`~/trivia-bot/data` to `/app/data`).
- Verifies the database and shows logs.

### 5. Verify the Bot

Check the database:
Run: `sqlite3 ~/trivia-bot/data/trivia.db "SELECT COUNT(*) FROM questions;"`
- Should show however many questions you've added. You'll want to come back and check this after you add some questions and restart the bot.
- In Discord, use `!!trivia list` to see available questions.

## Usage

### Commands

- `!!trivia start`: Start a trivia game.
- `!!trivia answer <your_answer>`: Answer the current question (first correct answer scores points).
- `!!trivia next`: Get the next question.
- `!!trivia addq <question> | <answer>`: Add a new question (admin only).
- `!!trivia scores`: Show the leaderboard with players and teams sorted by score (highest to lowest).
- `!!trivia addteam <team_name>`: Create a team (case-insensitive, e.g., TeamA, teama).
- `!!trivia jointeam <team_name>`: Join a team (case-insensitive).
- `!!trivia list`: List how many questions are in the database.
- `!!trivia list questions`: Write out all the questions, without answers.
- `!!trivia list answers`: Write out all the questions and their answers.

### Example

1. Start a game:
- Run in Discord: `!trivia start`
- Then: `!!trivia next`

2. Add a team and join:
Run in Discord: !!trivia addteam BibleScholars
Then: !!trivia jointeam biblescholars

3. Answer a question (first correct answer scores points).
- Run in Discord: `!!trivia answer <your_answer>`

4. Check scores:
- Run in Discord: `!!trivia scores`

## Updating the Database

To update trivia.db with a new version:

1. Backup the current database:
Run: `cp ~/trivia-bot/data/trivia.db ~/trivia-bot/trivia.db.bak`

2. Copy the new database:
Run: `cp ~/trivia-bot/new-trivia.db /home/localpie/bots/trivia-bot/data/trivia.db`
Then: `chmod 664 ~/trivia-bot/data/trivia.db`
Then: `chown localpie:localpie ~/trivia-bot/data/trivia.db`

3. Restart the bot:
Run: `./manage-bot.sh`

## Development

### Contributing

1. Fork the repository.
2. Create a feature branch.
5. Open a pull request.

### Building Locally

To build without Docker:
Run: `go build -o trivia-bot main.go`
Then: `./trivia-bot`

