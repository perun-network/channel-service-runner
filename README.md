<h1 align="center"><br>
    <a href="https://perun.network/"><img src=".assets/go-perun.png" alt="Perun" width="196"></a>
<br></h1>

<h2 align="center">Perun-Channel-Service-Runner</h2>

<p align="center">
  <a href="https://www.apache.org/licenses/LICENSE-2.0.txt"><img src="https://img.shields.io/badge/license-Apache%202-blue" alt="License: Apache 2.0"></a>
</p>

This repository contains a go-service for running perun-channel-service. To run this service add your details in the config.json, compile and execute. 

## Config

`config.json` contains all the config. Modify it to suit your needs:

| Key         | Description                                              |
|-------------|----------------------------------------------------------|
| `host`      | Host address on which channel-service will run          |
| `ws_url`    | URL of your wallet                                      |
| `network`   | Choose either `"devnet"`, `"testnet"` or `"mainnet"`    |
| `public_key`| Public key of the user (wallet)                         |
| `database`  | Path for database                                       |
| `logfile`   | Path for logfile                                        |
