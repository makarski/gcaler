gcaler
------

The name is a concatenation of `google calendar scheduler`.  
Currently supported are two subcommands  

- `plan` create events from a predefined template, i.e. calendar as code
- `list` list daily calendar events in terminal

Set up
------

This guide implies you have successfully installed and configured [golang](https://golang.org/doc/install) on your machine.

1. Create an app in google developer console
    - follow **Step 1** from the [guide](https://developers.google.com/google-apps/calendar/quickstart/go#step_1_turn_on_the_api_name)
    - copy `client_secret.json` to the project root

After the first run the google access token will cached in `$HOME/.gcaler/access_token.json`

2. Configure your app

```bash
# 1: create a config
cp templates/default.toml.dist templates/your_name.toml

# 2: modify "your_name.toml" contents with your data
```

3. Run `go install`

4. Run `gcaler` to start using the cli tool

```bash
$ gcaler -help

# provide custom templates location; defauls are in the bin working dir
$ gcaler -templates /path/to/templates -credentials /path/to/google/credentials.json {cmd}
```                     

License
-------
See the [LICENSE](LICENSE.txt) file for license rights and limitations (MIT).
