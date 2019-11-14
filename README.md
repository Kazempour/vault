# vault
An OPS to configure and work with HashiCorp Vault

### For running on local 
1. Download vault binary and run it in your local env (https://www.vaultproject.io/downloads.html)
2. run `./vault server -dev`
3. Retrieve `ROOT_TOKEN` from the logs
4. `ops run vault --url "http://host.docker.internal:8200" --token [ROOT_TOKEN] --team [YOUR_TEAM_NAME] configure` this will setup the team structure in the vault and will return a token associated with your team
5. You can use the returned token from previous step to register your vault with CTO.ai CLI 
6. You can also use this op to work with created team in the vault:  
e.g. `ops run vault --url "http://host.docker.internal:8200" --token [TEAM_TOKEN] --team [YOUR_TEAM_NAME] secret list`