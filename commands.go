package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"strconv"
	"time"
	"github.com/google/uuid"
	"github.com/jms-guy/gator/internal/config"
	"github.com/jms-guy/gator/internal/database"
)

type state struct {		//State struct holding database & config information
	db *database.Queries
	cfg	*config.Config
}

type command struct {	//List of commands for cli
	name	string
	args	[]string
}

type commands struct {	//Each command in the list of commands
	cmds	map[string]func(*state, command) error
}

func handlerAgg(s *state, cmd command) error {	//Aggregator service
	if len(cmd.args) == 0 {
		return fmt.Errorf("missing time duration")
	}
	time_between_reqs := cmd.args[0]

	timeBetweenReqs, err := time.ParseDuration(time_between_reqs)
	if err != nil {
		return fmt.Errorf("error parsing duration string: %w", err)
	}
	fmt.Printf(" ~~Collecting feeds every %s~~\n", time_between_reqs)
	ticker := time.NewTicker(timeBetweenReqs)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func handlerBrowse(s *state, cmd command, user database.User) error {	//Browses posts from feeds followed by user, takes optional limit input
	var limit int32
	if len(cmd.args) == 0 {
		limit = 2
	} else {
		number, err := strconv.Atoi(cmd.args[0])
		if err != nil {
			return fmt.Errorf("number conversion error: %w", err)
		}
		limit = int32(number)
	}
	getPosts := database.GetPostsForUserParams{
		UserID: user.ID,
		Limit: limit,
	}
	posts, err := s.db.GetPostsForUser(context.Background(), getPosts)
	if err != nil {
		return fmt.Errorf("error retrieving posts: %w", err)
	}
	if len(posts) == 0 {
    	fmt.Println("No posts found. You might not be following any feeds, or the feeds don't have any posts yet.")
    	return nil 
	}
	fmt.Printf("Showing %d most recent posts from your feeds:\n\n", limit)

	for _, post := range posts {
		if post.Title.Valid {
        	fmt.Printf(" ** %s **\n", post.Title.String)
    	} else {
        	fmt.Println(" ** [No Title] **")
    	}
		fmt.Printf(" ** Published: %v\n", post.PublishedAt.Format("Jan 2, 2006 at 3:04 PM"))
		fmt.Println(" ~~~~~~~~~~")
		if post.Description.Valid {
        	fmt.Printf(" %s\n", post.Description.String)
    	} else {
        	fmt.Println(" [No Description]")
    	}
		fmt.Println(" ~~~~~~~~~~")
	}
	return nil
}

func scrapeFeeds(s *state) error {	//Grabs feeds from the feeds table, and sends fetch requests based on time since last fetched
	feedToFetch, err := s.db.GetNextFeedToFetch(context.Background())	//Grabs a feed from feeds that current user follows
	if err != nil {
		return fmt.Errorf("error getting feed to fetch: %w", err)
	}
	err = s.db.MarkFeedFetched(context.Background(), feedToFetch.ID)	//Marks it fetched at time
	if err != nil {
		return fmt.Errorf("error marking feed as fetched: %w", err)
	}

	feed, fetchErr := fetchFeed(context.Background(), feedToFetch.Url)	//Fetches contents
	if fetchErr != nil {
		return fmt.Errorf("error fetching rss feed of url: %s: %w", feedToFetch.Url, fetchErr)
	}
	fmt.Println("~~~~~~~~~~~~~~~~~~~~")
	fmt.Printf("Feed: %s\n", feed.Channel.Title)	//Prints contents
	if len(feed.Channel.Item) == 0 {
		fmt.Printf(" ~~ No posts in %s ~~\n", feed.Channel.Title)
	}
	var added, skipped int
	
	for _, item := range feed.Channel.Item {	//Iterates over posts in feed
		parsedDate, err := parseDate(item.PubDate)	//Parses publication date
		if err != nil {
			return fmt.Errorf("error parsing post's publication date: %w", err)
		}

		// For a string that might be empty or nil
		var title sql.NullString
		if item.Title != "" {
			title = sql.NullString{
				String: item.Title,
				Valid: true,
			}
		} else {
			title = sql.NullString{
				Valid: false,
			}
		}
		// Same for description
		var description sql.NullString
		if item.Description != "" {
			description = sql.NullString{
				String: item.Description,
				Valid: true,
			}
		} else {
			description = sql.NullString{
				Valid: false,
			}
		}

		newPost := database.CreatePostParams{	//Create post params
			ID: uuid.New(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			Title: title,
			Url: item.Link,
			Description: description,
			PublishedAt: parsedDate,
			FeedID: feedToFetch.ID,
		}
		_, err = s.db.CreatePost(context.Background(), newPost)	//Creates post in posts table
		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") ||
				strings.Contains(err.Error(), "duplicate key") {
				skipped++
				continue
			} else {
				return fmt.Errorf("error saving post to database: %w", err)
			}
		}
		if title.Valid {
    		fmt.Printf(" ~~ %s ~~ Saved to database\n", title.String)
		} else {
    		fmt.Println(" ~~ [No Title] ~~ Saved to database")
		}
		added++
	}
	fmt.Printf("Added %d new posts, skipped %d existing posts\n", added, skipped)

	return nil
}

func parseDate(dateStr string) (time.Time, error) {	//Parse a datetime and return it in a parsed format for easier sorting
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		time.RFC822,
		"2006-01-02T15:04:05Z",
        "2006-01-02T15:04:05-07:00",
        "2006-01-02T15:04:05+07:00",
        "2006-01-02 15:04:05",
        "Mon, 2 Jan 2006 15:04:05 MST",
        "2 Jan 2006 15:04:05 -0700",
	}
	var parsedTime time.Time
	var err error

	for _, format := range formats {
		parsedTime, err = time.Parse(format, dateStr)
		if err == nil {
			return parsedTime, nil
		}
	}
	return time.Time{}, err
}

