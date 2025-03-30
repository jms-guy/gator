package main

import (
	"database/sql"
	"fmt"
	"os"
	"github.com/jms-guy/gator/internal/config"
	"github.com/jms-guy/gator/internal/database"

	_ "github.com/lib/pq"	//Imported for side effects
)

func main() {
	var s state	//Initalizes state struct
	configuration, err := config.Read()	//Sets config from file
	if err != nil {
		fmt.Println(err)
		return
	}
	s.cfg = &configuration

	dataBase, err := sql.Open("postgres", s.cfg.DbUrl)	//Sets database
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dbQueries := database.New(dataBase)
	s.db = dbQueries	//Sets queries

	commands := commands{	//Initialized commands for cli
		cmds: make(map[string]func(*state, command) error),
	}
	commands.register("login", handlerLogin)	//Login command	- logs in user
	commands.register("register", handlerRegister)	//Register command	- adds user to database
	commands.register("reset", handlerReset)	//Reset command	- clears database of all data
	commands.register("users", handlerUsers)	//Users command	- lists users in database
	commands.register("agg", handlerAgg)	//Aggregator command - handles long-running aggregator service - input a time duration
	commands.register("addfeed", middlewareLoggedIn(handlerAddFeed))	//Addfeed command - adds a feed to database
	commands.register("feeds", handlerFeeds)	//Feeds command - lists feeds in database
	commands.register("follow", middlewareLoggedIn(handlerFollow))	//Follow command - adds a follow record, for the given url feed and current user
	commands.register("following", middlewareLoggedIn(handlerFollowing))	//Following command - lists all feeds being followed by current user
	commands.register("unfollow", middlewareLoggedIn(handlerUnfollow))	//Unfollows a feed for current user
	commands.register("browse", middlewareLoggedIn(handlerBrowse))

	args := os.Args	//Gets user input arguments
	if len(args) < 2 {
		fmt.Println("Not enough arguments.")
		os.Exit(1)
	}

	cmd := command{	//Sets the input command
		name: args[1],
		args: args[2:],
	}
	
	if err := commands.run(&s, cmd); err != nil {	//Runs command
		fmt.Println(err)
		os.Exit(1)
	}
}