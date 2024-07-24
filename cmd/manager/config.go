package main

import "github.com/kelseyhightower/envconfig"

type Config struct {
	Listen string `default:":8080"`
}

func NewConfig() Config {
	c := Config{}
	envconfig.MustProcess("mars", &c)
	return c
}
