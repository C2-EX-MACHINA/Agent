# C2-EX-MACHINA Agent

Welcome to the C2 Server Agent repository! This agent is responsible for communication between equipment and the command and control server. This repository contains all the necessary source code to configure and run the agent on your equipment.

## Requirements

There is no requirements.

## Install

### Download executable

 - [Download executable from github releases](https://github.com/C2-EX-MACHINA/Agent/releases)

### Compile from source code

To configure the agent on your equipment, please follow the following steps:
Download the source code of this repository using the git clone command:

```bash
git clone https://github.com/evaris237/Agent.git
cd Agent
go build C2agent.go
```

## How the agent works

Once the agent is installed and configured, it will communicate with the C2 server to transmit the necessary information. You can monitor and control the equipment using the C2 server control panel.

## Licence

Licensed under the [GPL, version 3](https://www.gnu.org/licenses/).