package config

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
)

// AppConfig holds the global configuration for the project
var AppConfig *Config

// Config represents the configuration struct for the project
type Config struct {
	ShellName     string
	Prompt        string
	DisplayCdPath string
	AbsolutePath  string
	UserHomeDir   string
	Username      string
}

// InitConfig initializes the global configuration instance
func InitConfig() {
	curr, err := user.Current()
	if err != nil {
		panic(err)
	}
	AppConfig = &Config{
		ShellName:     "gosh",
		Prompt:        "$",
		DisplayCdPath: "",
		AbsolutePath:  "",
		UserHomeDir:   curr.HomeDir,
		Username:      curr.Username,
	}

	if AppConfig.Username == "root" {
		AppConfig.Prompt = "#"
	}

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	AppConfig.AbsolutePath = filepath.Dir(ex)
	if err = syscall.Chdir(AppConfig.AbsolutePath); err != nil {
		panic(err)
	}

	AppConfig.DisplayCdPath = AppConfig.AbsolutePath
	if strings.HasPrefix(AppConfig.AbsolutePath, AppConfig.UserHomeDir) && AppConfig.Username != "root" {
		AppConfig.DisplayCdPath = strings.Replace(AppConfig.DisplayCdPath, AppConfig.UserHomeDir, "~", 1)
	}

}
