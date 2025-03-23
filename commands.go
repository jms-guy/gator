package main

import (
	"fmt"
	"github.com/jms-guy/rss_aggregator/internal/config"
)

type state struct {
	configFile	*config.Config
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

	if err := s.configFile.SetUser(userName); err != nil {
        return err
    }
	fmt.Printf("User has been set to %s\n", userName)
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