# Colgen – Collection Generator for Go

[![Go Report Card](https://goreportcard.com/badge/github.com/vmkteam/colgen)](https://goreportcard.com/report/github.com/vmkteam/colgen)

Colgen is a powerful Go code generation tool that automates the creation of collection methods and types based on special comment annotations.

## Features

- Generates collection types and utility methods from simple annotations
- Supports multiple generation modes:
    - Base collection types (`[]Struct`)
    - Field value collectors
    - Unique field value collectors
    - Converters and Index functions
- AI-assisted review/readme/tests generation
- Code injection capabilities
- Customizable through command-line flags

## Installation

```sh
go install github.com/vmkteam/colgen/cmd/colgen@latest
```

## Usage

### Comment Format

```go
//go:generate colgen
//colgen:News,Category,Tag
//colgen:News:TagIDs,UniqueTagIDs,Map(db),UUID
//colgen:Episode:ShowIDs,MapP(db.SiteUser),Index(MovieID)
//colgen:Show:MapP(db)
//colgen:Season:mapp(db)
```

### Command Line Flags

| Flag         | Description                           | Default |
|--------------|---------------------------------------|---------|
| `-list`      | Use "List" suffix for collections     | false   |
| `-imports`   | Custom import paths (comma-separated) | ""      |
| `-funcpkg`   | Package for Map & MapP functions      | ""      |
| `-write-key` | Write assistant key to homedir        | ""      |

## Generation Modes

### Base Generators

For `//colgen:<struct>,<struct>,...`:
- Creates `type <struct>s []<struct>`
- Generates methods:
    - `IDs() []<id type>` - Returns all IDs in slice
    - `Index() map[<id type>]<struct>` - Returns map of ID to struct

### Custom Generators

- `Index(field)` - Create index by specified field (default: ID)
- `<Field>` - Collect all values from field
- `Unique<Field>` - Collect unique values from field
- `MapP` - Generate mapping function with package prefix
- `Map` - Generate mapping function (can be lowercase for private)

### Inline Mode

```go
//colgen@NewCall(db)
//colgen@newUserSummary(newsportal.User,full,json)
```
#### AI Assistance

[//]: # (TODO: claude)

`colgen -write-key=<deepseek key>`

```go
//go:generate colgen 
//colgen@ai:<readme|review|tests>
```
