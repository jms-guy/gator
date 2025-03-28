package main

import (
	"context"
	"fmt"
	"time"
	"os"
	"github.com/google/uuid"
	"github.com/jms-guy/rss_aggregator/internal/config"
	"github.com/jms-guy/rss_aggregator/internal/database"
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
	url := "https://www.wagslane.dev/index.xml"

	feed, err := fetchFeed(context.Background(), url)
	if err != nil {
		return fmt.Errorf("error fetching rss feed of url: %s", url)
	}
	fmt.Println(feed)

	return nil
}

func handlerFollowing(s *state, cmd command) error {	//Lists all feeds being followed by current user
	currentUser := s.cfg.CurrentUserName

	user, err := s.db.GetUser(context.Background(), currentUser)	//Gets user data from users table
	if err != nil {
		return fmt.Errorf("error retrieving user from database: %w", err)
	}

	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)	//Get feed follows based on user
	if err != nil {
		return fmt.Errorf("error retrieving follows for %s: %w", currentUser, err)
	}
	if len(follows) == 0 {
    	fmt.Printf("%s is not following any feeds\n", currentUser)
    return nil
	}

	fmt.Printf("Feeds followed by %s:\n", currentUser)	//Returns follow data
	for _, follow := range follows {
		fmt.Println(follow.Name)
	}
	return nil
}

func handlerFollow(s *state, cmd command) error {	//Sets the current user to be following a given feed
	if len(cmd.args) == 0 {	//Checks arguments
		return fmt.Errorf("missing feed url")
	}
	url := cmd.args[0]
	currentUser := s.cfg.CurrentUserName

	user, err := s.db.GetUser(context.Background(), currentUser)	//Gets current user data from users table
	if err != nil {
		return fmt.Errorf("error retrieving user from database: %w", err)
	}

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
	fmt.Printf("%s\n", currentUser)
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

func handlerAddFeed(s *state, cmd command) error {	//Creates a feed in database
	if len(cmd.args) < 2 {	//Checks arguments
		return fmt.Errorf("expected input: 'addfeed -feedname- -url-")
	}

	currentUser := s.cfg.CurrentUserName
	feedname := cmd.args[0]
	url := cmd.args[1]

	user, err := s.db.GetUser(context.Background(), currentUser)	//Gets user data from users table - need user's ID
	if err != nil {
		return fmt.Errorf("error retrieving user from database: %w", err)
	}

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
	fmt.Printf("%s\n", currentUser)
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