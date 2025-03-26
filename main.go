package main

import (
	"database/sql"
	"fmt"
	"os"
	"github.com/jms-guy/rss_aggregator/internal/config"
	"github.com/jms-guy/rss_aggregator/internal/database"

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
	commands.register("login", handlerLogin)
	commands.register("register", handlerRegister)
	commands.register("reset", handlerReset)
	commands.register("users", handlerUsers)
	commands.register("agg", handlerAgg)

	args := os.Args	//Gets user input arguments
	if len(args) < 2 {
		fmt.Println("Not enough arguments.")
		os.Exit(1)
	}

	cmd := command{
		name: args[1],
		args: args[2:],
	}
	if err := commands.run(&s, cmd); err != nil {	//Runs command
		fmt.Println(err)
		os.Exit(1)
	}
}