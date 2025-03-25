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

type state struct {
	db *database.Queries
	cfg	*config.Config
}

type command struct {
	name	string
	args	[]string
}

type commands struct {
	cmds	map[string]func(*state, command) error
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.ListUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error retrieving users list: %w", err)
	}
	for _, name := range users {
		if name != s.cfg.CurrentUserName {
			fmt.Printf("* %s\n", name)
		} else {
			fmt.Printf("* %s (current)\n", name)
		}
	}
	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.ClearDatabase(context.Background())
	if err != nil {
		return fmt.Errorf("error clearing database: %w", err)
	}
	fmt.Println("database cleared successfully.")
	return nil
}

func handlerLogin(s *state, cmd command) error {
	argErr := argCheck(cmd.args)
	if argErr != nil {
		return argErr
	}
	userName := cmd.args[0]

	_, err := s.db.GetUser(context.Background(), userName)
	if err != nil {
		fmt.Println("User does not exist in database.")
		os.Exit(1)
	}

	if err := s.cfg.SetUser(userName); err != nil {
        return err
    }
	fmt.Printf("User has been set to %s\n", userName)
	return nil
}

func handlerRegister(s *state, cmd command) error {
	argErr := argCheck(cmd.args)
	if argErr != nil {
		return argErr
	}

	args := database.CreateUserParams{
		ID: uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name: cmd.args[0],
	}
	user, err := s.db.CreateUser(context.Background(), args)
	if err != nil {
		return fmt.Errorf("error registering user: %w", err)
	}
	s.cfg.SetUser(user.Name)
	fmt.Println("User was created successfully.")
	fmt.Printf("Id: %v created_at: %v updated at: %v name: %s\n", user.ID, user.CreatedAt, user.UpdatedAt, user.Name)
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.cmds[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	command, ok := c.cmds[cmd.name]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}
	return command(s, cmd)
}

func argCheck(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command input")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments given, expecting single string")
	}
	return nil
}