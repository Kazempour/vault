package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/vault/api"
	"github.com/urfave/cli"
)

type SecretsCredentials struct {
	Token      string
	URL        string
	EngineName string
}

func GetSecrets(creds SecretsCredentials, client *api.Client) ([]string, error) {

	secret, err := client.Logical().List(creds.EngineName)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	if secret == nil {
		return make([]string, 0, 0), nil
	}

	theKeys := secret.Data["keys"].([]interface{})
	retVal := make([]string, 0, len(theKeys))
	for _, y := range secret.Data["keys"].([]interface{}) {
		secretName, typeAssertionOk := y.(string)
		if !typeAssertionOk {
			return nil, fmt.Errorf("type assertion failed in GetSecrets")
		}
		retVal = append(retVal, secretName)
	}

	return retVal, nil
}

func GetSecretByKey(key string, creds SecretsCredentials, client *api.Client) (string, error) {
	secret, err := client.Logical().Read(creds.EngineName + "/" + key)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	m, ok := secret.Data[key]
	if ok {
		return m.(string), nil
	}

	return "", nil
}

func PutSecret(key string, value string, creds SecretsCredentials, client *api.Client) error {
	// TODO: some sanity checking on the keyname (eg. /^[\w-_.]+$/)
	// TODO: check if secret already exists first?
	_, err := client.Logical().Write(creds.EngineName+"/"+key, map[string]interface{}{
		key: value,
	})
	return err
}

// Note that this does a HARD-delete and not a soft delete like other delete/destroy calls...
func DeleteSecret(key string, creds SecretsCredentials, client *api.Client) error {
	//TODO: some sanity checking on the keyname (eg. /^[\w-_.]+$/)
	//TODO: check if secret already exists first?
	_, err := client.Logical().Delete(creds.EngineName + "/" + key)
	return err
}

func EnableEngine(creds SecretsCredentials, client *api.Client) (*api.Secret, error) {
	err := client.Sys().Mount(creds.EngineName, &api.MountInput{Type: "kv"})
	if err != nil {
		return nil, err
	}

	capabilities := "\"read\", \"create\", \"update\", \"list\", \"delete\""
	policy := fmt.Sprintf("path \"%s/*\" { capabilities = [%s] }", creds.EngineName, capabilities)
	err = client.Sys().PutPolicy(creds.EngineName, policy)
	if err != nil {
		return nil, err
	}

	request := api.TokenCreateRequest{Policies: []string{creds.EngineName}}
	return client.Auth().Token().Create(&request)
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

func main() {

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
				client, e := getClient(creds)
				if e != nil {
					return e
				}

				secret, e := EnableEngine(creds, client)
				if e != nil {
					return e
				}
				fmt.Printf("The vault token for the team %s: %s\n", creds.EngineName, secret.Auth.ClientToken)
				return nil
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

						client, e := getClient(creds)
						if e != nil {
							return e
						}

						// List secrets:
						listOfSecrets, theErr := GetSecrets(creds, client)
						if theErr != nil {
							return theErr
						}

						fmt.Printf("List of all secrets: %s\n", listOfSecrets)
						return nil
					},
				},
				{
					Name:  "add",
					Usage: "add a new secret",
					Action: func(c *cli.Context) error {

						client, e := getClient(creds)
						if e != nil {
							return e
						}

						// Create new secret:
						theErr := PutSecret(c.Args().First(), c.Args().Get(1), creds, client)
						if theErr != nil {
							return theErr
						}

						fmt.Println("New secret has been created")
						return nil
					},
				},
				{
					Name:  "get",
					Usage: "get an existing secret",
					Action: func(c *cli.Context) error {

						client, e := getClient(creds)
						if e != nil {
							return e
						}

						// Get a single secret:
						theSecret, theErr := GetSecretByKey(c.Args().First(), creds, client)
						if theErr != nil {
							return theErr
						}

						fmt.Printf("Retrieved a secret: %s -> %s\n", c.Args().First(), theSecret)
						return nil
					},
				},
				{
					Name:  "remove",
					Usage: "remove an existing secret",
					Action: func(c *cli.Context) error {

						client, e := getClient(creds)
						if e != nil {
							return e
						}

						// Hard delete a secret:
						theErr := DeleteSecret(c.Args().First(), creds, client)
						if theErr != nil {
							return theErr
						}

						fmt.Printf("%s has been deleted!\n", c.Args().First())
						return nil
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
