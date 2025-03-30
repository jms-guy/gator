# Gator

This is an RSS feed aggregator program. It parses rss feeds from given url's, and stores the information in a database.

## Installation

### Prerequisites
- Go (1.18 or later)
- PostgreSQL

### Option 1: Build from source
1. Clone this repository:
   ```bash
   git clone https://github.com/jms-guy/gator.git
   cd gator

2. Build executable:
    ```bash
    go build 

### Option 2: Install using Go tools
    go install github.com/jms-guy/gator@latest

## Database Setup
1. Create PostgreSQL database:
    ```bash
    -CREATE DATABASE gator;


2. Run migrations to set up database schema:
    ```bash
    gator migrate up

A config file(.gatorconfig.json) will be created in your home directory after the first user is registered.

## Commands
1. register 'user' ~~~Registers a user in the database
2. login 'user' ~~~Logs in user
3. users    ~~~Lists users in database
4. addfeed 'feed name' 'url'   ~~~Adds a feed to the database
5. feeds    ~~~Returns a list of feeds in the database
6. follow/unfollow 'url'    ~~~Logged in user can choose to follow/unfollow feeds in the database, to browse through posts
7. following    ~~~Returns a list of feeds that the currently logged in user is following
8. browse 'limit(3, 10, 15, etc.)'    ~~~Returns a list of posts for the user to browse, from feeds that they are currently following. Limit input sets the max number of posts seen at a time>
9. agg 'time(10s, 5m, 30m, 2h, etc.)'  T~~~his is the long-running aggregator service. Sends requests at a given time interval to feeds, collecting posts in database. 

## Basic Usage
 Register user. Add feeds to database. Different users can add different feeds, if a user adds a feed they are automatically following that feed, otherwise they must
manually follow it. Running the 'agg' command begins the aggregation process, fetching posts from feeds in the database. Once posts have been successfully fetched, 
they can be browsed.