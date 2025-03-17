package main

import (
	"fmt"
	"github.com/jms-guy/rss_aggregator/internal/config"
)

func main() {
	configuration, err := config.Read()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(configuration)

	if err := configuration.SetUser("guy"); err != nil {
		fmt.Println(err)
		return
	}

	newConfig, err := config.Read()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("DbUrl: %s\nCurrentUserName: %s\n", newConfig.DbUrl, newConfig.CurrentUserName)
}