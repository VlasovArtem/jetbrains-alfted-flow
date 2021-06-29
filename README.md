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

## Examples

## Build Example
```shell
go build && alfred build
```
