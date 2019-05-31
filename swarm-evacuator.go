package main

import (
	"github.com/spf13/cobra"
	"swarm-backup/cmd"
)

func main() {
	var cmdBackup = &cobra.Command{
		Use:   "backup [backup.json]",
		Short: "Backup current state of swarm",
		Args:  cobra.MinimumNArgs(1),
		Run:   cmd.Backup,
	}

	var cmdRestore = &cobra.Command{
		Use:   "restore [backup.json]",
		Short: "Restore swarm services from backup",
		Args:  cobra.MinimumNArgs(1),
		Run:   cmd.Restore,
	}

	var rootCmd = &cobra.Command{Use: ""}
	rootCmd.AddCommand(cmdBackup, cmdRestore)

	rootCmd.Execute()
}
