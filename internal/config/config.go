package config

import (
	"os"
)

// Config holds all action inputs.
type Config struct {
	SearchPaths      string
	ExcludeDirs      string
	ExcludeFileTypes string
	FileTypes        string
	Depth            string
	Reporter         string
	GroupBy          string
	Quiet            string
	Globbing         string
	RequireSchema    string
	NoSchema         string
	SchemaStore      string
	SchemaStorePath  string
	TypeMap          string
	SchemaMap        string
	Gitignore        string
	IgnoreFiles      string
	OnlyChanged      string
}

// Load reads configuration from os.Args positional arguments.
func Load() Config {
	return Config{
		SearchPaths:      os.Args[1],
		ExcludeDirs:      os.Args[2],
		ExcludeFileTypes: os.Args[3],
		FileTypes:        os.Args[4],
		Depth:            os.Args[5],
		Reporter:         os.Args[6],
		GroupBy:          os.Args[7],
		Quiet:            os.Args[8],
		Globbing:         os.Args[9],
		RequireSchema:    os.Args[10],
		NoSchema:         os.Args[11],
		SchemaStore:      os.Args[12],
		SchemaStorePath:  os.Args[13],
		TypeMap:          os.Args[14],
		SchemaMap:        os.Args[15],
		Gitignore:        os.Args[16],
		IgnoreFiles:      os.Args[17],
		OnlyChanged:      os.Args[18],
	}
}
