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

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("missing username")
	}
	if len(cmd.args) > 1 {
		return fmt.Errorf("too many arguments, expecting single username")
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
	if len(cmd.args) == 0 {
		return fmt.Errorf("missing name")
	}
	if len(cmd.args) > 1 {
		return fmt.Errorf("too many arguments, expecting single name")
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