func handlerUnfollow(s *state, cmd command, user database.User) error {	//Unfollows a feed for the current user - takes url input
	if len(cmd.args) == 0 {
		return fmt.Errorf("missing url")
	}
	url := cmd.args[0]
	feed, err := s.db.GetFeed(context.Background(), url)	//Gets feed data from feeds table
	if err != nil {
		return fmt.Errorf("error getting feed data: %w", err)
	}

	unfollowParams := database.UnfollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	unfollowErr := s.db.Unfollow(context.Background(), unfollowParams)
	if unfollowErr != nil {
		return fmt.Errorf("error unfollowing %s: %w", feed.Name, unfollowErr)
	}
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {	//Lists all feeds being followed by current user
	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)	//Get feed follows based on user
	if err != nil {
		return fmt.Errorf("error retrieving follows for %s: %w", user.Name, err)
	}
	if len(follows) == 0 {
    	fmt.Printf("%s is not following any feeds\n", user.Name)
    return nil
	}

	fmt.Printf("Feeds followed by %s:\n", user.Name)	//Returns follow data
	for _, follow := range follows {
		fmt.Println(follow.Name)
	}
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {	//Sets the current user to be following a given feed
	if len(cmd.args) == 0 {	//Checks arguments
		return fmt.Errorf("missing feed url")
	}
	url := cmd.args[0]

	feed, err := s.db.GetFeed(context.Background(), url)	//Gets feed data from feeds table
	if err != nil {
		return fmt.Errorf("error getting feed data: %w", err)
	}

	feedFollowParams := database.CreateFeedFollowParams{	//Sets follow params
		ID: uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID: user.ID,
		FeedID: feed.ID,
	}

	follow, err := s.db.CreateFeedFollow(context.Background(), feedFollowParams)	//Creates a new feed_follow in feed_follows table
	if err != nil {
		return fmt.Errorf("error following %s: %w", url, err)
	}

	fmt.Printf("Feed: %s\n", follow.FeedName)
	fmt.Printf("%s\n", user.Name)
	return nil
}

func handlerFeeds(s *state, cmd command) error {	//Returns list of feeds in database
	feeds, err := s.db.GetFeeds(context.Background())	//Gets feeds from feeds table
	if err != nil {
		return fmt.Errorf("error retrieving feeds: %w", err)
	}

	for _, feed := range feeds {
		userName, err := s.db.GetUserName(context.Background(), feed.UserID)	//Find username based on feed's user_id foreign key
		if err != nil {
			return fmt.Errorf("error getting name of user: %w", err)
		}

		fmt.Println(feed.Name)
		fmt.Println(feed.Url)
		fmt.Println(userName)
		fmt.Println("~~~~~~~~~~~~~~")
	}
	return nil
}

func handlerAddFeed(s *state, cmd command, user database.User) error {	//Creates a feed in database
	if len(cmd.args) < 2 {	//Checks arguments
		return fmt.Errorf("expected input: 'addfeed -feedname- -url-")
	}

	feedname := cmd.args[0]
	url := cmd.args[1]

	newFeed := database.CreateFeedParams{	//Set feed params
		ID: uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name: feedname,
		Url: url,
		UserID: user.ID,
	}

	feed, err := s.db.CreateFeed(context.Background(), newFeed)	//Create feed in feeds table
	if err != nil {
		return fmt.Errorf("error creating new feed: %w", err)
	}

	feedFollowParams := database.CreateFeedFollowParams{	//Sets feed follow params
		ID: uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID: user.ID,
		FeedID: feed.ID,
	}

	follow, err := s.db.CreateFeedFollow(context.Background(), feedFollowParams)	//Create a feed_follow in feed_follows table, linking created feed to current user
	if err != nil {
		return fmt.Errorf("error following %s: %w", url, err)
	}

	fmt.Printf("Feed: %s\n", follow.FeedName)
	fmt.Printf("%s\n", user.Name)
	return nil
}

func handlerUsers(s *state, cmd command) error {	//Returns list of users
	users, err := s.db.ListUsers(context.Background())	//Retrieves all user data from users table
	if err != nil {
		return fmt.Errorf("error retrieving users list: %w", err)
	}
	for _, name := range users {	//Prints data for each user
		if name != s.cfg.CurrentUserName {
			fmt.Printf("* %s\n", name)
		} else {
			fmt.Printf("* %s (current)\n", name)
		}
	}
	return nil
}

func handlerReset(s *state, cmd command) error {	//Resets database (for testing purposes)
	err := s.db.ClearDatabase(context.Background())
	if err != nil {
		return fmt.Errorf("error clearing database: %w", err)
	}
	fmt.Println("database cleared successfully.")
	return nil
}

func handlerLogin(s *state, cmd command) error {	//Logs in user (user must be registered)
	argErr := argCheck(cmd.args)	//Checks arguments
	if argErr != nil {
		return argErr
	}
	userName := cmd.args[0]

	_, err := s.db.GetUser(context.Background(), userName)	//Gets user info from users table
	if err != nil {
		fmt.Println("User does not exist in database.")
		os.Exit(1)
	}

	if err := s.cfg.SetUser(userName); err != nil {	//Sets user in config
        return err
    }
	fmt.Printf("User has been set to %s\n", userName)
	return nil
}

func handlerRegister(s *state, cmd command) error {	//Registers new user
	argErr := argCheck(cmd.args)	//Check arguments
	if argErr != nil {
		return argErr
	}

	args := database.CreateUserParams{	//Set user params
		ID: uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name: cmd.args[0],
	}
	user, err := s.db.CreateUser(context.Background(), args)	//Creates user in users table
	if err != nil {
		return fmt.Errorf("error registering user: %w", err)
	}
	s.cfg.SetUser(user.Name)	//Sets current user to registered user in config
	fmt.Println("User was created successfully.")
	fmt.Printf("Id: %v created_at: %v updated at: %v name: %s\n", user.ID, user.CreatedAt, user.UpdatedAt, user.Name)
	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {	//Closure/middleware function that handles log in functionality for necessary handler functions
	return func(s *state, cmd command) error {	//Returned inner function
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)	//Get user data from users table
		if err != nil {
			return fmt.Errorf("error retrieving user: %w", err)
		}
		return handler(s, cmd, user)	//Handler function call
	}
}

//Command functions
func (c *commands) register(name string, f func(*state, command) error) {	//Registers command in state/commands struct
	c.cmds[name] = f
}

func (c *commands) run(s *state, cmd command) error {	//Runs command in main
	command, ok := c.cmds[cmd.name]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}
	return command(s, cmd)
}

func argCheck(args []string) error {	//Error check for arg input
	if len(args) == 0 {
		return fmt.Errorf("no command input")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments given, expecting single string")
	}
	return nil
}