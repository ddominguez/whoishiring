# Who is hiring
This is my personal `Hacker News - Who is hiring?` application.

This app allows me to read all job posts for the current month and helps me keep track of posts that I have alread seen.

![screenshot](./screenshots/screen.png)

# Info
This app uses the [Hacker News API](https://github.com/HackerNews/API) to fetch job posts from
the current `Who is hiring?` thread and saves them locally to an SQLite database.

## Dependencies
* [goose](https://pressly.github.io/goose/) - for sql migrations
* [sqlx](https://github.com/jmoiron/sqlx) - for db queries in go

