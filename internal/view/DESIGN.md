package view

# Full example

## Room view

```shell

  ● Waku: 8 peer(s)
    Room: W1Rka5Dz8GVGzgy1iq6gS5puS1x3JroeW4ZDTfBZkCfm (dealer)

  Issue:  https://github.com/golang/go/issues/19412
          proposal: spec: add sum types / discriminated unions
          [LanguageChange] [v2] [Proposal] [NeedsInvestigation]
                                                    
  ╭───────┬─────┬─────────┬────────┬─────────╮ ╭──────╮                                       
  │ Alice │ Bob │ Charlie │ Didukh │ Sirotin │ │ HINT │                       
  ├───────┼─────┼─────────┼────────┼─────────┤ ├──────┤
  │   1   │  3  │    8    │   13   │    3    │ │   3  │                                         
  ╰───────┴─────┴─────────┴────────┴─────────╯ ╰──────╯    
  Advice: start discussion between Alice and Didukh                                                      
              ╭───╮                                                       
  ╭───╮ ╭───╮ │ 3 │ ╭───╮ ╭───╮ ╭────╮ ╭────╮ ╭────╮                                 
  │ 1 │ │ 2 │ ╰───╯ │ 5 │ │ 8 │ │ 13 │ │ 21 │ │ 34 │                                    
  ╰───╯ ╰───╯       ╰───╯ ╰───╯ ╰────╯ ╰────╯ ╰────╯                                    
                ^
                
 Use [←] and [→] arrows to select a card and press [Enter]
 [Tab] To switch to issues list view
 [C] Switch to command mode  [Q] Leave room  [E] Exit  [H] Help

```

## Issues list view

```shell

  Room:   238mvM91TbC9UJdUqKwtWFb

  Issues:  
    13  https://github.com/six78/2-story-points-cli/issues/1                                
  > -   https://github.com/golang/go/issues/19412                                                          
    -   https://github.com/protocolbuffers/protobuf/issues/2224                                
    -   https://github.com/vim/vim/issues/2034                                
    ... 7 more issues     
        
 Use [↓] and [↑] arrows to select issue to deal and press [Enter]
 [Tab] To switch to room view
 [C] Switch to command mode  [Q] Leave room  [E] Exit  [H] Help

```
# Components

## Header info

```shell
  Log:	  file:////~/Library/Application%20Support/six78/2sp/logs/waku-pp-2024-03-28T09:01:16Z.log
  Room:   238mvM91TbC9UJdUqKwtWFb
  
  Issue:  https://github.com/six78/2-story-points-cli/issues/42
  Title:  Implement room encryption
```

### Log file path

Log filepath is only shown in `--debug` mode. \
What else you want to know?

### Room

Room shows the RoomID

### Issue

Note that an empty string is kept even when `Title` field is empty. This is to prevent UI from jumping.

2.1. When `Issue.TitleOrURL` is empty:
```shell
   Issue:  <spinner> Waiting for dealer
   
```

2.1. When `Issue.TitleOrURL` is not a URL or provider is not supported:
```shell
   Issue:  <Issue.TitleOrURL>
   
```

2.1. When `Issue.TitleOrURL` is a URL
```shell
   Issue:  <Issue.TitleOrURL>
   Title:  <spinner> Fetching  
```
```shell
   Issue:  <Issue.TitleOrURL>
   Title:  <Fetched issue title>
```

## Players

```shell
  ╭───────┬─────┬─────────┬────────┬─────────╮
  │ Alice │ Bob │ Charlie │ Didukh │ Sirotin │
  ├───────┼─────┼─────────┼────────┼─────────┤
  │   X   │  ✓  │    8    │   13   │<spinner>│
  ╰───────┴─────┴─────────┴────────┴─────────╯
```

### Players

Players are rendered as a table. Vote can be one of these:
- `<spinner>`: voting in progress, player not voted yet
- `✓`: voting in progress, player voted,
- `X`: votes revealed, player didn't vote
- `<value>`: votes revealed, player voted with <value>

### Result 

Next to players table there's a single-column result table.
When voting is in progress, result table should be half-transparent. This is not to disturb the player.

The result value follows these rules:
- `<empty>`: voting in progress
- `<spinner>`: votes revealed, result not published yet
- `<value>`: votes revealed, result is <value>

## Deck

Deck is rendered as a series of single-cell tables. Each cell is a card from a deck.\
Switching between cards is done with `←` and `→` keys.

There are 2 modifiers of card UI - selected and vote.\
Therefore the card can be in one of these 4 states:

<table>
<tr>
<th>Modifiers</th>
<td> - </td>
<td>Selected</td>
<td>Vote</td>
<td>Vote & Selected</td>
</tr>

<tr>
<th>Description</th>
<td>No active modifiers</td>
<td>Card selected with cursor</td>
<td>Card published as vote</td>
<td>Card published as vote and selected with cursor</td>
</tr>

<tr>
<th>View</th>
<td>

```shell

╭───╮
│ 4 │
╰───╯

```
</td>
<td>

```shell

╭───╮
│ 4 │
╰───╯
  ^
```
</td>
<td>

```shell
╭───╮
│ 4 │
╰───╯


```
</td>
<td>

```shell
╭───╮
│ 4 │
╰───╯

  ^
```
</td>
</tr>
</table>

For example for a Fibbonacci deck  

```shell
  Your vote:
              ╭───╮
  ╭───╮ ╭───╮ │ 3 │ ╭───╮ ╭───╮ ╭────╮ ╭────╮ ╭────╮
  │ 1 │ │ 2 │ ╰───╯ │ 5 │ │ 8 │ │ 13 │ │ 21 │ │ 34 │
  ╰───╯ ╰───╯       ╰───╯ ╰───╯ ╰────╯ ╰────╯ ╰────╯
                      ^ 
```

## Actions

### Command mode 

```shell

┃ Type a command...
<last-command-error>
```

### Player interactive mode

```shell
 To vote se [←] and [→] arrows to select a card and press [Enter]
 [Tab] To switch to issues list view
 [C] Switch to manual mode  [Q] Leave room  [E] Exit
<last-command-error>
```

### Dealer interactive mode

```shell
 To vote se [←] and [→] arrows to select a card and press [Enter]
 [Tab] To switch to issues list view
 [R] Reveal  [F] Finish and deal next issue  [A] Add issue
 [C] Switch to manual mode  [Q] Leave room  [E] Exit
<last-command-error>
```

## Issues list view

```shell
  13  https://github.com/six78/2-story-points-cli/issues/42
  3   https://github.com/six78/2-story-points-cli/issues/19
> -   https://github.com/six78/2-story-points-cli/issues/12
  -   https://github.com/six78/2-story-points-cli/issues/1 
  -   https://github.com/six78/2-story-points-cli/issues/34
  ... 7 more issues
```