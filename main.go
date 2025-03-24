package main

import (
	"database/sql"
	"fmt"
	"os"
	"github.com/jms-guy/rss_aggregator/internal/config"
	"github.com/jms-guy/rss_aggregator/internal/database"

	_ "github.com/lib/pq"
)

func main() {
	var s state
	configuration, err := config.Read()
	if err != nil {
		fmt.Println(err)
		return
	}
	s.cfg = &configuration
	dataBase, err := sql.Open("postgres", s.cfg.DbUrl)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	dbQueries := database.New(dataBase)
	s.db = dbQueries

	commands := commands{
		cmds: make(map[string]func(*state, command) error),
	}
	commands.register("login", handlerLogin)
	commands.register("register", handlerRegister)

	args := os.Args
	if len(args) < 2 {
		fmt.Println("Not enough arguments.")
		os.Exit(1)
	}

	cmd := command{
		name: args[1],
		args: args[2:],
	}
	if err := commands.run(&s, cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}