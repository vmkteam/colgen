# Colgen – Collection Generator for Go

[![Go Report Card](https://goreportcard.com/badge/github.com/vmkteam/colgen)](https://goreportcard.com/report/github.com/vmkteam/colgen)

Colgen generates collection types and utility methods for Go structs from comment annotations. Eliminates boilerplate: extracting IDs, indexing, grouping, hydrating relations, converting between layers.

## Installation

```sh
go install github.com/vmkteam/colgen/cmd/colgen@latest
```

## Quick Start

Given a struct:

```go
type News struct {
    ID       int
    StatusID int
    TagIDs   []int
    AuthorID *int
    Author   *Author
    Tags     []Tag
}
```

Add annotations to `collection.go`:

```go
//go:generate colgen
//colgen:News
//colgen:News:StatusID,TagIDs,UniqueAuthorID,Group(TagIDs),Count(StatusID),Fill(Tags,TagIDs),Fill(Author,AuthorID)
```

Run `go generate` and get:

```go
type NewsList []News

func (ll NewsList) IDs() []int              { /* ... */ }
func (ll NewsList) Index() map[int]News     { /* ... */ }
func (ll NewsList) StatusIDs() []int        { /* r[i] = ll[i].StatusID */ }
func (ll NewsList) TagIDs() []int           { /* flatten: append(r, ll[i].TagIDs...) */ }
func (ll NewsList) UniqueAuthorIDs() []int  { /* nil-check + dedup */ }
func (ll NewsList) GroupByTagIDs() map[int]NewsList  { /* fan-out */ }
func (ll NewsList) CountByStatusID(v int) int        { /* ... */ }
func (ll NewsList) FillTags(related Tags) NewsList    { /* many-to-many */ }
func (ll NewsList) FillAuthor(related Authors) NewsList { /* one-to-one */ }
```

## Annotation Format

```go
//go:generate colgen [flags]
//colgen:Struct1,Struct2              // base: collection type + IDs() + Index()
//colgen:Struct1:Field,Group(Name)    // custom generators
```

### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-list` | Use "List" suffix instead of "s" | false |
| `-imports` | Custom import paths (comma-separated) | "" |
| `-funcpkg` | Package for Map/MapP helper functions | "" |

## Generators

### Base

`//colgen:Review` generates:
- `type Reviews []Review`
- `IDs() []<id_type>` — all IDs
- `Index() map[<id_type>]Review` — index by ID

Requires an `ID` field in the struct.

### Custom

| Directive | Generated method | Notes |
|-----------|-----------------|-------|
| `<Field>` | `Fields() []T` | scalar: direct assign, `*T`: nil-check + deref, `[]T`: flatten |
| `Unique<Field>` | `UniqueFields() []T` | dedup via map; supports scalar, `[]T`, `*T` |
| `Index(Field)` | `IndexByField() map[T]Struct` | |
| `Group(Field)` | `GroupByField() map[T]List` | `[]T` field: fan-out (element appears in multiple groups) |
| `Count(Field)` | `CountByField(v T) int` | count elements matching value |
| `Fill(Target,FK)` | `FillTarget(related List) List` | `[]int` FK: many-to-many, `*int` FK: one-to-one |
| `Cast(Iface)` | `Ifaces() []Iface` | struct must implement interface via pointer receiver |
| `MapP(pkg)` | `NewStructs([]pkg.Struct) Structs` | converter, expects `NewStruct(*pkg.T) *T` |
| `Map(pkg)` | `NewStructs([]pkg.Struct) Structs` | converter, expects `NewStruct(pkg.T) T` |
| `mapp(pkg)` | `newStructs([]pkg.Struct) []Struct` | private converter |
| `map(pkg)` | `newStructs([]pkg.Struct) []Struct` | private converter |

Combine with commas: `//colgen:Issue:MapP(db),Group(ReviewFileID),Count(StatusID)`

## Inline Mode

Generate constructors directly in the source file:

```go
//colgen@NewCall(db)
//colgen@newUserSummary(newsportal.User,full,json)
```

## AI Assistance

```go
//colgen@ai:readme            // generate README (deepseek by default)
//colgen@ai:tests(deepseek)   // generate tests
//colgen@ai:review(claude)    // code review
```

Setup: `colgen -write-key=<key>` (deepseek) or `colgen -write-key=<key> -ai=claude`.
