package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/vault/api"
	"github.com/urfave/cli"
	. "git.cto.ai/sdk-go/pkg/sdk"
)

type SecretsCredentials struct {
	Token      string
	URL        string
	EngineName string
}

func getSecrets(creds SecretsCredentials, sdk *Sdk) error {

	client, e := getClient(creds)
	if e != nil {
		return e
	}

	secret, err := client.Logical().List(creds.EngineName)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if secret == nil {
		return nil
	}

	theKeys := secret.Data["keys"].([]interface{})
	retVal := make([]string, 0, len(theKeys))
	for _, y := range secret.Data["keys"].([]interface{}) {
		secretName, typeAssertionOk := y.(string)
		if !typeAssertionOk {
			return fmt.Errorf("type assertion failed in getSecrets")
		}
		retVal = append(retVal, secretName)
	}

	return sdk.Print(fmt.Sprintf("List of all secrets: %s\n", retVal))
}

func getSecretByKey(key string, creds SecretsCredentials, sdk *Sdk) error {

	client, e := getClient(creds)
	if e != nil {
		return e
	}

	secret, err := client.Logical().Read(creds.EngineName + "/" + key)
	if err != nil {
		return err
	}

	m, ok := secret.Data[key]
	if ok {
		return sdk.Print(fmt.Sprintf("Retrieved a secret: %s -> %s\n", key, m.(string)))
	} else {
		return fmt.Errorf("could not retrieve secret: %s", key)
	}
}

func putSecret(key string, value string, creds SecretsCredentials, sdk *Sdk) error {

	client, e := getClient(creds)
	if e != nil {
		return e
	}

	// TODO: some sanity checking on the keyname (eg. /^[\w-_.]+$/)
	// TODO: check if secret already exists first?
	_, err := client.Logical().Write(creds.EngineName+"/"+key, map[string]interface{}{
		key: value,
	})

	if err != nil {
		return err
	}

	return sdk.Print(fmt.Sprintln("New secret has been created"))
}

// Note that this does a HARD-delete and not a soft delete like other delete/destroy calls...
func deleteSecret(key string, creds SecretsCredentials, sdk *Sdk) error {

	client, e := getClient(creds)
	if e != nil {
		return e
	}

	//TODO: some sanity checking on the keyname (eg. /^[\w-_.]+$/)
	//TODO: check if secret already exists first?
	_, err := client.Logical().Delete(creds.EngineName + "/" + key)

	if err != nil {
		return err
	}

	return sdk.Print(fmt.Sprintf("%s has been deleted!\n", key))
}

func enableEngine(creds SecretsCredentials, sdk *Sdk) error {

	client, e := getClient(creds)
	if e != nil {
		return e
	}

	err := client.Sys().Mount(creds.EngineName, &api.MountInput{Type: "kv"})
	if err != nil {
		return err
	}

	capabilities := "\"read\", \"create\", \"update\", \"list\", \"delete\""
	policy := fmt.Sprintf("path \"%s/*\" { capabilities = [%s] }", creds.EngineName, capabilities)
	err = client.Sys().PutPolicy(creds.EngineName, policy)
	if err != nil {
		return err
	}

	request := api.TokenCreateRequest{Policies: []string{creds.EngineName}}

	secret, err := client.Auth().Token().Create(&request)

	if err != nil {
		return err
	}

	return sdk.Print(fmt.Sprintf("The vault token for the team %s: %s\n", creds.EngineName, secret.Auth.ClientToken))
}

func getClient(creds SecretsCredentials) (*api.Client, error) {

	config := &api.Config{
		Address: creds.URL,
	}
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	client.SetToken(creds.Token)

	return client, nil
}

func argsHandler(args []string, sdk *Sdk) error {

	creds := SecretsCredentials{}

	app := cli.NewApp()

	app.Name = "vault"
	app.Usage = "An OPS to work with HashiCorp Vault"
	app.Version = "0.1.0"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "url",
			Value:       "http://host.docker.internal:8200",
			Usage:       "The vault base URL",
			Destination: &creds.URL,
		},
		cli.StringFlag{
			Name:        "token",
			Value:       "",
			Usage:       "The vault token",
			Destination: &creds.Token,
		},
		cli.StringFlag{
			Name:        "team",
			Value:       "",
			Usage:       "The CTO.ai team name",
			Destination: &creds.EngineName,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "configure",
			Usage: "Configure the team structure on the vault",
			Action: func(c *cli.Context) error {
				return enableEngine(creds, sdk)
			},
		},
		{
			Name:  "secret",
			Usage: "Options for working with secrets",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "list all secrets",
					Action: func(c *cli.Context) error {
						return getSecrets(creds, sdk)
					},
				},
				{
					Name:  "add",
					Usage: "add a new secret",
					Action: func(c *cli.Context) error {
						return putSecret(c.Args().First(), c.Args().Get(1), creds, sdk)
					},
				},
				{
					Name:  "get",
					Usage: "get an existing secret",
					Action: func(c *cli.Context) error {
						return getSecretByKey(c.Args().First(), creds, sdk)
					},
				},
				{
					Name:  "remove",
					Usage: "remove an existing secret",
					Action: func(c *cli.Context) error {
						return deleteSecret(c.Args().First(), creds, sdk)
					},
				},
			},
		},
	}

	return app.Run(args)
}

func promptHandler(sdk *Sdk) error {

	creds := SecretsCredentials{}

	command, err := sdk.PromptList([]string{"Configure the team structure on the vault", "Options for working with secrets"}, "command", "Please select your command", "Configure the team structure on the vault", "c")

	if err != nil {
		return err
	}

	creds.URL, err = sdk.PromptInput("url", "The vault base URL", "http://host.docker.internal:8200", "u", false)
	if err != nil {
		return err
	}

	creds.EngineName, err = sdk.PromptInput("team", "The CTO.ai team name", "demo", "n", false)
	if err != nil {
		return err
	}

	var msg string

	if command == "Configure the team structure on the vault" {
		msg = "The vault root token"

	} else {
		msg = "The vault team token"
	}

	creds.Token, err = sdk.PromptInput("token", msg, "", "t", false)
	if err != nil {
		return err
	}

	if command == "Configure the team structure on the vault" {
		return enableEngine(creds, sdk)
	} else {
		subCommand, err := sdk.PromptList([]string{"list", "add", "get", "remove"}, "command", "Please select your command", "list", "c")

		if err != nil {
			return err
		}

		switch subCommand {
		case "list":
			return getSecrets(creds, sdk)
		case "add": {
			key, err := sdk.PromptInput("key", "Secret Name", "", "k", false)

			if err != nil {
				return err
			}

			value, err := sdk.PromptInput("value", "Secret Value", "", "v", false)

			if err != nil {
				return err
			}

			return putSecret(key, value, creds, sdk)
		}
		case "get": {
			key, err := sdk.PromptInput("key", "Secret Name", "", "k", false)

			if err != nil {
				return err
			}

			return getSecretByKey(key, creds, sdk)
		}

		case "remove": {
			key, err := sdk.PromptInput("key", "Secret Name", "", "k", false)

			if err != nil {
				return err
			}

			return deleteSecret(key, creds, sdk)
		}
		}
	}

	return nil
}

func main() {

	sdk := New()

	if len(os.Args) > 1 {
		err := argsHandler(os.Args, sdk)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err := promptHandler(sdk)
		if err != nil {
			log.Fatal(err)
		}
	}
}
