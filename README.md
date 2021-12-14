# Jetbrains Alfred Flow

The flow is allow to open Jetbrains project.

The flow requires Toolbox

Currently supported images for the next projects:
1. Intellij Idea
2. Rider
3. GoLand
4. DataGrip

## How it works
The flow read details from the folder `~/Library/Application Support/Jebrains/*` and try to find the next files:
* recentProjects.xml
* recentSolutions.xml (Rider)

After that the flow try to find appropriate app links from the folder `~/Library/Application Support/JetBrains/Toolbox/apps/*`

## Shortcut
`SHIFT-CMD-E`

## Command
`openj`

## Examples
![Screenshot 2021-06-29 at 22 11 08](https://user-images.githubusercontent.com/3629440/123855486-a1b75d00-d928-11eb-8c66-e50cbb725b0f.png)

## Build Example
```shell
go build && alfred build
```

## Known Issues
Currently if you lauch an application (for example Intellij Idea) using alfred it rewrites environment variables. And if you lanch the project it will take the environment properties from lauched application.
