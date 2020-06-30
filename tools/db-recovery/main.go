package main

import (
	"fmt"
	"os"

	"github.com/iotaledger/goshimmer/packages/database"
	"github.com/iotaledger/goshimmer/packages/database/prefix"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	cfgDatabaseDir = "database"
)

func init() {
	flag.String(cfgDatabaseDir, "mainnetdb", "path to the database folder")
}

func main() {

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()
	err := viper.BindPFlags(flag.CommandLine)
	if err != nil {
		return err
	}

	dbDir := viper.GetString(cfgDatabaseDir)
	ok, err := exists(dbDir)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("database folder does not exist: %s", dbDir)
	}

	db, err := database.NewDB(dbDir)
	if err != nil {
		return err
	}

	healthStore := db.NewStore().WithRealm([]byte{prefix.DBPrefixHealth})
	return healthStore.Delete([]byte("db_health"))
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, err
}
