package main

import (
	"fmt"
	"os"

	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/commands"
)

func main() {
	rootCmd, err := commands.InitRunCommand()
	if err != nil {
		fmt.Println("init run command: %w", err)
		os.Exit(1)
	}
	err = rootCmd.Execute()
	if err != nil {
		fmt.Println("run command: %w", err)
		os.Exit(1)
	}
}
